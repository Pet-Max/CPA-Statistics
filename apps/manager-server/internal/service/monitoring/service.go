package monitoring

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/seakee/cpa-statistics/apps/manager-server/internal/service/pricing"
	"github.com/seakee/cpa-statistics/apps/manager-server/internal/store"
)

const (
	defaultEventsLimit      = 100
	maxEventsLimit          = 50000
	recentWindowMS          = 30 * 60 * 1000
	timelineHourMS          = 60 * 60 * 1000
	timelineDayMS           = 24 * 60 * 60 * 1000
	maxFilledTimelinePoints = 366
)

type Service struct {
	store *store.Store
}

func New(store *store.Store) *Service {
	return &Service{store: store}
}

type Request struct {
	FromMS           int64   `json:"from_ms"`
	ToMS             int64   `json:"to_ms"`
	NowMS            int64   `json:"now_ms"`
	SearchQuery      string  `json:"search_query"`
	SearchAPIKeyHash string  `json:"search_api_key_hash"`
	Filters          Filters `json:"filters"`
	Include          Include `json:"include"`
}

type Filters struct {
	Models            []string `json:"models"`
	Providers         []string `json:"providers"`
	Accounts          []string `json:"accounts"`
	AuthIndices       []string `json:"auth_indices"`
	APIKeyHashes      []string `json:"api_key_hashes"`
	SourceHashes      []string `json:"source_hashes"`
	IncludeFailed     *bool    `json:"include_failed"`
	FailedOnly        bool     `json:"failed_only"`
	ExcludeZeroTokens bool     `json:"exclude_zero_token"`
}

type Include struct {
	Summary            bool        `json:"summary"`
	Timeline           bool        `json:"timeline"`
	HourlyDistribution bool        `json:"hourly_distribution"`
	ModelShare         bool        `json:"model_share"`
	ChannelShare       bool        `json:"channel_share"`
	ModelStats         bool        `json:"model_stats"`
	FailureSources     bool        `json:"failure_sources"`
	AccountStats       bool        `json:"account_stats"`
	APIKeyStats        bool        `json:"api_key_stats"`
	FilterOptions      bool        `json:"filter_options"`
	TaskBuckets        bool        `json:"task_buckets"`
	RecentFailures     int         `json:"recent_failures"`
	EventsPage         *EventsPage `json:"events_page"`
	Granularity        string      `json:"granularity"`
}

type EventsPage struct {
	Limit    int    `json:"limit"`
	BeforeMS *int64 `json:"before_ms"`
}

type Response struct {
	GeneratedAtMS      int64              `json:"generated_at_ms"`
	Granularity        string             `json:"granularity"`
	Summary            *Summary           `json:"summary,omitempty"`
	Timeline           []TimelinePoint    `json:"timeline,omitempty"`
	HourlyDistribution []HourlyPoint      `json:"hourly_distribution,omitempty"`
	ModelShare         []ModelShareRow    `json:"model_share,omitempty"`
	ModelStats         []ModelStat        `json:"model_stats,omitempty"`
	ChannelShare       []ChannelShareRow  `json:"channel_share,omitempty"`
	FailureSources     []FailureSourceRow `json:"failure_sources,omitempty"`
	AccountStats       []AccountStatRow   `json:"account_stats,omitempty"`
	APIKeyStats        []APIKeyStatRow    `json:"api_key_stats,omitempty"`
	FilterOptions      *FilterOptions     `json:"filter_options,omitempty"`
	TaskBuckets        []TaskBucketRow    `json:"task_buckets,omitempty"`
	RecentFailures     []RecentFailure    `json:"recent_failures,omitempty"`
	Events             *EventsResponse    `json:"events,omitempty"`
}

type Summary struct {
	TotalCalls            int64    `json:"total_calls"`
	SuccessCalls          int64    `json:"success_calls"`
	FailureCalls          int64    `json:"failure_calls"`
	SuccessRate           float64  `json:"success_rate"`
	InputTokens           int64    `json:"input_tokens"`
	OutputTokens          int64    `json:"output_tokens"`
	CachedTokens          int64    `json:"cached_tokens"`
	CacheReadTokens       int64    `json:"cache_read_tokens"`
	CacheCreationTokens   int64    `json:"cache_creation_tokens"`
	ReasoningTokens       int64    `json:"reasoning_tokens"`
	TotalTokens           int64    `json:"total_tokens"`
	TotalCost             float64  `json:"total_cost"`
	AverageLatencyMS      *float64 `json:"average_latency_ms"`
	AverageTTFTMS         *float64 `json:"average_ttft_ms"`
	ZeroTokenCalls        int64    `json:"zero_token_calls"`
	RPM30M                float64  `json:"rpm_30m"`
	TPM30M                float64  `json:"tpm_30m"`
	AvgDailyRequests      float64  `json:"avg_daily_requests"`
	AvgDailyTokens        float64  `json:"avg_daily_tokens"`
	ApproxTasks           int64    `json:"approx_tasks"`
	ApproxTaskFailures    int64    `json:"approx_task_failures"`
	ApproxTaskSuccessRate float64  `json:"approx_task_success_rate"`
	ZeroTokenModels       []string `json:"zero_token_models"`
}

type TimelinePoint struct {
	BucketMS int64  `json:"bucket_ms"`
	Label    string `json:"label"`
	Calls    int64  `json:"calls"`
	Tokens   int64  `json:"tokens"`
	Success  int64  `json:"success"`
	Failure  int64  `json:"failure"`
}

type HourlyPoint struct {
	Hour   int   `json:"hour"`
	Calls  int64 `json:"calls"`
	Tokens int64 `json:"tokens"`
}

type ModelShareRow struct {
	Model  string  `json:"model"`
	Calls  int64   `json:"calls"`
	Tokens int64   `json:"tokens"`
	Cost   float64 `json:"cost"`
}

type ModelStat struct {
	Model               string  `json:"model"`
	Calls               int64   `json:"calls"`
	SuccessCalls        int64   `json:"success_calls"`
	FailureCalls        int64   `json:"failure_calls"`
	SuccessRate         float64 `json:"success_rate"`
	InputTokens         int64   `json:"input_tokens"`
	OutputTokens        int64   `json:"output_tokens"`
	CachedTokens        int64   `json:"cached_tokens"`
	CacheReadTokens     int64   `json:"cache_read_tokens"`
	CacheCreationTokens int64   `json:"cache_creation_tokens"`
	TotalTokens         int64   `json:"total_tokens"`
	Cost                float64 `json:"cost"`
}

