package store

import (
	"context"
	"database/sql"

	"github.com/seakee/cpa-statistics/apps/manager-server/internal/model"
	"github.com/seakee/cpa-statistics/apps/manager-server/internal/repository/apikeyalias"
	"github.com/seakee/cpa-statistics/apps/manager-server/internal/repository/deadletter"
	"github.com/seakee/cpa-statistics/apps/manager-server/internal/repository/modelprice"
	"github.com/seakee/cpa-statistics/apps/manager-server/internal/repository/setting"
	sqliterepo "github.com/seakee/cpa-statistics/apps/manager-server/internal/repository/sqlite"
	"github.com/seakee/cpa-statistics/apps/manager-server/internal/repository/usageevent"
	"github.com/seakee/cpa-statistics/apps/manager-server/internal/security"
	"github.com/seakee/cpa-statistics/apps/manager-server/internal/usage"
)

type Setup = model.Setup
type ManagerConfig = model.ManagerConfig
type AdminCredential = model.AdminCredential
type BootstrapState = model.BootstrapState
type ManagerCPAConnectionConfig = model.ManagerCPAConnectionConfig
type ManagerCollectorConfig = model.ManagerCollectorConfig
type ManagerExternalUsageServiceConfig = model.ManagerExternalUsageServiceConfig
type InsertResult = model.InsertResult
type ModelPrice = model.ModelPrice
type ModelPriceSyncResult = model.ModelPriceSyncResult
type APIKeyAlias = model.APIKeyAlias

type ClearDataOptions struct {
	UsageEvents bool
	DeadLetters bool
}

type ClearDataResult struct {
	UsageEvents int64 `json:"usageEvents"`
	DeadLetters int64 `json:"deadLetters"`
}

// Aggregation result types re-exported for service-layer consumers.
type Aggregate = usageevent.Aggregate
type ModelStat = usageevent.ModelStat
type RecentFailure = usageevent.RecentFailure
type AnalyticsFilter = usageevent.AnalyticsFilter
type TimelinePoint = usageevent.TimelinePoint
type HourlyPoint = usageevent.HourlyPoint
type ChannelModelStat = usageevent.ChannelModelStat
type FailureSourceStat = usageevent.FailureSourceStat
type AccountModelStat = usageevent.AccountModelStat
type APIKeyModelStat = usageevent.APIKeyModelStat
type TaskBucket = usageevent.TaskBucket
type EventPageItem = usageevent.EventPageItem
type EventsPage = usageevent.EventsPage

type Store struct {
	db *sql.DB

	Settings      setting.Repository
	UsageEvents   usageevent.Repository
	DeadLetters   deadletter.Repository
	ModelPrices   modelprice.Repository
	APIKeyAliases apikeyalias.Repository
}

func Open(path string, protector ...*security.Protector) (*Store, error) {
	db, err := sqliterepo.Open(path)
	if err != nil {
		return nil, err
	}
	if _, err := usageevent.New(db).BackfillCacheAccounting(context.Background(), 500); err != nil {
		_ = db.Close()
		return nil, err
	}
	return New(db, protector...), nil
}

func New(db *sql.DB, protector ...*security.Protector) *Store {
	return &Store{
		db:            db,
		Settings:      setting.New(db, protector...),
		UsageEvents:   usageevent.New(db),
		DeadLetters:   deadletter.New(db),
		ModelPrices:   modelprice.New(db),
		APIKeyAliases: apikeyalias.New(db),
	}
}

func (s *Store) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) SaveSetup(ctx context.Context, setup Setup) error {
	return s.Settings.SaveSetup(ctx, setup)
}

func (s *Store) LoadSetup(ctx context.Context) (Setup, bool, error) {
	return s.Settings.LoadSetup(ctx)
}

func (s *Store) SaveManagerConfig(ctx context.Context, cfg ManagerConfig) error {
	return s.Settings.SaveManagerConfig(ctx, cfg)
}

func (s *Store) LoadManagerConfig(ctx context.Context) (ManagerConfig, bool, error) {
	return s.Settings.LoadManagerConfig(ctx)
}

func (s *Store) SaveAdminCredential(ctx context.Context, credential AdminCredential) error {
	return s.Settings.SaveAdminCredential(ctx, credential)
}

