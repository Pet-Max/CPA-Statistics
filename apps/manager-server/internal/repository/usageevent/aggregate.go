package usageevent

import (
	"context"
	"database/sql"
)

// Aggregate captures roll-up metrics for a usage_events window.
type Aggregate struct {
	TotalCalls          int64
	SuccessCalls        int64
	FailureCalls        int64
	InputTokens         int64
	OutputTokens        int64
	ReasoningTokens     int64
	CachedTokens        int64
	CacheReadTokens     int64
	CacheCreationTokens int64
	TotalTokens         int64
	AvgLatencyMS        sql.NullFloat64
	ZeroTokenCalls      int64
}

// ModelStat aggregates per-model totals.
type ModelStat struct {
	Model               string
	BillingModel        string
	Calls               int64
	SuccessCalls        int64
	InputTokens         int64
	OutputTokens        int64
	ReasoningTokens     int64
	CachedTokens        int64
	CacheReadTokens     int64
	CacheCreationTokens int64
	TotalTokens         int64
}

// RecentFailure holds the columns required to display a recent failure entry.
type RecentFailure struct {
	TimestampMS           int64
	Model                 string
	APIKeyHash            string
	Source                string
	SourceHash            string
	AuthIndex             string
	Endpoint              string
	LatencyMS             sql.NullInt64
	AccountSnapshot       string
	AuthLabelSnapshot     string
	AuthProviderSnapshot  string
	AuthProjectIDSnapshot string
	FailStatusCode        sql.NullInt64
	FailSummary           string
}

const aggregateSQL = `select
	count(*),
	sum(case when failed = 0 then 1 else 0 end),
	sum(case when failed = 1 then 1 else 0 end),
	coalesce(sum(input_tokens), 0),
	coalesce(sum(output_tokens), 0),
	coalesce(sum(reasoning_tokens), 0),
	coalesce(sum(max(max(cached_tokens, cache_tokens) - max(cache_read_tokens, 0) - max(cache_creation_tokens, 0), 0)), 0),
	coalesce(sum(cache_read_tokens), 0),
	coalesce(sum(cache_creation_tokens), 0),
	coalesce(sum(total_tokens), 0),
	avg(nullif(latency_ms, 0)),
	coalesce(sum(case when total_tokens = 0 and failed = 0 then 1 else 0 end), 0)
from usage_events
where timestamp_ms >= ? and timestamp_ms < ?`

// AggregateBetween computes summary metrics over [fromMs, toMs).
func (r *repository) AggregateBetween(ctx context.Context, fromMs, toMs int64) (Aggregate, error) {
	row := r.db.QueryRowContext(ctx, aggregateSQL, fromMs, toMs)
	var agg Aggregate
	var success, failure sql.NullInt64
	if err := row.Scan(
		&agg.TotalCalls,
		&success,
		&failure,
		&agg.InputTokens,
		&agg.OutputTokens,
		&agg.ReasoningTokens,
		&agg.CachedTokens,
		&agg.CacheReadTokens,
		&agg.CacheCreationTokens,
		&agg.TotalTokens,
		&agg.AvgLatencyMS,
		&agg.ZeroTokenCalls,
	); err != nil {
		return Aggregate{}, err
	}
	agg.SuccessCalls = success.Int64
	agg.FailureCalls = failure.Int64
	return agg, nil
}

const topModelsSQL = `with top_models as (
	select
		model,
		count(*) as model_calls
	from usage_events
	where timestamp_ms >= ? and timestamp_ms < ?
	group by model
	order by model_calls desc
	limit ?
)
select
	e.model,
	coalesce(nullif(e.resolved_model, ''), e.model) as billing_model,
	count(*) as calls,
	sum(case when e.failed = 0 then 1 else 0 end) as success,
	coalesce(sum(e.input_tokens), 0),
	coalesce(sum(e.output_tokens), 0),
	coalesce(sum(e.reasoning_tokens), 0),
	coalesce(sum(max(max(e.cached_tokens, e.cache_tokens) - max(e.cache_read_tokens, 0) - max(e.cache_creation_tokens, 0), 0)), 0),
	coalesce(sum(e.cache_read_tokens), 0),
	coalesce(sum(e.cache_creation_tokens), 0),
	coalesce(sum(e.total_tokens), 0)
from usage_events e
join top_models t on t.model = e.model
where e.timestamp_ms >= ? and e.timestamp_ms < ?
group by e.model, billing_model
order by max(t.model_calls) desc, e.model, calls desc`