type ChannelShareRow struct {
	AuthIndex            string   `json:"auth_index"`
	Source               string   `json:"source,omitempty"`
	AccountSnapshot      string   `json:"account_snapshot,omitempty"`
	AuthLabelSnapshot    string   `json:"auth_label_snapshot,omitempty"`
	AuthProviderSnapshot string   `json:"auth_provider_snapshot,omitempty"`
	Calls                int64    `json:"calls"`
	Success              int64    `json:"success"`
	Failure              int64    `json:"failure"`
	Tokens               int64    `json:"tokens"`
	Cost                 float64  `json:"cost"`
	AvgLatencyMS         *float64 `json:"average_latency_ms"`
}

type FailureSourceRow struct {
	Source               string   `json:"source,omitempty"`
	SourceHash           string   `json:"source_hash"`
	AuthIndex            string   `json:"auth_index"`
	AccountSnapshot      string   `json:"account_snapshot,omitempty"`
	AuthLabelSnapshot    string   `json:"auth_label_snapshot,omitempty"`
	AuthProviderSnapshot string   `json:"auth_provider_snapshot,omitempty"`
	Calls                int64    `json:"calls"`
	Failure              int64    `json:"failure"`
	LastSeenMS           int64    `json:"last_seen_ms"`
	AvgLatencyMS         *float64 `json:"average_latency_ms"`
}

type AccountStatRow struct {
	ID                   string                `json:"id"`
	AccountSnapshot      string                `json:"account_snapshot,omitempty"`
	AuthLabelSnapshot    string                `json:"auth_label_snapshot,omitempty"`
	AuthProviderSnapshot string                `json:"auth_provider_snapshot,omitempty"`
	AuthIndices          []string              `json:"auth_indices,omitempty"`
	Sources              []string              `json:"sources,omitempty"`
	SourceHashes         []string              `json:"source_hashes,omitempty"`
	Calls                int64                 `json:"calls"`
	SuccessCalls         int64                 `json:"success_calls"`
	FailureCalls         int64                 `json:"failure_calls"`
	SuccessRate          float64               `json:"success_rate"`
	InputTokens          int64                 `json:"input_tokens"`
	OutputTokens         int64                 `json:"output_tokens"`
	CachedTokens         int64                 `json:"cached_tokens"`
	CacheReadTokens      int64                 `json:"cache_read_tokens"`
	CacheCreationTokens  int64                 `json:"cache_creation_tokens"`
	TotalTokens          int64                 `json:"total_tokens"`
	Cost                 float64               `json:"cost"`
	AvgLatencyMS         *float64              `json:"average_latency_ms"`
	LastSeenMS           int64                 `json:"last_seen_ms"`
	Models               []AccountModelStatRow `json:"models,omitempty"`
}

type AccountModelStatRow struct {
	Model               string  `json:"model"`
	Calls               int64   `json:"calls"`
	SuccessCalls        int64   `json:"success_calls"`
	FailureCalls        int64   `json:"failure_calls"`
	SuccessRate         float64 `json:"success_rate"`
	InputTokens         int64   `json:"input_tokens"`
	OutputTokens        int64   `json:"output_tokens"`
	CachedTokens        int64   `json:"cached_tokens"`
	CacheReadTokens     int64   `json:"cache_read_tokens"`
	CacheCreationTokens int64   `json:"cache_creation_tokens"`
	TotalTokens         int64   `json:"total_tokens"`
	Cost                float64 `json:"cost"`
	LastSeenMS          int64   `json:"last_seen_ms"`
}

type APIKeyStatRow struct {
	ID                   string                `json:"id"`
	APIKeyHash           string                `json:"api_key_hash"`
	AccountSnapshot      string                `json:"account_snapshot,omitempty"`
	AuthLabelSnapshot    string                `json:"auth_label_snapshot,omitempty"`
	AuthProviderSnapshot string                `json:"auth_provider_snapshot,omitempty"`
	AuthIndices          []string              `json:"auth_indices,omitempty"`
	Sources              []string              `json:"sources,omitempty"`
	SourceHashes         []string              `json:"source_hashes,omitempty"`
	Calls                int64                 `json:"calls"`
	SuccessCalls         int64                 `json:"success_calls"`
	FailureCalls         int64                 `json:"failure_calls"`
	SuccessRate          float64               `json:"success_rate"`
	InputTokens          int64                 `json:"input_tokens"`
	OutputTokens         int64                 `json:"output_tokens"`
	CachedTokens         int64                 `json:"cached_tokens"`
	CacheReadTokens      int64                 `json:"cache_read_tokens"`
	CacheCreationTokens  int64                 `json:"cache_creation_tokens"`
	TotalTokens          int64                 `json:"total_tokens"`
	Cost                 float64               `json:"cost"`
	AvgLatencyMS         *float64              `json:"average_latency_ms"`
	LastSeenMS           int64                 `json:"last_seen_ms"`
	Models               []AccountModelStatRow `json:"models,omitempty"`
}

type FilterOptions struct {
	AccountStats []AccountStatRow  `json:"account_stats,omitempty"`
	APIKeyStats  []APIKeyStatRow   `json:"api_key_stats,omitempty"`
	ChannelShare []ChannelShareRow `json:"channel_share,omitempty"`
	ModelStats   []ModelStat       `json:"model_stats,omitempty"`
}

type TaskBucketRow struct {
	BucketKey           string   `json:"bucket_key"`
	Total               int64    `json:"total"`
	Success             int64    `json:"success"`
	Failure             int64    `json:"failure"`
	FirstMS             int64    `json:"first_ms"`
	LastMS              int64    `json:"last_ms"`
	Source              string   `json:"source"`
	SourceHash          string   `json:"source_hash"`
	AuthIndex           string   `json:"auth_index"`
	Models              []string `json:"models"`
	Endpoints           []string `json:"endpoints"`
	InputTokens         int64    `json:"input_tokens"`
	OutputTokens        int64    `json:"output_tokens"`
	CachedTokens        int64    `json:"cached_tokens"`
	CacheReadTokens     int64    `json:"cache_read_tokens"`
	CacheCreationTokens int64    `json:"cache_creation_tokens"`
	TotalTokens         int64    `json:"total_tokens"`
	AvgLatencyMS        *float64 `json:"average_latency_ms"`
	MaxLatencyMS        *int64   `json:"max_latency_ms"`
}

type RecentFailure struct {
	TimestampMS           int64  `json:"timestamp_ms"`
	Model                 string `json:"model"`
	APIKeyHash            string `json:"api_key_hash"`
	Source                string `json:"source,omitempty"`
	SourceHash            string `json:"source_hash"`
	AuthIndex             string `json:"auth_index"`
	AccountSnapshot       string `json:"account_snapshot,omitempty"`
	AuthLabelSnapshot     string `json:"auth_label_snapshot,omitempty"`
	AuthProviderSnapshot  string `json:"auth_provider_snapshot,omitempty"`
	AuthProjectIDSnapshot string `json:"auth_project_id_snapshot,omitempty"`
	Endpoint              string `json:"endpoint"`
	DurationMS            *int64 `json:"duration_ms"`
	FailStatusCode        *int64 `json:"fail_status_code,omitempty"`
	FailSummary           string `json:"fail_summary,omitempty"`
}