func (s *Store) LoadAdminCredential(ctx context.Context) (AdminCredential, bool, error) {
	return s.Settings.LoadAdminCredential(ctx)
}

func (s *Store) SaveBootstrapState(ctx context.Context, state BootstrapState) error {
	return s.Settings.SaveBootstrapState(ctx, state)
}

func (s *Store) LoadBootstrapState(ctx context.Context) (BootstrapState, bool, error) {
	return s.Settings.LoadBootstrapState(ctx)
}

func (s *Store) HasHistoricalData(ctx context.Context) (bool, error) {
	return s.Settings.HasHistoricalData(ctx)
}

func (s *Store) LoadModelPrices(ctx context.Context) (map[string]ModelPrice, error) {
	return s.ModelPrices.LoadAll(ctx)
}

func (s *Store) SaveModelPrices(ctx context.Context, prices map[string]ModelPrice) error {
	return s.ModelPrices.ReplaceAll(ctx, prices)
}

func (s *Store) UpsertSyncedModelPrices(ctx context.Context, prices map[string]ModelPrice) (ModelPriceSyncResult, error) {
	return s.ModelPrices.UpsertSynced(ctx, prices)
}

func (s *Store) LoadAPIKeyAliases(ctx context.Context) ([]APIKeyAlias, error) {
	return s.APIKeyAliases.LoadAll(ctx)
}

func (s *Store) UpsertAPIKeyAliases(ctx context.Context, aliases []APIKeyAlias) error {
	return s.APIKeyAliases.UpsertMany(ctx, aliases, nil, false)
}

func (s *Store) UpsertAPIKeyAliasesWithActiveHashes(ctx context.Context, aliases []APIKeyAlias, activeHashes []string, allowOrphanCleanup bool) error {
	return s.APIKeyAliases.UpsertMany(ctx, aliases, activeHashes, allowOrphanCleanup)
}

func (s *Store) DeleteAPIKeyAlias(ctx context.Context, apiKeyHash string) error {
	return s.APIKeyAliases.Delete(ctx, apiKeyHash)
}

func (s *Store) InsertEvents(ctx context.Context, events []usage.Event) (InsertResult, error) {
	return s.UsageEvents.InsertBatch(ctx, events)
}

func (s *Store) AddDeadLetter(ctx context.Context, payload string, parseErr error) error {
	return s.DeadLetters.Insert(ctx, payload, parseErr.Error())
}

func (s *Store) RecentEvents(ctx context.Context, limit int) ([]usage.Event, error) {
	return s.UsageEvents.ListRecent(ctx, limit)
}

func (s *Store) Counts(ctx context.Context) (events int64, deadLetters int64, err error) {
	events, err = s.UsageEvents.Count(ctx)
	if err != nil {
		return 0, 0, err
	}
	deadLetters, err = s.DeadLetters.Count(ctx)
	if err != nil {
		return 0, 0, err
	}
	return events, deadLetters, nil
}

