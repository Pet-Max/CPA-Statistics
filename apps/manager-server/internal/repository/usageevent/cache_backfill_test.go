package usageevent

import (
	"context"
	"path/filepath"
	"testing"

	sqliterepo "github.com/seakee/cpa-statistics/apps/manager-server/internal/repository/sqlite"
	"github.com/seakee/cpa-statistics/apps/manager-server/internal/usage"
)

func TestBackfillCacheAccountingIsIdempotent(t *testing.T) {
	db, err := sqliterepo.Open(filepath.Join(t.TempDir(), "usage.sqlite"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	repo := New(db)
	event := usage.Event{
		EventHash:           "cache-backfill",
		TimestampMS:         1,
		Timestamp:           "2026-07-16T00:00:00Z",
		Provider:            "openai",
		Model:               "gpt-test",
		InputTokens:         1_000,
		CachedTokens:        400,
		CacheReadTokens:     200,
		CacheCreationTokens: 100,
		TotalTokens:         1_000,
		CreatedAtMS:         1,
	}
	if _, err := repo.InsertBatch(context.Background(), []usage.Event{event}); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if _, err := db.Exec(`update usage_events set cache_input_mode = '', billable_input_tokens = 0, normalized_total_input_tokens = 0`); err != nil {
		t.Fatalf("reset legacy fields: %v", err)
	}

	updated, err := repo.BackfillCacheAccounting(context.Background(), 1)
	if err != nil || updated != 1 {
		t.Fatalf("backfill = (%d, %v)", updated, err)
	}
	updated, err = repo.BackfillCacheAccounting(context.Background(), 1)
	if err != nil || updated != 0 {
		t.Fatalf("idempotent backfill = (%d, %v)", updated, err)
	}
	events, err := repo.ListRecent(context.Background(), 10)
	if err != nil || len(events) != 1 {
		t.Fatalf("events = %#v, %v", events, err)
	}
	if events[0].CacheInputMode != usage.CacheInputModeIncluded || events[0].BillableInputTokens != 600 || events[0].NormalizedTotalInputTokens != 1_000 {
		t.Fatalf("backfilled event = %#v", events[0])
	}
}