type EventsResponse struct {
	Items        []EventRow `json:"items"`
	NextBeforeMS int64      `json:"next_before_ms"`
	HasMore      bool       `json:"has_more"`
}

type EventRow struct {
	EventHash             string `json:"event_hash"`
	TimestampMS           int64  `json:"timestamp_ms"`
	Model                 string `json:"model"`
	ResolvedModel         string `json:"resolved_model,omitempty"`
	Endpoint              string `json:"endpoint"`
	Method                string `json:"method"`
	Path                  string `json:"path"`
	AuthIndex             string `json:"auth_index"`
	Source                string `json:"source"`
	SourceHash            string `json:"source_hash"`
	APIKeyHash            string `json:"api_key_hash"`
	AccountSnapshot       string `json:"account_snapshot"`
	AuthLabelSnapshot     string `json:"auth_label_snapshot"`
	AuthProviderSnapshot  string `json:"auth_provider_snapshot"`
	AuthProjectIDSnapshot string `json:"auth_project_id_snapshot,omitempty"`
	ReasoningEffort       string `json:"reasoning_effort,omitempty"`
	ServiceTier           string `json:"service_tier,omitempty"`
	ExecutorType          string `json:"executor_type,omitempty"`
	InputTokens           int64  `json:"input_tokens"`
	OutputTokens          int64  `json:"output_tokens"`
	CachedTokens          int64  `json:"cached_tokens"`
	CacheReadTokens       int64  `json:"cache_read_tokens"`
	CacheCreationTokens   int64  `json:"cache_creation_tokens"`
	ReasoningTokens       int64  `json:"reasoning_tokens"`
	TotalTokens           int64  `json:"total_tokens"`
	LatencyMS             *int64 `json:"latency_ms"`
	TTFTMS                *int64 `json:"ttft_ms"`
	Failed                bool   `json:"failed"`
	FailStatusCode        *int64 `json:"fail_status_code,omitempty"`
	FailSummary           string `json:"fail_summary,omitempty"`
}

func (s *Service) Analytics(ctx context.Context, req Request) (Response, error) {
	if req.FromMS <= 0 || req.ToMS <= 0 || req.FromMS >= req.ToMS {
		return Response{}, errors.New("from_ms and to_ms are required and from_ms must be less than to_ms")
	}
	nowMS := req.NowMS
	if nowMS <= 0 {
		nowMS = time.Now().UnixMilli()
	}
	granularity := normalizeGranularity(req.Include.Granularity, req.FromMS, req.ToMS)
	filter := buildFilter(req)
	prices, err := s.store.LoadModelPrices(ctx)
	if err != nil {
		return Response{}, err
	}

	response := Response{
		GeneratedAtMS: time.Now().UnixMilli(),
		Granularity:   granularity,
	}

	var modelStats []store.ModelStat
	needsModelStats := req.Include.Summary || req.Include.ModelShare || req.Include.ModelStats
	if needsModelStats {
		modelStats, err = s.store.ModelStatsWithFilter(ctx, filter, 0)
		if err != nil {
			return Response{}, err
		}
	}

	var taskBuckets []store.TaskBucket
	if req.Include.Summary || req.Include.TaskBuckets {
		taskBuckets, err = s.store.TaskBucketsWithFilter(ctx, filter)
		if err != nil {
			return Response{}, err
		}
	}

	if req.Include.Summary {
		agg, err := s.store.AggregateWithFilter(ctx, filter)
		if err != nil {
			return Response{}, err
		}
		rollingFilter := filter
		rollingFilter.FromMS = nowMS - recentWindowMS
		rollingFilter.ToMS = nowMS
		rollingAgg, err := s.store.AggregateWithFilter(ctx, rollingFilter)
		if err != nil {
			return Response{}, err
		}
		activeDays, err := s.store.ActiveDaysWithFilter(ctx, filter)
		if err != nil {
			return Response{}, err
		}
		zeroTokenModels, err := s.store.ZeroTokenModelsWithFilter(ctx, filter)
		if err != nil {
			return Response{}, err
		}
		response.Summary = buildSummary(agg, rollingAgg, activeDays, modelStats, taskBuckets, prices, zeroTokenModels)
	}
	if req.Include.Timeline {
		points, err := s.store.TimelineWithFilter(ctx, filter, granularity)
		if err != nil {
			return Response{}, err
		}
		response.Timeline = buildTimeline(points, granularity, req.FromMS, req.ToMS)
	}
	if req.Include.HourlyDistribution {
		points, err := s.store.HourlyDistributionWithFilter(ctx, filter)
		if err != nil {
			return Response{}, err
		}
		response.HourlyDistribution = buildHourly(points)
	}
	if req.Include.ModelShare {
		response.ModelShare = buildModelShare(modelStats, prices)
	}
	if req.Include.ModelStats {
		response.ModelStats = buildModelStats(modelStats, prices)
	}
	if req.Include.ChannelShare {
		stats, err := s.store.ChannelModelStatsWithFilter(ctx, filter)
		if err != nil {
			return Response{}, err
		}
		response.ChannelShare = buildChannelShare(stats, prices)
	}
	if req.Include.FailureSources {
		stats, err := s.store.FailureSourcesWithFilter(ctx, filter)
		if err != nil {
			return Response{}, err
		}
		response.FailureSources = buildFailureSources(stats)
	}
	if req.Include.AccountStats {
		stats, err := s.store.AccountModelStatsWithFilter(ctx, filter)
		if err != nil {
			return Response{}, err
		}
		response.AccountStats = buildAccountStats(stats, prices)
	}
	if req.Include.APIKeyStats {
		stats, err := s.store.APIKeyModelStatsWithFilter(ctx, filter)
		if err != nil {
			return Response{}, err
		}
		response.APIKeyStats = buildAPIKeyStats(stats, prices)
	}
	if req.Include.FilterOptions {
		options, err := s.filterOptions(ctx, filter, prices)
		if err != nil {
			return Response{}, err
		}
		response.FilterOptions = options
	}
	if req.Include.TaskBuckets {
		response.TaskBuckets = buildTaskBuckets(taskBuckets)
	}
	if req.Include.RecentFailures > 0 {
		failures, err := s.store.RecentFailuresWithFilter(ctx, filter, req.Include.RecentFailures)
		if err != nil {
			return Response{}, err
		}
		response.RecentFailures = buildRecentFailures(failures)
	}
	if req.Include.EventsPage != nil {
		limit := req.Include.EventsPage.Limit
		if limit <= 0 {
			limit = defaultEventsLimit
		}
		if limit > maxEventsLimit {
			limit = maxEventsLimit
		}
		beforeMS := int64(0)
		if req.Include.EventsPage.BeforeMS != nil {
			beforeMS = *req.Include.EventsPage.BeforeMS
		}
		page, err := s.store.EventsPageWithFilter(ctx, filter, beforeMS, limit)
		if err != nil {
			return Response{}, err
		}
		response.Events = buildEvents(page)
	}

	return response, nil
}