func (s *Store) ClearData(ctx context.Context, opts ClearDataOptions) (ClearDataResult, error) {
	result := ClearDataResult{}
	if !opts.UsageEvents && !opts.DeadLetters {
		return result, nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return result, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if opts.UsageEvents {
		res, err := tx.ExecContext(ctx, `delete from usage_events`)
		if err != nil {
			return result, err
		}
		result.UsageEvents, _ = res.RowsAffected()
	}
	if opts.DeadLetters {
		res, err := tx.ExecContext(ctx, `delete from dead_letter_events`)
		if err != nil {
			return result, err
		}
		result.DeadLetters, _ = res.RowsAffected()
	}

	if err := tx.Commit(); err != nil {
		return result, err
	}
	return result, nil
}

func (s *Store) ExportJSONL(ctx context.Context) ([]byte, error) {
	return s.UsageEvents.ExportJSONL(ctx)
}

// AggregateBetween computes summary metrics over [fromMs, toMs).
func (s *Store) AggregateBetween(ctx context.Context, fromMs, toMs int64) (Aggregate, error) {
	return s.UsageEvents.AggregateBetween(ctx, fromMs, toMs)
}

// TopModelsBetween returns the most active models ordered by call count.
func (s *Store) TopModelsBetween(ctx context.Context, fromMs, toMs int64, limit int) ([]ModelStat, error) {
	return s.UsageEvents.TopModelsBetween(ctx, fromMs, toMs, limit)
}

// ModelStatsBetween returns per-model totals for all models in a window.
func (s *Store) ModelStatsBetween(ctx context.Context, fromMs, toMs int64) ([]ModelStat, error) {
	return s.UsageEvents.ModelStatsBetween(ctx, fromMs, toMs)
}

// RecentFailuresBetween returns the most recent failed events in window.
func (s *Store) RecentFailuresBetween(ctx context.Context, fromMs, toMs int64, limit int) ([]RecentFailure, error) {
	return s.UsageEvents.RecentFailuresBetween(ctx, fromMs, toMs, limit)
}

func (s *Store) HourlyTimelineBetween(ctx context.Context, fromMs, toMs int64) ([]TimelinePoint, error) {
	return s.UsageEvents.HourlyTimelineBetween(ctx, fromMs, toMs)
}

func (s *Store) BucketTimelineBetween(ctx context.Context, fromMs, toMs int64, bucketMs int64) ([]TimelinePoint, error) {
	return s.UsageEvents.BucketTimelineBetween(ctx, fromMs, toMs, bucketMs)
}

func (s *Store) AggregateWithFilter(ctx context.Context, filter AnalyticsFilter) (Aggregate, error) {
	return s.UsageEvents.AggregateWithFilter(ctx, filter)
}

func (s *Store) ModelStatsWithFilter(ctx context.Context, filter AnalyticsFilter, limit int) ([]ModelStat, error) {
	return s.UsageEvents.ModelStatsWithFilter(ctx, filter, limit)
}

func (s *Store) TimelineWithFilter(ctx context.Context, filter AnalyticsFilter, granularity string) ([]TimelinePoint, error) {
	return s.UsageEvents.TimelineWithFilter(ctx, filter, granularity)
}

func (s *Store) HourlyDistributionWithFilter(ctx context.Context, filter AnalyticsFilter) ([]HourlyPoint, error) {
	return s.UsageEvents.HourlyDistributionWithFilter(ctx, filter)
}

func (s *Store) ChannelModelStatsWithFilter(ctx context.Context, filter AnalyticsFilter) ([]ChannelModelStat, error) {
	return s.UsageEvents.ChannelModelStatsWithFilter(ctx, filter)
}

func (s *Store) FailureSourcesWithFilter(ctx context.Context, filter AnalyticsFilter) ([]FailureSourceStat, error) {
	return s.UsageEvents.FailureSourcesWithFilter(ctx, filter)
}

func (s *Store) AccountModelStatsWithFilter(ctx context.Context, filter AnalyticsFilter) ([]AccountModelStat, error) {
	return s.UsageEvents.AccountModelStatsWithFilter(ctx, filter)
}

func (s *Store) APIKeyModelStatsWithFilter(ctx context.Context, filter AnalyticsFilter) ([]APIKeyModelStat, error) {
	return s.UsageEvents.APIKeyModelStatsWithFilter(ctx, filter)
}

func (s *Store) TaskBucketsWithFilter(ctx context.Context, filter AnalyticsFilter) ([]TaskBucket, error) {
	return s.UsageEvents.TaskBucketsWithFilter(ctx, filter)
}

func (s *Store) RecentFailuresWithFilter(ctx context.Context, filter AnalyticsFilter, limit int) ([]RecentFailure, error) {
	return s.UsageEvents.RecentFailuresWithFilter(ctx, filter, limit)
}

func (s *Store) EventsPageWithFilter(ctx context.Context, filter AnalyticsFilter, beforeMS int64, limit int) (EventsPage, error) {
	return s.UsageEvents.EventsPageWithFilter(ctx, filter, beforeMS, limit)
}

func (s *Store) ActiveDaysWithFilter(ctx context.Context, filter AnalyticsFilter) (int64, error) {
	return s.UsageEvents.ActiveDaysWithFilter(ctx, filter)
}

func (s *Store) ZeroTokenModelsWithFilter(ctx context.Context, filter AnalyticsFilter) ([]string, error) {
	return s.UsageEvents.ZeroTokenModelsWithFilter(ctx, filter)
}
