import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Modal } from '@/components/ui/Modal';
import {
  IconCheck,
  IconDatabaseZap,
  IconPencil,
  IconPlus,
  IconRefreshCw,
  IconSearch,
  IconTrash2,
} from '@/components/ui/icons';
import { usePanelFeatureAvailability } from '@/hooks/usePanelFeatureAvailability';
import type { ModelPriceSyncCandidate, ModelPriceSyncResponse } from '@/services/api/usageService';
import { useNotificationStore } from '@/stores/useNotificationStore';
import { useUsageData } from '@/features/monitoring/hooks/useUsageData';
import {
  applyCandidatePrice,
  buildModelPriceRows,
  buildModelPriceSummary,
  buildPriceFromDraft,
  buildSyncPriceModelsFromUsage,
  createEmptyPriceDraft,
  createPriceDraft,
  filterModelPriceRows,
  formatPriceUnit,
  type ModelPriceFilter,
  type PriceDraft,
} from '@/features/monitoring/model/modelPricesPageModel';
import {
  readModelPricesPageUiState,
  writeModelPricesPageUiState,
} from './modelPricesPageUiState';
import styles from './ModelPricesPage.module.scss';

const FILTERS: ModelPriceFilter[] = ['all', 'missing', 'candidates', 'saved'];

const resolveErrorMessage = (error: unknown, fallback: string) => {
  const rawMessage = error instanceof Error ? error.message : String(error || fallback);
  return rawMessage === 'model_price_sync_requires_usage_service'
    ? 'model_price_sync_requires_usage_service'
    : rawMessage;
};

interface ModelPricesPageProps {
  embedded?: boolean;
  dataPageLayout?: boolean;
  dataTransferActions?: ReactNode;
}