func buildFilter(req Request) store.AnalyticsFilter {
	includeFailed := true
	if req.Filters.IncludeFailed != nil {
		includeFailed = *req.Filters.IncludeFailed
	}
	return store.AnalyticsFilter{
		FromMS:            req.FromMS,
		ToMS:              req.ToMS,
		SearchQuery:       req.SearchQuery,
		SearchAPIKeyHash:  req.SearchAPIKeyHash,
		Models:            req.Filters.Models,
		Providers:         req.Filters.Providers,
		Accounts:          req.Filters.Accounts,
		AuthIndices:       req.Filters.AuthIndices,
		APIKeyHashes:      req.Filters.APIKeyHashes,
		SourceHashes:      req.Filters.SourceHashes,
		IncludeFailed:     includeFailed,
		FailedOnly:        req.Filters.FailedOnly,
		ExcludeZeroTokens: req.Filters.ExcludeZeroTokens,
	}
}

func (s *Service) filterOptions(ctx context.Context, filter store.AnalyticsFilter, prices map[string]store.ModelPrice) (*FilterOptions, error) {
	optionFilter := filter
	optionFilter.Models = nil
	optionFilter.Providers = nil
	optionFilter.Accounts = nil
	optionFilter.AuthIndices = nil
	optionFilter.APIKeyHashes = nil
	optionFilter.SourceHashes = nil
	optionFilter.IncludeFailed = true
	optionFilter.FailedOnly = false
	optionFilter.ExcludeZeroTokens = false

	accountStats, err := s.store.AccountModelStatsWithFilter(ctx, optionFilter)
	if err != nil {
		return nil, err
	}
	apiKeyStats, err := s.store.APIKeyModelStatsWithFilter(ctx, optionFilter)
	if err != nil {
		return nil, err
	}
	channelStats, err := s.store.ChannelModelStatsWithFilter(ctx, optionFilter)
	if err != nil {
		return nil, err
	}
	modelStats, err := s.store.ModelStatsWithFilter(ctx, optionFilter, 0)
	if err != nil {
		return nil, err
	}

	return &FilterOptions{
		AccountStats: buildAccountStats(accountStats, prices),
		APIKeyStats:  buildAPIKeyStats(apiKeyStats, prices),
		ChannelShare: buildChannelShare(channelStats, prices),
		ModelStats:   buildModelStats(modelStats, prices),
	}, nil
}

func normalizeGranularity(input string, fromMS int64, toMS int64) string {
	if input == "day" || input == "hour" {
		return input
	}
	if toMS-fromMS <= 24*60*60*1000 {
		return "hour"
	}
	return "day"
}

func buildSummary(agg store.Aggregate, rolling store.Aggregate, activeDays int64, modelStats []store.ModelStat, taskBuckets []store.TaskBucket, prices map[string]store.ModelPrice, zeroTokenModels []string) *Summary {
	dayCount := activeDays
	if dayCount <= 0 {
		dayCount = 1
	}
	taskFailures := int64(0)
	for _, bucket := range taskBuckets {
		if bucket.Failure > 0 {
			taskFailures++
		}
	}
	approxTasks := int64(len(taskBuckets))
	return &Summary{
		TotalCalls:            agg.TotalCalls,
		SuccessCalls:          agg.SuccessCalls,
		FailureCalls:          agg.FailureCalls,
		SuccessRate:           ratio(agg.SuccessCalls, agg.TotalCalls),
		InputTokens:           agg.InputTokens,
		OutputTokens:          agg.OutputTokens,
		CachedTokens:          agg.CachedTokens,
		CacheReadTokens:       agg.CacheReadTokens,
		CacheCreationTokens:   agg.CacheCreationTokens,
		ReasoningTokens:       agg.ReasoningTokens,
		TotalTokens:           agg.TotalTokens,
		TotalCost:             sumCost(modelStats, prices),
		AverageLatencyMS:      nullableFloat(agg.AvgLatencyMS.Valid, agg.AvgLatencyMS.Float64),
		AverageTTFTMS:         nullableFloat(agg.AvgTTFTMS.Valid, agg.AvgTTFTMS.Float64),
		ZeroTokenCalls:        agg.ZeroTokenCalls,
		RPM30M:                float64(rolling.TotalCalls) / 30,
		TPM30M:                float64(rolling.TotalTokens) / 30,
		AvgDailyRequests:      float64(agg.TotalCalls) / float64(dayCount),
		AvgDailyTokens:        float64(agg.TotalTokens) / float64(dayCount),
		ApproxTasks:           approxTasks,
		ApproxTaskFailures:    taskFailures,
		ApproxTaskSuccessRate: ratio(approxTasks-taskFailures, approxTasks),
		ZeroTokenModels:       zeroTokenModels,
	}
}

func buildTimeline(points []store.TimelinePoint, granularity string, fromMS int64, toMS int64) []TimelinePoint {
	if len(points) == 0 {
		return nil
	}

	bucketSize := timelineBucketSize(granularity)
	startMS := timelineBucketStart(fromMS, granularity, fromMS)
	endMS := timelineBucketStart(toMS-1, granularity, fromMS)
	if endMS < startMS {
		endMS = startMS
	}

	bucketCount := int((endMS-startMS)/bucketSize) + 1
	pointsByBucket := make(map[int64]store.TimelinePoint, len(points))
	for _, point := range points {
		pointsByBucket[point.BucketMS] = point
	}

	if bucketCount > 0 && bucketCount <= maxFilledTimelinePoints {
		result := make([]TimelinePoint, 0, bucketCount)
		for bucketMS := startMS; bucketMS <= endMS; bucketMS += bucketSize {
			point := pointsByBucket[bucketMS]
			result = append(result, TimelinePoint{
				BucketMS: bucketMS,
				Label:    timelineLabel(bucketMS, granularity),
				Calls:    point.Calls,
				Tokens:   point.Tokens,
				Success:  point.Success,
				Failure:  point.Failure,
			})
		}
		return result
	}

	result := make([]TimelinePoint, 0, len(points))
	for _, point := range points {
		result = append(result, TimelinePoint{
			BucketMS: point.BucketMS,
			Label:    timelineLabel(point.BucketMS, granularity),
			Calls:    point.Calls,
			Tokens:   point.Tokens,
			Success:  point.Success,
			Failure:  point.Failure,
		})
	}
	return result
}

