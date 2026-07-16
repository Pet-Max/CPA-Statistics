package usageevent

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/seakee/cpa-statistics/apps/manager-server/internal/model"
	sqliterepo "github.com/seakee/cpa-statistics/apps/manager-server/internal/repository/sqlite"
	"github.com/seakee/cpa-statistics/apps/manager-server/internal/usage"
)

func TestModelPriceMigrationCandidatesRequireUniqueResolvedModel(t *testing.T) {
	db, err := sqliterepo.Open(filepath.Join(t.TempDir(), "usage.sqlite"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	repo := New(db)
	events := []usage.Event{
		{EventHash: "one", TimestampMS: 1, Timestamp: "2026-07-16T00:00:00Z", Model: "alias-one", RequestedModel: "alias-one", ResolvedModel: "gpt-real", CreatedAtMS: 1},
		{EventHash: "two", TimestampMS: 2, Timestamp: "2026-07-16T00:00:01Z", Model: "alias-many", RequestedModel: "alias-many", ResolvedModel: "gpt-a", CreatedAtMS: 2},
		{EventHash: "three", TimestampMS: 3, Timestamp: "2026-07-16T00:00:02Z", Model: "alias-many", RequestedModel: "alias-many", ResolvedModel: "gpt-b", CreatedAtMS: 3},
	}
	if _, err := repo.InsertBatch(context.Background(), events); err != nil {
		t.Fatalf("insert: %v", err)
	}
	sets, err := repo.ModelPriceMigrationCandidates(context.Background(), map[string]model.ModelPrice{
		"alias-one":  {Prompt: 1, Completion: 2, Cache: 0.5, Source: "manual"},
		"alias-many": {Prompt: 3, Completion: 4, Cache: 1, Source: "manual"},
	})
	if err != nil {
		t.Fatalf("candidates: %v", err)
	}
	if len(sets) != 1 || sets[0].Model != "gpt-real" || len(sets[0].Candidates) != 1 {
		t.Fatalf("sets = %#v", sets)
	}
	candidate := sets[0].Candidates[0]
	if candidate.SourceModelID != "alias-one" || candidate.Reason != "legacy_alias_price" || candidate.Price.Source != "migration" {
		t.Fatalf("candidate = %#v", candidate)
	}
}