export function ModelPricesPage({
  embedded = false,
  dataPageLayout = false,
  dataTransferActions,
}: ModelPricesPageProps) {
  const { t } = useTranslation();
  const { showNotification } = useNotificationStore();
  const featureAvailability = usePanelFeatureAvailability();
  const { usage, loading, modelPrices, setModelPrices, syncModelPrices, usageServiceAvailable } =
    useUsageData();
  const initialUiState = useRef(readModelPricesPageUiState());
  const [search, setSearch] = useState(() => initialUiState.current.search);
  const [filter, setFilter] = useState<ModelPriceFilter>(() => initialUiState.current.filter);
  const [syncing, setSyncing] = useState(false);
  const [syncResult, setSyncResult] = useState<ModelPriceSyncResponse | null>(null);
  const [selectedCandidates, setSelectedCandidates] = useState<Record<string, string>>({});
  const [draft, setDraft] = useState<PriceDraft>(() => createEmptyPriceDraft());
  const [manualEditorOpen, setManualEditorOpen] = useState(false);

  const syncModels = useMemo(
    () => buildSyncPriceModelsFromUsage(usage, modelPrices),
    [modelPrices, usage]
  );

  const candidateSets = useMemo(() => syncResult?.candidates ?? [], [syncResult?.candidates]);
  const rows = useMemo(
    () => buildModelPriceRows(usage, modelPrices, candidateSets),
    [candidateSets, modelPrices, usage]
  );
  const summary = useMemo(() => buildModelPriceSummary(rows), [rows]);
  const visibleRows = useMemo(
    () => filterModelPriceRows(rows, filter, search),
    [filter, rows, search]
  );

  const filterCounts = useMemo<Record<ModelPriceFilter, number>>(
    () => ({
      all: summary.total,
      missing: summary.missing,
      candidates: summary.candidates,
      saved: summary.saved,
    }),
    [summary]
  );

  useEffect(() => {
    writeModelPricesPageUiState({ search, filter });
  }, [filter, search]);

  const handleSync = async () => {
    if (syncModels.length === 0) {
      showNotification(t('usage_stats.model_price_sync_no_models'), 'warning');
      return;
    }
    setSyncing(true);
    try {
      const result = await syncModelPrices(syncModels);
      setSyncResult(result);
      showNotification(
        t('model_prices.sync_success_detail', {
          imported: result.imported,
          candidates: result.candidates?.length ?? 0,
          unmatched: result.unmatched?.length ?? 0,
        }),
        'success'
      );
    } catch (error: unknown) {
      const message = resolveErrorMessage(error, t('common.unknown_error'));
      showNotification(
        `${t('usage_stats.model_price_sync_failed')}: ${
          message === 'model_price_sync_requires_usage_service'
            ? t('usage_stats.model_price_sync_requires_usage_service')
            : message
        }`,
        'error'
      );
    } finally {
      setSyncing(false);
    }
  };

  const handleConfirmCandidate = async (model: string, candidate: ModelPriceSyncCandidate) => {
    const next = applyCandidatePrice(modelPrices, model, candidate);
    if (candidate.reason === 'legacy_alias_price' && candidate.sourceModelId !== model) {
      delete next[candidate.sourceModelId];
    }
    await setModelPrices(next);
    setSyncResult((previous) =>
      previous
        ? {
            ...previous,
            candidates: previous.candidates?.filter((set) => set.model !== model),
            unmatched: previous.unmatched?.filter((item) => item !== model),
          }
        : previous
    );
    showNotification(t('model_prices.candidate_confirmed'), 'success');
  };

  const handleSaveDraft = async () => {
    const price = buildPriceFromDraft(draft);
    const model = draft.model.trim();
    if (!model || !price) {
      showNotification(t('usage_stats.model_price_model_required'), 'warning');
      return;
    }
    await setModelPrices({
      ...modelPrices,
      [model]: {
        ...price,
      },
    });
    setDraft(createEmptyPriceDraft());
    setManualEditorOpen(false);
    showNotification(t('usage_stats.model_price_saved'), 'success');
  };

  const handleDelete = async (model: string) => {
    const next = { ...modelPrices };
    delete next[model];
    await setModelPrices(next);
    if (draft.model === model) {
      setDraft(createEmptyPriceDraft());
      setManualEditorOpen(false);
    }
  };

  const setDraftField = (field: keyof PriceDraft, value: string) => {
    setDraft((previous) => ({ ...previous, [field]: value }));
  };

  const openManualEditor = (model = '', price = modelPrices[model]) => {
    setDraft(createPriceDraft(model, price));
    setManualEditorOpen(true);
  };

  const closeManualEditor = () => {
    setDraft(createEmptyPriceDraft());
    setManualEditorOpen(false);
  };

  const renderDataPageControl = (icon: ReactNode, label: ReactNode) =>
    dataPageLayout ? (
      <span data-model-prices-control-content="true">
        <span data-model-prices-control-icon="true">{icon}</span>
        <span data-model-prices-control-label="true">{label}</span>
      </span>
    ) : (
      label
    );

  return (
    <div
      className={`${styles.page} ${dataPageLayout ? styles.dataPageLayout : ''}`}
      data-model-prices-layout={dataPageLayout ? 'data-page' : undefined}
    >
      <section
        className={styles.actionBar}
        aria-label={t('common.action')}
        data-model-prices-section={dataPageLayout ? 'actions' : undefined}
      >
        <div
          className={styles.titleGroup}
          data-model-prices-action-part={dataPageLayout ? 'title' : undefined}
        >
          {!embedded && featureAvailability.requestMonitoringAvailable ? (
            <Link to="/usage" className={styles.backLink}>
              {t('model_prices.back_to_monitoring')}
            </Link>
          ) : null}
        </div>
        <div
          className={styles.actionGroup}
          data-model-prices-action-part={dataPageLayout ? 'controls' : undefined}
        >
          <span
            className={styles.metaPill}
            data-model-prices-control-item="status"
            data-model-prices-control-state={usageServiceAvailable ? 'connected' : 'disconnected'}
          >
            {renderDataPageControl(
              <IconCheck size={18} />,
              usageServiceAvailable
                ? dataPageLayout
                  ? (
                    <>
                      <span data-model-prices-status-name="true">Manager Server</span>
                      <span data-model-prices-status-connected="true">已连接</span>
                    </>
                  )
                  : t('model_prices.usage_service_ready')
                : dataPageLayout
                  ? t('model_prices.usage_service_disconnected', { defaultValue: '未连接' })
                  : t('model_prices.usage_service_required')
            )}
          </span>
          <span className={styles.metaPill} data-model-prices-control-item="count">
            {renderDataPageControl(
              <IconDatabaseZap size={18} />,
              t('model_prices.sync_model_count', { count: syncModels.length })
            )}
          </span>
          {dataPageLayout ? dataTransferActions : null}
          {dataPageLayout ? (
            <div data-model-prices-control-item="manual">
              <Button
                size="xs"
                variant="secondary"
                iconOnly
                onClick={() => openManualEditor()}
                className={styles.toolbarButton}
                aria-label={t('model_prices.add_manual')}
                title={t('model_prices.add_manual')}
              >
                <span data-model-prices-control-icon="true">
                  <IconPlus size={18} />
                </span>
              </Button>
              <span data-model-prices-control-label="true" aria-hidden="true">
                {t('model_prices.add_manual')}
              </span>
            </div>
          ) : (
            <Button
              size="xs"
              variant="secondary"
              onClick={() => openManualEditor()}
              className={styles.toolbarButton}
            >
              {t('model_prices.add_manual')}
            </Button>
          )}
          {dataPageLayout ? (
            <div data-model-prices-control-item="sync">
              <Button
                size="xs"
                iconOnly
                onClick={() => void handleSync()}
                loading={syncing}
                aria-label={t('usage_stats.model_price_sync')}
                title={t('usage_stats.model_price_sync')}
              >
                <span data-model-prices-control-icon="true">
                  <IconRefreshCw size={18} />
                </span>
              </Button>
              <span data-model-prices-control-label="true" aria-hidden="true">
                {t('usage_stats.model_price_sync')}
              </span>
            </div>
          ) : (
            <Button size="xs" onClick={() => void handleSync()} loading={syncing}>
              {t('usage_stats.model_price_sync')}
            </Button>
          )}
        </div>
      </section>

      <section
        className={styles.pricePanel}
        data-model-prices-section={dataPageLayout ? 'content' : undefined}
      >
        <div className={styles.panelToolbar}>
          <div className={styles.searchWrap}>
            <Input
              className={styles.compactInput}
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder={t('model_prices.search_placeholder')}
              rightElement={<IconSearch size={15} />}
            />
          </div>
          <div className={styles.filterGroup}>
            {FILTERS.map((item) => (
              <button
                key={item}
                type="button"
                className={`${styles.filterButton} ${
                  filter === item ? styles.filterButtonActive : ''
                }`}
                onClick={() => setFilter(item)}
              >
                <span>{t(`model_prices.filter_${item}`)}</span>
                <strong>{filterCounts[item]}</strong>
              </button>
            ))}
          </div>
        </div>

        {syncResult?.sourceResults?.length ? (
          <div className={styles.sourceResults}>
            {syncResult.sourceResults.map((result) => (
              <span
                key={result.source}
                className={result.error ? styles.sourceResultError : undefined}
                title={result.error || undefined}
              >
                <strong>{result.source}</strong>
                {result.error
                  ? t('model_prices.source_result_failed')
                  : t('model_prices.source_result_ok', {
                      models: result.models,
                      skipped: result.skipped,
                    })}
              </span>
            ))}
          </div>
        ) : null}

        {loading ? (
          <div className={styles.emptyState}>{t('common.loading')}</div>
        ) : visibleRows.length === 0 ? (
          <div className={styles.emptyState}>{t('model_prices.empty')}</div>
        ) : (
          <div className={styles.tableWrap}>
            <table className={styles.priceTable}>
              <thead>
                <tr>
                  <th>{t('usage_stats.model_name')}</th>
                  <th>{t('model_prices.calls')}</th>
                  <th>{t('usage_stats.model_price_prompt')}</th>
                  <th>{t('usage_stats.model_price_completion')}</th>
                  <th>{t('usage_stats.model_price_cache')}</th>
                  <th>{t('model_prices.source')}</th>
                  <th>{t('common.action')}</th>
                </tr>
              </thead>
              <tbody>
                {visibleRows.map((row) => {
                  const candidates =
                    candidateSets.find((candidateSet) => candidateSet.model === row.model)
                      ?.candidates ?? [];
                  const selectedSource =
                    selectedCandidates[row.model] || candidates[0]?.sourceModelId || '';
                  const selectedCandidate =
                    candidates.find((candidate) => candidate.sourceModelId === selectedSource) ??
                    candidates[0];

                  return (
                    <tr key={row.model}>
                      <td className={styles.modelCell}>
                        <div className={styles.modelContent}>
                          <strong>{row.model}</strong>
                          {!row.hasPrice && candidates.length > 0 ? (
                            <span>{t('model_prices.needs_confirmation')}</span>
                          ) : !row.hasPrice ? (
                            <span>{t('model_prices.no_price')}</span>
                          ) : null}
                          {row.aliases.length > 0 ? (
                            <small>
                              {t('model_prices.aliases')}: {row.aliases.join(', ')}
                            </small>
                          ) : null}
                        </div>
                      </td>
                      <td>{row.calls}</td>
                      <td>{formatPriceUnit(row.price?.prompt)}</td>
                      <td>{formatPriceUnit(row.price?.completion)}</td>
                      <td>{formatPriceUnit(row.price?.cache)}</td>
                      <td className={styles.sourceCell}>
                        {row.price ? (
                          <div className={styles.sourceContent}>
                            <span className={styles.sourceBadge}>
                              {row.price.source || 'manual'}
                            </span>
                            {row.price.sourceModelId ? (
                              <small>{row.price.sourceModelId}</small>
                            ) : null}
                          </div>
                        ) : selectedCandidate ? (
                          <div className={styles.sourceContent}>
                            <span className={styles.sourceBadge}>
                              {selectedCandidate.price.source || 'sync'}
                            </span>
                            <small>{selectedCandidate.sourceModelId}</small>
                          </div>
                        ) : (
                          <span className={styles.sourcePlaceholder}>--</span>
                        )}
                      </td>
                      <td
                        className={`${styles.actionsCell} ${
                          !row.hasPrice && candidates.length > 0 ? styles.candidateActionsCell : ''
                        }`}
                      >
                        <div className={styles.rowActions}>
                          {row.hasPrice ? (
                            <>
                              <button
                                type="button"
                                className={styles.iconAction}
                                title={t('common.edit')}
                                aria-label={t('common.edit')}
                                onClick={() => openManualEditor(row.model, row.price)}
                              >
                                <IconPencil size={14} />
                              </button>
                              <button
                                type="button"
                                className={styles.iconAction}
                                title={t('common.delete')}
                                aria-label={t('common.delete')}
                                onClick={() => void handleDelete(row.model)}
                              >
                                <IconTrash2 size={14} />
                              </button>
                            </>
                          ) : candidates.length > 0 && selectedCandidate ? (
                            <div className={styles.candidateControl}>
                              <select
                                value={selectedSource}
                                onChange={(event) =>
                                  setSelectedCandidates((previous) => ({
                                    ...previous,
                                    [row.model]: event.target.value,
                                  }))
                                }
                                aria-label={t('model_prices.candidate_select')}
                              >
                                {candidates.map((candidate) => (
                                  <option
                                    key={candidate.sourceModelId}
                                    value={candidate.sourceModelId}
                                  >
                                    {`${candidate.price.source || 'sync'} · ${candidate.sourceModelId} · ${Math.round(candidate.score * 100)}%`}
                                  </option>
                                ))}
                              </select>
                              <button
                                type="button"
                                className={styles.confirmButton}
                                onClick={() =>
                                  void handleConfirmCandidate(row.model, selectedCandidate)
                                }
                              >
                                <span>{t('model_prices.confirm_candidate')}</span>
                              </button>
                            </div>
                          ) : (
                            <button
                              type="button"
                              className={styles.confirmButton}
                              onClick={() => openManualEditor(row.model)}
                            >
                              <span>{t('model_prices.add_manual')}</span>
                            </button>
                          )}
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}

        {syncResult?.unmatched?.length ? (
          <div className={styles.unmatchedBlock}>
            <strong>{t('model_prices.unmatched_title')}</strong>
            <span>{syncResult.unmatched.join(', ')}</span>
          </div>
        ) : null}
      </section>

      <Modal
        open={manualEditorOpen}
        onClose={closeManualEditor}
        title={
          modelPrices[draft.model.trim()]
            ? t('model_prices.edit_manual_title', { defaultValue: '编辑模型价格' })
            : t('model_prices.add_manual')
        }
        width={680}
        className={styles.manualEditorModal}
        footer={
          <div className={styles.manualEditorActions}>
            <Button variant="secondary" size="sm" onClick={closeManualEditor}>
              {t('common.cancel')}
            </Button>
            <Button size="sm" onClick={() => void handleSaveDraft()}>
              {t('common.save')}
            </Button>
          </div>
        }
      >
        <div className={styles.manualEditorBody}>
          <div className={styles.manualEditorGrid}>
            <div className={styles.manualEditorColumn}>
              <Input
                label={t('usage_stats.model_name')}
                className={styles.compactInput}
                value={draft.model}
                onChange={(event) => setDraftField('model', event.target.value)}
                placeholder="gpt-5.5"
              />
              <p className={styles.manualEditorHint}>{t('model_prices.manual_hint')}</p>
            </div>
            <div className={styles.manualEditorColumn}>
              <Input
                label={`${t('usage_stats.model_price_prompt')} ($/1M)`}
                className={styles.compactInput}
                type="number"
                value={draft.prompt}
                onChange={(event) => setDraftField('prompt', event.target.value)}
                placeholder="0.0000"
                step="0.0001"
              />
              <Input
                label={`${t('usage_stats.model_price_completion')} ($/1M)`}
                className={styles.compactInput}
                type="number"
                value={draft.completion}
                onChange={(event) => setDraftField('completion', event.target.value)}
                placeholder="0.0000"
                step="0.0001"
              />
              <Input
                label={`${t('usage_stats.model_price_cache')} ($/1M)`}
                className={styles.compactInput}
                type="number"
                value={draft.cache}
                onChange={(event) => setDraftField('cache', event.target.value)}
                placeholder="0.0000"
                step="0.0001"
              />
            </div>
          </div>
        </div>
      </Modal>
    </div>
  );
}