func timelineBucketSize(granularity string) int64 {
	if granularity == "day" {
		return timelineDayMS
	}
	return timelineHourMS
}

func timelineBucketStart(timestampMS int64, granularity string, fromMS int64) int64 {
	bucketSize := timelineBucketSize(granularity)
	anchorMS := (fromMS / bucketSize) * bucketSize
	if granularity == "day" {
		anchorMS = fromMS
	}
	return ((timestampMS-anchorMS)/bucketSize)*bucketSize + anchorMS
}

func buildHourly(points []store.HourlyPoint) []HourlyPoint {
	result := make([]HourlyPoint, 0, len(points))
	for _, point := range points {
		result = append(result, HourlyPoint(point))
	}
	return result
}

func buildModelShare(stats []store.ModelStat, prices map[string]store.ModelPrice) []ModelShareRow {
	aggregated := aggregateModelStats(stats, prices)
	result := make([]ModelShareRow, 0, len(aggregated))
	for _, stat := range aggregated {
		result = append(result, ModelShareRow{
			Model:  stat.Model,
			Calls:  stat.Calls,
			Tokens: stat.TotalTokens,
			Cost:   stat.Cost,
		})
	}
	return result
}

func buildModelStats(stats []store.ModelStat, prices map[string]store.ModelPrice) []ModelStat {
	aggregated := aggregateModelStats(stats, prices)
	result := make([]ModelStat, 0, len(aggregated))
	for _, stat := range aggregated {
		result = append(result, ModelStat{
			Model:               stat.Model,
			Calls:               stat.Calls,
			SuccessCalls:        stat.SuccessCalls,
			FailureCalls:        stat.Calls - stat.SuccessCalls,
			SuccessRate:         ratio(stat.SuccessCalls, stat.Calls),
			InputTokens:         stat.InputTokens,
			OutputTokens:        stat.OutputTokens,
			CachedTokens:        stat.CachedTokens,
			CacheReadTokens:     stat.CacheReadTokens,
			CacheCreationTokens: stat.CacheCreationTokens,
			TotalTokens:         stat.TotalTokens,
			Cost:                stat.Cost,
		})
	}
	return result
}

type aggregatedModelStat struct {
	Model               string
	Calls               int64
	SuccessCalls        int64
	InputTokens         int64
	OutputTokens        int64
	CachedTokens        int64
	CacheReadTokens     int64
	CacheCreationTokens int64
	TotalTokens         int64
	Cost                float64
}

func aggregateModelStats(stats []store.ModelStat, prices map[string]store.ModelPrice) []aggregatedModelStat {
	grouped := make(map[string]*aggregatedModelStat, len(stats))
	order := make([]string, 0, len(stats))
	for _, stat := range stats {
		entry := grouped[stat.Model]
		if entry == nil {
			entry = &aggregatedModelStat{Model: stat.Model}
			grouped[stat.Model] = entry
			order = append(order, stat.Model)
		}
		entry.Calls += stat.Calls
		entry.SuccessCalls += stat.SuccessCalls
		entry.InputTokens += stat.InputTokens
		entry.OutputTokens += stat.OutputTokens
		entry.CachedTokens += stat.CachedTokens
		entry.CacheReadTokens += stat.CacheReadTokens
		entry.CacheCreationTokens += stat.CacheCreationTokens
		entry.TotalTokens += stat.TotalTokens
		entry.Cost += costForStat(stat, prices)
	}
	result := make([]aggregatedModelStat, 0, len(order))
	for _, model := range order {
		result = append(result, *grouped[model])
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].Calls > result[j].Calls
	})
	return result
}

func buildChannelShare(stats []store.ChannelModelStat, prices map[string]store.ModelPrice) []ChannelShareRow {
	type accumulator struct {
		row        ChannelShareRow
		latencySum float64
		latencyN   int64
	}
	grouped := map[string]*accumulator{}
	for _, stat := range stats {
		authIndex := stat.AuthIndex
		if authIndex == "" {
			authIndex = "-"
		}
		entry := grouped[authIndex]
		if entry == nil {
			entry = &accumulator{row: ChannelShareRow{
				AuthIndex:            authIndex,
				Source:               stat.Source,
				AccountSnapshot:      stat.AccountSnapshot,
				AuthLabelSnapshot:    stat.AuthLabelSnapshot,
				AuthProviderSnapshot: stat.AuthProviderSnapshot,
			}}
			grouped[authIndex] = entry
		}
		fillChannelShareSnapshots(&entry.row, stat)
		entry.row.Calls += stat.Calls
		entry.row.Success += stat.SuccessCalls
		entry.row.Failure += stat.FailureCalls
		entry.row.Tokens += stat.TotalTokens
		entry.row.Cost += costForChannelStat(stat, prices)
		if stat.AvgLatencyMS.Valid {
			entry.latencySum += stat.AvgLatencyMS.Float64
			entry.latencyN++
		}
	}
	result := make([]ChannelShareRow, 0, len(grouped))
	for _, entry := range grouped {
		if entry.latencyN > 0 {
			value := entry.latencySum / float64(entry.latencyN)
			entry.row.AvgLatencyMS = &value
		}
		result = append(result, entry.row)
	}
	return result
}

func buildFailureSources(stats []store.FailureSourceStat) []FailureSourceRow {
	result := make([]FailureSourceRow, 0, len(stats))
	for _, stat := range stats {
		result = append(result, FailureSourceRow{
			Source:               stat.Source,
			SourceHash:           stat.SourceHash,
			AuthIndex:            stat.AuthIndex,
			AccountSnapshot:      stat.AccountSnapshot,
			AuthLabelSnapshot:    stat.AuthLabelSnapshot,
			AuthProviderSnapshot: stat.AuthProviderSnapshot,
			Calls:                stat.Calls,
			Failure:              stat.FailureCalls,
			LastSeenMS:           stat.LastSeenMS,
			AvgLatencyMS:         nullableFloat(stat.AvgLatencyMS.Valid, stat.AvgLatencyMS.Float64),
		})
	}
	return result
}

type accountStatAccumulator struct {
	row            AccountStatRow
	authIndices    map[string]struct{}
	sources        map[string]struct{}
	sourceHashes   map[string]struct{}
	models         map[string]*AccountModelStatRow
	latencySum     float64
	latencySamples int64
}

type apiKeyStatAccumulator struct {
	row            APIKeyStatRow
	authIndices    map[string]struct{}
	sources        map[string]struct{}
	sourceHashes   map[string]struct{}
	models         map[string]*AccountModelStatRow
	latencySum     float64
	latencySamples int64
}