// TopModelsBetween returns the most active models ordered by call count.
func (r *repository) TopModelsBetween(ctx context.Context, fromMs, toMs int64, limit int) ([]ModelStat, error) {
	if limit <= 0 {
		limit = 5
	}
	rows, err := r.db.QueryContext(ctx, topModelsSQL, fromMs, toMs, limit, fromMs, toMs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make([]ModelStat, 0, limit)
	for rows.Next() {
		var stat ModelStat
		if err := rows.Scan(
			&stat.Model,
			&stat.BillingModel,
			&stat.Calls,
			&stat.SuccessCalls,
			&stat.InputTokens,
			&stat.OutputTokens,
			&stat.ReasoningTokens,
			&stat.CachedTokens,
			&stat.CacheReadTokens,
			&stat.CacheCreationTokens,
			&stat.TotalTokens,
		); err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}
	return stats, rows.Err()
}

const modelStatsSQL = `select
	model,
	coalesce(nullif(resolved_model, ''), model) as billing_model,
	count(*) as calls,
	sum(case when failed = 0 then 1 else 0 end) as success,
	coalesce(sum(input_tokens), 0),
	coalesce(sum(output_tokens), 0),
	coalesce(sum(reasoning_tokens), 0),
	coalesce(sum(max(max(cached_tokens, cache_tokens) - max(cache_read_tokens, 0) - max(cache_creation_tokens, 0), 0)), 0),
	coalesce(sum(cache_read_tokens), 0),
	coalesce(sum(cache_creation_tokens), 0),
	coalesce(sum(total_tokens), 0)
from usage_events
where timestamp_ms >= ? and timestamp_ms < ?
group by model, billing_model
order by calls desc`

// ModelStatsBetween returns per-model totals for all models in a window.
func (r *repository) ModelStatsBetween(ctx context.Context, fromMs, toMs int64) ([]ModelStat, error) {
	rows, err := r.db.QueryContext(ctx, modelStatsSQL, fromMs, toMs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make([]ModelStat, 0)
	for rows.Next() {
		var stat ModelStat
		if err := rows.Scan(
			&stat.Model,
			&stat.BillingModel,
			&stat.Calls,
			&stat.SuccessCalls,
			&stat.InputTokens,
			&stat.OutputTokens,
			&stat.ReasoningTokens,
			&stat.CachedTokens,
			&stat.CacheReadTokens,
			&stat.CacheCreationTokens,
			&stat.TotalTokens,
		); err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}
	return stats, rows.Err()
}

const recentFailuresSQL = `select
	timestamp_ms, model,
	coalesce(api_key_hash, ''),
	coalesce(source, ''),
	coalesce(source_hash, ''),
	coalesce(auth_index, ''),
	coalesce(endpoint, ''),
	latency_ms,
	coalesce(account_snapshot, ''),
	coalesce(auth_label_snapshot, ''),
	coalesce(auth_provider_snapshot, ''),
	coalesce(auth_project_id_snapshot, ''),
	fail_status_code,
	coalesce(fail_summary, '')
from usage_events
where failed = 1 and timestamp_ms >= ? and timestamp_ms < ?
order by timestamp_ms desc, id desc
limit ?`

// RecentFailuresBetween returns the most recent failed events.
func (r *repository) RecentFailuresBetween(ctx context.Context, fromMs, toMs int64, limit int) ([]RecentFailure, error) {
	if limit <= 0 {
		limit = 5
	}
	rows, err := r.db.QueryContext(ctx, recentFailuresSQL, fromMs, toMs, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]RecentFailure, 0, limit)
	for rows.Next() {
		var rf RecentFailure
		if err := rows.Scan(
			&rf.TimestampMS,
			&rf.Model,
			&rf.APIKeyHash,
			&rf.Source,
			&rf.SourceHash,
			&rf.AuthIndex,
			&rf.Endpoint,
			&rf.LatencyMS,
			&rf.AccountSnapshot,
			&rf.AuthLabelSnapshot,
			&rf.AuthProviderSnapshot,
			&rf.AuthProjectIDSnapshot,
			&rf.FailStatusCode,
			&rf.FailSummary,
		); err != nil {
			return nil, err
		}
		results = append(results, rf)
	}
	return results, rows.Err()
}

// HourlyTimelineBetween returns hourly buckets relative to fromMs over [fromMs, toMs).
func (r *repository) HourlyTimelineBetween(ctx context.Context, fromMs, toMs int64) ([]TimelinePoint, error) {
	return r.BucketTimelineBetween(ctx, fromMs, toMs, 3600000)
}

// BucketTimelineBetween returns buckets relative to fromMs over [fromMs, toMs).
func (r *repository) BucketTimelineBetween(ctx context.Context, fromMs, toMs int64, bucketMs int64) ([]TimelinePoint, error) {
	if bucketMs <= 0 {
		bucketMs = 3600000
	}
	rows, err := r.db.QueryContext(ctx, `select
	cast((timestamp_ms - ?) / ? as integer) as bucket_index,
	count(*),
	coalesce(sum(total_tokens), 0),
	sum(case when failed = 0 then 1 else 0 end),
	sum(case when failed = 1 then 1 else 0 end)
from usage_events
where timestamp_ms >= ? and timestamp_ms < ?
group by bucket_index
order by bucket_index`, fromMs, bucketMs, fromMs, toMs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	points := make([]TimelinePoint, 0)
	for rows.Next() {
		var bucketIndex int64
		var point TimelinePoint
		if err := rows.Scan(&bucketIndex, &point.Calls, &point.Tokens, &point.Success, &point.Failure); err != nil {
			return nil, err
		}
		if bucketIndex < 0 {
			continue
		}
		point.BucketMS = fromMs + bucketIndex*bucketMs
		points = append(points, point)
	}
	return points, rows.Err()
}
