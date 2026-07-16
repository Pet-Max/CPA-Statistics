package usageevent

import (
	"context"

	"github.com/seakee/cpa-statistics/apps/manager-server/internal/usage"
)

// BackfillCacheAccounting adds canonical billable input values to records
// imported before v1.0.3. It is idempotent: a populated cache_input_mode marks
// an event as completed, and records are processed in bounded transactions.
func (r *repository) BackfillCacheAccounting(ctx context.Context, batchSize int) (int, error) {
	if batchSize <= 0 {
		batchSize = 500
	}
	total := 0
	for {
		rows, err := r.db.QueryContext(ctx, `select
			id, coalesce(provider, ''), coalesce(executor_type, ''), model,
			coalesce(requested_model, ''), coalesce(resolved_model, ''), coalesce(auth_provider_snapshot, ''),
			coalesce(raw_json, ''), input_tokens, cached_tokens, cache_tokens, cache_read_tokens, cache_creation_tokens
			from usage_events
			where coalesce(cache_input_mode, '') = ''
			order by id
			limit ?`, batchSize)
		if err != nil {
			return total, err
		}
		type item struct {
			id                                                                    int64
			provider, executor, model, requested, resolved, providerSnapshot, raw string
			input, cached, cache, cacheRead, cacheCreation                        int64
		}
		items := make([]item, 0, batchSize)
		for rows.Next() {
			var value item
			if err := rows.Scan(&value.id, &value.provider, &value.executor, &value.model, &value.requested, &value.resolved, &value.providerSnapshot, &value.raw, &value.input, &value.cached, &value.cache, &value.cacheRead, &value.cacheCreation); err != nil {
				_ = rows.Close()
				return total, err
			}
			items = append(items, value)
		}
		if err := rows.Close(); err != nil {
			return total, err
		}
		if len(items) == 0 {
			return total, nil
		}

		tx, err := r.db.BeginTx(ctx, nil)
		if err != nil {
			return total, err
		}
		stmt, err := tx.PrepareContext(ctx, `update usage_events
			set cache_input_mode = ?, billable_input_tokens = ?, normalized_total_input_tokens = ?
			where id = ? and coalesce(cache_input_mode, '') = ''`)
		if err != nil {
			_ = tx.Rollback()
			return total, err
		}
		for _, value := range items {
			accounting := usage.NormalizeCacheAccounting(usage.CacheInputContext{
				ExplicitMode:     usage.CacheInputModeFromRawJSON(value.raw),
				ExecutorType:     value.executor,
				Provider:         value.provider,
				ProviderSnapshot: value.providerSnapshot,
				ResolvedModel:    value.resolved,
				RequestedModel:   value.requested,
				DisplayModel:     value.model,
			}, value.input, value.cached, value.cache, value.cacheRead, value.cacheCreation)
			if _, err := stmt.ExecContext(ctx, accounting.Mode, accounting.UncachedInputTokens, accounting.TotalInputTokens, value.id); err != nil {
				_ = stmt.Close()
				_ = tx.Rollback()
				return total, err
			}
		}
		if err := stmt.Close(); err != nil {
			_ = tx.Rollback()
			return total, err
		}
		if err := tx.Commit(); err != nil {
			return total, err
		}
		total += len(items)
	}
}