func buildAccountStats(stats []store.AccountModelStat, prices map[string]store.ModelPrice) []AccountStatRow {
	grouped := map[string]*accountStatAccumulator{}
	for _, stat := range stats {
		id := accountGroupKey(stat.AccountSnapshot, stat.AuthLabelSnapshot, stat.Source, stat.AuthIndex)
		entry := grouped[id]
		if entry == nil {
			entry = &accountStatAccumulator{
				row: AccountStatRow{
					ID:                   id,
					AccountSnapshot:      stat.AccountSnapshot,
					AuthLabelSnapshot:    stat.AuthLabelSnapshot,
					AuthProviderSnapshot: stat.AuthProviderSnapshot,
				},
				authIndices:  map[string]struct{}{},
				sources:      map[string]struct{}{},
				sourceHashes: map[string]struct{}{},
				models:       map[string]*AccountModelStatRow{},
			}
			grouped[id] = entry
		}
		fillAccountStatSnapshots(&entry.row, stat.AccountSnapshot, stat.AuthLabelSnapshot, stat.AuthProviderSnapshot)
		addSetValue(entry.authIndices, stat.AuthIndex)
		addSetValue(entry.sources, stat.Source)
		addSetValue(entry.sourceHashes, stat.SourceHash)
		cost := costForAccountModelStat(stat, prices)
		addAccountTotals(
			&entry.row.Calls,
			&entry.row.SuccessCalls,
			&entry.row.FailureCalls,
			&entry.row.InputTokens,
			&entry.row.OutputTokens,
			&entry.row.CachedTokens,
			&entry.row.CacheReadTokens,
			&entry.row.CacheCreationTokens,
			&entry.row.TotalTokens,
			&entry.row.Cost,
			stat.Calls,
			stat.SuccessCalls,
			stat.FailureCalls,
			stat.InputTokens,
			stat.OutputTokens,
			stat.CachedTokens,
			stat.CacheReadTokens,
			stat.CacheCreationTokens,
			stat.TotalTokens,
			cost,
		)
		if stat.LastSeenMS > entry.row.LastSeenMS {
			entry.row.LastSeenMS = stat.LastSeenMS
		}
		if stat.AvgLatencyMS.Valid && stat.LatencySamples > 0 {
			entry.latencySum += stat.AvgLatencyMS.Float64 * float64(stat.LatencySamples)
			entry.latencySamples += stat.LatencySamples
		}
		addAccountModelStat(entry.models, stat.Model, stat.Calls, stat.SuccessCalls, stat.FailureCalls, stat.InputTokens, stat.OutputTokens, stat.CachedTokens, stat.CacheReadTokens, stat.CacheCreationTokens, stat.TotalTokens, cost, stat.LastSeenMS)
	}

	result := make([]AccountStatRow, 0, len(grouped))
	for _, entry := range grouped {
		entry.row.SuccessRate = ratio(entry.row.SuccessCalls, entry.row.Calls)
		entry.row.AuthIndices = sortedSetValues(entry.authIndices)
		entry.row.Sources = sortedSetValues(entry.sources)
		entry.row.SourceHashes = sortedSetValues(entry.sourceHashes)
		entry.row.Models = sortedAccountModelStats(entry.models)
		if entry.latencySamples > 0 {
			value := entry.latencySum / float64(entry.latencySamples)
			entry.row.AvgLatencyMS = &value
		}
		result = append(result, entry.row)
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].LastSeenMS > result[j].LastSeenMS ||
			(result[i].LastSeenMS == result[j].LastSeenMS && result[i].Calls > result[j].Calls) ||
			(result[i].LastSeenMS == result[j].LastSeenMS && result[i].Calls == result[j].Calls && result[i].Cost > result[j].Cost)
	})
	return result
}

func buildAPIKeyStats(stats []store.APIKeyModelStat, prices map[string]store.ModelPrice) []APIKeyStatRow {
	grouped := map[string]*apiKeyStatAccumulator{}
	for _, stat := range stats {
		id := apiKeyGroupKey(stat.APIKeyHash, stat.SourceHash, stat.AuthIndex, stat.Source, stat.AuthProviderSnapshot)
		entry := grouped[id]
		if entry == nil {
			entry = &apiKeyStatAccumulator{
				row: APIKeyStatRow{
					ID:                   id,
					APIKeyHash:           stat.APIKeyHash,
					AccountSnapshot:      stat.AccountSnapshot,
					AuthLabelSnapshot:    stat.AuthLabelSnapshot,
					AuthProviderSnapshot: stat.AuthProviderSnapshot,
				},
				authIndices:  map[string]struct{}{},
				sources:      map[string]struct{}{},
				sourceHashes: map[string]struct{}{},
				models:       map[string]*AccountModelStatRow{},
			}
			grouped[id] = entry
		}
		fillAPIKeyStatSnapshots(&entry.row, stat.APIKeyHash, stat.AccountSnapshot, stat.AuthLabelSnapshot, stat.AuthProviderSnapshot)
		addSetValue(entry.authIndices, stat.AuthIndex)
		addSetValue(entry.sources, stat.Source)
		addSetValue(entry.sourceHashes, stat.SourceHash)
		cost := costForAPIKeyModelStat(stat, prices)
		addAccountTotals(
			&entry.row.Calls,
			&entry.row.SuccessCalls,
			&entry.row.FailureCalls,
			&entry.row.InputTokens,
			&entry.row.OutputTokens,
			&entry.row.CachedTokens,
			&entry.row.CacheReadTokens,
			&entry.row.CacheCreationTokens,
			&entry.row.TotalTokens,
			&entry.row.Cost,
			stat.Calls,
			stat.SuccessCalls,
			stat.FailureCalls,
			stat.InputTokens,
			stat.OutputTokens,
			stat.CachedTokens,
			stat.CacheReadTokens,
			stat.CacheCreationTokens,
			stat.TotalTokens,
			cost,
		)
		if stat.LastSeenMS > entry.row.LastSeenMS {
			entry.row.LastSeenMS = stat.LastSeenMS
		}
		if stat.AvgLatencyMS.Valid && stat.LatencySamples > 0 {
			entry.latencySum += stat.AvgLatencyMS.Float64 * float64(stat.LatencySamples)
			entry.latencySamples += stat.LatencySamples
		}
		addAccountModelStat(entry.models, stat.Model, stat.Calls, stat.SuccessCalls, stat.FailureCalls, stat.InputTokens, stat.OutputTokens, stat.CachedTokens, stat.CacheReadTokens, stat.CacheCreationTokens, stat.TotalTokens, cost, stat.LastSeenMS)
	}

	result := make([]APIKeyStatRow, 0, len(grouped))
	for _, entry := range grouped {
		entry.row.SuccessRate = ratio(entry.row.SuccessCalls, entry.row.Calls)
		entry.row.AuthIndices = sortedSetValues(entry.authIndices)
		entry.row.Sources = sortedSetValues(entry.sources)
		entry.row.SourceHashes = sortedSetValues(entry.sourceHashes)
		entry.row.Models = sortedAccountModelStats(entry.models)
		if entry.latencySamples > 0 {
			value := entry.latencySum / float64(entry.latencySamples)
			entry.row.AvgLatencyMS = &value
		}
		result = append(result, entry.row)
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].LastSeenMS > result[j].LastSeenMS ||
			(result[i].LastSeenMS == result[j].LastSeenMS && result[i].Calls > result[j].Calls) ||
			(result[i].LastSeenMS == result[j].LastSeenMS && result[i].Calls == result[j].Calls && result[i].Cost > result[j].Cost)
	})
	return result
}

func fillChannelShareSnapshots(row *ChannelShareRow, stat store.ChannelModelStat) {
	if row.Source == "" {
		row.Source = stat.Source
	}
	if row.AccountSnapshot == "" {
		row.AccountSnapshot = stat.AccountSnapshot
	}
	if row.AuthLabelSnapshot == "" {
		row.AuthLabelSnapshot = stat.AuthLabelSnapshot
	}
	if row.AuthProviderSnapshot == "" {
		row.AuthProviderSnapshot = stat.AuthProviderSnapshot
	}
}

func accountGroupKey(accountSnapshot, authLabelSnapshot, source, authIndex string) string {
	if strings.TrimSpace(accountSnapshot) != "" {
		return accountSnapshot
	}
	if strings.TrimSpace(authLabelSnapshot) != "" {
		return authLabelSnapshot
	}
	if strings.TrimSpace(source) != "" {
		return source
	}
	if strings.TrimSpace(authIndex) != "" {
		return authIndex
	}
	return "-"
}

func apiKeyGroupKey(apiKeyHash, sourceHash, authIndex, source, provider string) string {
	if strings.TrimSpace(apiKeyHash) != "" {
		return strings.ToLower(strings.TrimSpace(apiKeyHash))
	}
	parts := []string{"unknown-client-api-key"}
	for _, value := range []string{sourceHash, authIndex, source, provider} {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			trimmed = "-"
		}
		parts = append(parts, trimmed)
	}
	return strings.Join(parts, ":")
}

func fillAccountStatSnapshots(row *AccountStatRow, accountSnapshot, authLabelSnapshot, authProviderSnapshot string) {
	if row.AccountSnapshot == "" {
		row.AccountSnapshot = accountSnapshot
	}
	if row.AuthLabelSnapshot == "" {
		row.AuthLabelSnapshot = authLabelSnapshot
	}
	if row.AuthProviderSnapshot == "" {
		row.AuthProviderSnapshot = authProviderSnapshot
	}
}

func fillAPIKeyStatSnapshots(row *APIKeyStatRow, apiKeyHash, accountSnapshot, authLabelSnapshot, authProviderSnapshot string) {
	if row.APIKeyHash == "" {
		row.APIKeyHash = apiKeyHash
	}
	if row.AccountSnapshot == "" {
		row.AccountSnapshot = accountSnapshot
	}
	if row.AuthLabelSnapshot == "" {
		row.AuthLabelSnapshot = authLabelSnapshot
	}
	if row.AuthProviderSnapshot == "" {
		row.AuthProviderSnapshot = authProviderSnapshot
	}
}

func addSetValue(values map[string]struct{}, value string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return
	}
	values[trimmed] = struct{}{}
}

func sortedSetValues(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func addAccountTotals(
	calls *int64,
	successCalls *int64,
	failureCalls *int64,
	inputTokens *int64,
	outputTokens *int64,
	cachedTokens *int64,
	cacheReadTokens *int64,
	cacheCreationTokens *int64,
	totalTokens *int64,
	cost *float64,
	addCalls int64,
	addSuccessCalls int64,
	addFailureCalls int64,
	addInputTokens int64,
	addOutputTokens int64,
	addCachedTokens int64,
	addCacheReadTokens int64,
	addCacheCreationTokens int64,
	addTotalTokens int64,
	addCost float64,
) {
	*calls += addCalls
	*successCalls += addSuccessCalls
	*failureCalls += addFailureCalls
	*inputTokens += addInputTokens
	*outputTokens += addOutputTokens
	*cachedTokens += addCachedTokens
	*cacheReadTokens += addCacheReadTokens
	*cacheCreationTokens += addCacheCreationTokens
	*totalTokens += addTotalTokens
	*cost += addCost
}

func addAccountModelStat(
	models map[string]*AccountModelStatRow,
	model string,
	calls int64,
	successCalls int64,
	failureCalls int64,
	inputTokens int64,
	outputTokens int64,
	cachedTokens int64,
	cacheReadTokens int64,
	cacheCreationTokens int64,
	totalTokens int64,
	cost float64,
	lastSeenMS int64,
) {
	modelKey := model
	if strings.TrimSpace(modelKey) == "" {
		modelKey = "-"
	}
	entry := models[modelKey]
	if entry == nil {
		entry = &AccountModelStatRow{Model: modelKey}
		models[modelKey] = entry
	}
	entry.Calls += calls
	entry.SuccessCalls += successCalls
	entry.FailureCalls += failureCalls
	entry.InputTokens += inputTokens
	entry.OutputTokens += outputTokens
	entry.CachedTokens += cachedTokens
	entry.CacheReadTokens += cacheReadTokens
	entry.CacheCreationTokens += cacheCreationTokens
	entry.TotalTokens += totalTokens
	entry.Cost += cost
	if lastSeenMS > entry.LastSeenMS {
		entry.LastSeenMS = lastSeenMS
	}
	entry.SuccessRate = ratio(entry.SuccessCalls, entry.Calls)
}

func sortedAccountModelStats(models map[string]*AccountModelStatRow) []AccountModelStatRow {
	result := make([]AccountModelStatRow, 0, len(models))
	for _, model := range models {
		result = append(result, *model)
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].Cost > result[j].Cost ||
			(result[i].Cost == result[j].Cost && result[i].Calls > result[j].Calls) ||
			(result[i].Cost == result[j].Cost && result[i].Calls == result[j].Calls && result[i].LastSeenMS > result[j].LastSeenMS)
	})
	return result
}

func buildTaskBuckets(buckets []store.TaskBucket) []TaskBucketRow {
	result := make([]TaskBucketRow, 0, len(buckets))
	for _, bucket := range buckets {
		result = append(result, TaskBucketRow{
			BucketKey:           bucket.BucketKey,
			Total:               bucket.Total,
			Success:             bucket.Success,
			Failure:             bucket.Failure,
			FirstMS:             bucket.FirstMS,
			LastMS:              bucket.LastMS,
			Source:              bucket.Source,
			SourceHash:          bucket.SourceHash,
			AuthIndex:           bucket.AuthIndex,
			Models:              splitCSV(bucket.Models),
			Endpoints:           splitCSV(bucket.Endpoints),
			InputTokens:         bucket.InputTokens,
			OutputTokens:        bucket.OutputTokens,
			CachedTokens:        bucket.CachedTokens,
			CacheReadTokens:     bucket.CacheReadTokens,
			CacheCreationTokens: bucket.CacheCreationTokens,
			TotalTokens:         bucket.TotalTokens,
			AvgLatencyMS:        nullableFloat(bucket.AvgLatencyMS.Valid, bucket.AvgLatencyMS.Float64),
			MaxLatencyMS:        nullableInt(bucket.MaxLatencyMS.Valid, bucket.MaxLatencyMS.Int64),
		})
	}
	return result
}

func buildRecentFailures(failures []store.RecentFailure) []RecentFailure {
	result := make([]RecentFailure, 0, len(failures))
	for _, failure := range failures {
		result = append(result, RecentFailure{
			TimestampMS:           failure.TimestampMS,
			Model:                 failure.Model,
			APIKeyHash:            failure.APIKeyHash,
			Source:                failure.Source,
			SourceHash:            failure.SourceHash,
			AuthIndex:             failure.AuthIndex,
			AccountSnapshot:       failure.AccountSnapshot,
			AuthLabelSnapshot:     failure.AuthLabelSnapshot,
			AuthProviderSnapshot:  failure.AuthProviderSnapshot,
			AuthProjectIDSnapshot: failure.AuthProjectIDSnapshot,
			Endpoint:              failure.Endpoint,
			DurationMS:            nullableInt(failure.LatencyMS.Valid, failure.LatencyMS.Int64),
			FailStatusCode:        nullableInt(failure.FailStatusCode.Valid, failure.FailStatusCode.Int64),
			FailSummary:           failure.FailSummary,
		})
	}
	return result
}

func buildEvents(page store.EventsPage) *EventsResponse {
	items := make([]EventRow, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, EventRow{
			EventHash:             item.EventHash,
			TimestampMS:           item.TimestampMS,
			Model:                 item.Model,
			ResolvedModel:         item.ResolvedModel,
			Endpoint:              item.Endpoint,
			Method:                item.Method,
			Path:                  item.Path,
			AuthIndex:             item.AuthIndex,
			Source:                item.Source,
			SourceHash:            item.SourceHash,
			APIKeyHash:            item.APIKeyHash,
			AccountSnapshot:       item.AccountSnapshot,
			AuthLabelSnapshot:     item.AuthLabelSnapshot,
			AuthProviderSnapshot:  item.AuthProviderSnapshot,
			AuthProjectIDSnapshot: item.AuthProjectIDSnapshot,
			ReasoningEffort:       item.ReasoningEffort,
			ServiceTier:           item.ServiceTier,
			ExecutorType:          item.ExecutorType,
			InputTokens:           item.InputTokens,
			OutputTokens:          item.OutputTokens,
			CachedTokens:          item.CachedTokens,
			CacheReadTokens:       item.CacheReadTokens,
			CacheCreationTokens:   item.CacheCreationTokens,
			ReasoningTokens:       item.ReasoningTokens,
			TotalTokens:           item.TotalTokens,
			LatencyMS:             nullableInt(item.LatencyMS.Valid, item.LatencyMS.Int64),
			TTFTMS:                nullableInt(item.TTFTMS.Valid, item.TTFTMS.Int64),
			Failed:                item.Failed,
			FailStatusCode:        nullableInt(item.FailStatusCode.Valid, item.FailStatusCode.Int64),
			FailSummary:           item.FailSummary,
		})
	}
	return &EventsResponse{Items: items, NextBeforeMS: page.NextBeforeMS, HasMore: page.HasMore}
}

func sumCost(stats []store.ModelStat, prices map[string]store.ModelPrice) float64 {
	total := 0.0
	for _, stat := range stats {
		total += costForStat(stat, prices)
	}
	return total
}

func costForStat(stat store.ModelStat, prices map[string]store.ModelPrice) float64 {
	model := stat.BillingModel
	if model == "" {
		model = stat.Model
	}
	return pricing.CostForModel(model, pricing.ModelTokens{
		InputTokens:         stat.BillableInputTokens,
		OutputTokens:        stat.OutputTokens,
		CachedTokens:        stat.CachedTokens,
		CacheReadTokens:     stat.CacheReadTokens,
		CacheCreationTokens: stat.CacheCreationTokens,
	}, prices)
}

func costForChannelStat(stat store.ChannelModelStat, prices map[string]store.ModelPrice) float64 {
	model := stat.BillingModel
	if model == "" {
		model = stat.Model
	}
	return pricing.CostForModel(model, pricing.ModelTokens{
		InputTokens:         stat.BillableInputTokens,
		OutputTokens:        stat.OutputTokens,
		CachedTokens:        stat.CachedTokens,
		CacheReadTokens:     stat.CacheReadTokens,
		CacheCreationTokens: stat.CacheCreationTokens,
	}, prices)
}

func costForAccountModelStat(stat store.AccountModelStat, prices map[string]store.ModelPrice) float64 {
	model := stat.BillingModel
	if model == "" {
		model = stat.Model
	}
	return pricing.CostForModel(model, pricing.ModelTokens{
		InputTokens:         stat.BillableInputTokens,
		OutputTokens:        stat.OutputTokens,
		CachedTokens:        stat.CachedTokens,
		CacheReadTokens:     stat.CacheReadTokens,
		CacheCreationTokens: stat.CacheCreationTokens,
	}, prices)
}

func costForAPIKeyModelStat(stat store.APIKeyModelStat, prices map[string]store.ModelPrice) float64 {
	model := stat.BillingModel
	if model == "" {
		model = stat.Model
	}
	return pricing.CostForModel(model, pricing.ModelTokens{
		InputTokens:         stat.BillableInputTokens,
		OutputTokens:        stat.OutputTokens,
		CachedTokens:        stat.CachedTokens,
		CacheReadTokens:     stat.CacheReadTokens,
		CacheCreationTokens: stat.CacheCreationTokens,
	}, prices)
}

func ratio(part int64, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(part) / float64(total)
}

func nullableFloat(valid bool, value float64) *float64 {
	if !valid {
		return nil
	}
	return &value
}

func nullableInt(valid bool, value int64) *int64 {
	if !valid {
		return nil
	}
	return &value
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func timelineLabel(bucketMS int64, granularity string) string {
	tm := time.UnixMilli(bucketMS).UTC()
	if granularity == "day" {
		return tm.Format("01/02")
	}
	return tm.Format("15:04")
}
