import { useId, type ReactNode } from 'react';
import type { TFunction } from 'i18next';
import { Button } from '@/components/ui/Button';
import { IconCopy, IconFilter } from '@/components/ui/icons';
import {
  PaginationControls,
  RecentPattern,
} from '@/features/monitoring/components/MonitoringShared';
import { MonitoringPanel } from '@/features/monitoring/components/MonitoringPanel';
import { formatPercent } from '@/features/monitoring/components/accountOverviewPresentation';
import { buildRealtimeSourceDisplay } from '@/features/monitoring/realtimeSourceDisplay';
import type { MonitoringEventRow } from '@/features/monitoring/hooks/useMonitoringData';
import { useNotificationStore } from '@/stores/useNotificationStore';
import { copyToClipboard } from '@/utils/clipboard';
import { maskSensitiveText, truncateText } from '@/utils/format';
import { formatCompactNumber, formatUsd } from '@/utils/usage';
import styles from '../MonitoringCenterPage.module.scss';

type RealtimeLogRow = MonitoringEventRow & {
  requestCount: number;
  successRate: number;
  streamKey: string;
  recentPattern: boolean[];
};

type PaginationState<T> = {
  currentPage: number;
  totalPages: number;
  pageItems: T[];
  startItem: number;
  endItem: number;
};

type RealtimeEventsPanelProps = {
  embedded?: boolean;
  rows: RealtimeLogRow[];
  pagination: PaginationState<RealtimeLogRow>;
  pageSize: number;
  scopedFailureCount: number;
  failedOnlyActive: boolean;
  eventsHasMore: boolean;
  eventsLoadingMore: boolean;
  overallLoading: boolean;
  hasPrices: boolean;
  locale: string;
  emptyState: ReactNode;
  t: TFunction;
  onToggleFailedOnly: () => void;
  onPageChange: (page: number) => void;
  onPageSizeChange: (pageSize: number) => void;
  onLoadMoreEvents: () => void;
};

export type RealtimeEventsPanelActionsProps = {
  rowCount: number;
  scopedFailureCount: number;
  failedOnlyActive: boolean;
  t: TFunction;
  onToggleFailedOnly: () => void;
};

const REALTIME_PAGE_SIZE_OPTIONS = [10, 50, 100, 150, 300] as const;

const formatOptionalText = (value: string | null | undefined) => {
  const trimmed = String(value || '').trim();
  return trimmed || '-';
};

const formatReadableText = (value: string | null | undefined) => {
  const trimmed = String(value || '').trim();
  return trimmed && trimmed !== '-' ? trimmed : '';
};

const formatShortHash = (value: string | null | undefined) => {
  const trimmed = formatReadableText(value);
  return trimmed ? `#${trimmed.slice(0, 8)}` : '';
};

const buildRealtimeApiKeyDisplay = (row: MonitoringEventRow, t: TFunction) => {
  const label = formatReadableText(row.apiKeyLabel);
  const masked = formatReadableText(row.apiKeyMasked);
  const hash = formatReadableText(row.apiKeyHash);
  const shortHash = formatShortHash(hash);
  const display = label || masked || shortHash;

  if (!display) {
    return null;
  }

  const titleParts = [
    `${t('monitoring.realtime_api_key_label')}: ${display}`,
    masked && masked !== display ? `${t('monitoring.realtime_api_key_masked')}: ${masked}` : '',
    hash ? `${t('monitoring.realtime_api_key_hash')}: ${hash}` : '',
    formatReadableText(row.executorType)
      ? `${t('monitoring.executor_type_short')}: ${formatReadableText(row.executorType)}`
      : '',
  ].filter(Boolean);

  return {
    display,
    title: titleParts.join('\n'),
  };
};

const formatTokensPerSecond = (value: number | null | undefined, locale: string) => {
  if (typeof value !== 'number' || !Number.isFinite(value) || value <= 0) return '--';

  const absValue = Math.abs(value);
  const maximumFractionDigits = absValue < 1 ? 2 : absValue < 10 ? 1 : 0;
  try {
    return new Intl.NumberFormat(locale, {
      maximumFractionDigits,
      minimumFractionDigits: 0,
    }).format(value);
  } catch {
    return value.toFixed(maximumFractionDigits);
  }
};

const formatRealtimeCompactDuration = (value: number | null | undefined, locale: string) => {
  if (value === null || value === undefined) return '--';

  const parsed = Number(value);
  if (!Number.isFinite(parsed) || parsed < 0) return '--';

  const formatNumber = (numberValue: number, maximumFractionDigits: number) => {
    try {
      return new Intl.NumberFormat(locale, {
        maximumFractionDigits,
        minimumFractionDigits: 0,
      }).format(numberValue);
    } catch {
      return numberValue.toFixed(maximumFractionDigits);
    }
  };

  if (parsed < 1000) return `${formatNumber(Math.round(parsed), 0)} ms`;

  const seconds = parsed / 1000;
  return `${formatNumber(seconds, seconds < 10 ? 2 : 1)} s`;
};

const getRealtimeDurationToneClass = (value: number | null | undefined) => {
  if (value === null || value === undefined) return undefined;

  const parsed = Number(value);
  if (!Number.isFinite(parsed) || parsed < 0) return undefined;

  if (parsed >= 30000) return styles.badText;
  if (parsed >= 15000) return styles.warnText;
  return styles.goodText;
};

const formatRealtimeDateParts = (timestampMs: number, locale: string) => {
  const date = new Date(timestampMs);
  return {
    date: date.toLocaleDateString(locale, {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
    }),
    time: date.toLocaleTimeString(locale, {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false,
    }),
  };
};

const buildFailureMetaText = (row: MonitoringEventRow, t: TFunction) => {
  if (!row.failed) return '';
  const parts: string[] = [];
  if (row.failStatusCode) {
    parts.push(`${t('monitoring.fail_status_code_short')} ${row.failStatusCode}`);
  }
  const body = maskSensitiveText(row.failSummary || '');
  if (body) {
    parts.push(truncateText(body, 96));
  }
  return parts.join(' · ');
};

const buildFailureDetails = (row: MonitoringEventRow, t: TFunction) => {
  if (!row.failed) return null;
  const summary = maskSensitiveText(row.failSummary || '');
  if (!row.failStatusCode && !summary) return null;
  const statusText = row.failStatusCode
    ? `${t('monitoring.fail_status_code_short')} ${row.failStatusCode}`
    : '';
  return {
    statusCode: row.failStatusCode,
    statusText,
    summary,
    label: buildFailureMetaText(row, t),
    copyText: [statusText, summary].filter(Boolean).join('\n'),
  };
};

export function RealtimeEventsPanelActions({
  rowCount,
  scopedFailureCount,
  failedOnlyActive,
  t,
  onToggleFailedOnly,
}: RealtimeEventsPanelActionsProps) {
  return (
    <div className={`${styles.inlineMetrics} ${styles.realtimeHeaderActions}`}>
      <span>{`${t('monitoring.log_rows')}: ${rowCount}`}</span>
      <span>{`${t('monitoring.recent_failures')}: ${scopedFailureCount}`}</span>
      <button
        type="button"
        className={[styles.filterToggleChip, failedOnlyActive ? styles.filterToggleChipActive : '']
          .filter(Boolean)
          .join(' ')}
        onClick={onToggleFailedOnly}
      >
        <IconFilter size={14} aria-hidden="true" />
        {t('monitoring.filter_status_failed')}
      </button>
    </div>
  );
}

export function RealtimeEventsPanel({
  embedded = false,
  rows,
  pagination,
  pageSize,
  scopedFailureCount,
  failedOnlyActive,
  eventsHasMore,
  eventsLoadingMore,
  overallLoading,
  hasPrices,
  locale,
  emptyState,
  t,
  onToggleFailedOnly,
  onPageChange,
  onPageSizeChange,
  onLoadMoreEvents,
}: RealtimeEventsPanelProps) {
  const tooltipIdPrefix = useId();
  const showNotification = useNotificationStore((state) => state.showNotification);
  const handleCopyFailureDetails = async (text: string) => {
    const copied = await copyToClipboard(text);
    showNotification(
      t(copied ? 'notification.link_copied' : 'notification.copy_failed'),
      copied ? 'success' : 'error'
    );
  };
  const actions = (
    <RealtimeEventsPanelActions
      rowCount={rows.length}
      scopedFailureCount={scopedFailureCount}
      failedOnlyActive={failedOnlyActive}
      t={t}
      onToggleFailedOnly={onToggleFailedOnly}
    />
  );
  const content = (
    <>
      <div className={styles.tableWrapper}>
        <table className={`${styles.table} ${styles.realtimeTable}`}>
          <thead>
            <tr>
              <th>{t('monitoring.column_source_api_key')}</th>
              <th>{t('monitoring.column_model')}</th>
              <th>{t('monitoring.reasoning_effort')}</th>
              <th>{t('monitoring.recent_status')}</th>
              <th>{t('monitoring.request_status')}</th>
              <th>{t('monitoring.column_success_rate')}</th>
              <th>{t('monitoring.total_calls')}</th>
              <th className={styles.realtimeTpsColumn}>{t('monitoring.column_output_tps')}</th>
              <th className={styles.realtimeLatencyColumn}>
                <span className={styles.realtimeLatencyHeader}>
                  <span className={styles.realtimeMetricLeft}>
                    {t('monitoring.ttft_short')}
                  </span>
                  <span className={styles.realtimeMetricSeparator}>｜</span>
                  <span className={styles.realtimeMetricRight}>
                    {t('monitoring.elapsed_short')}
                  </span>
                </span>
              </th>
              <th>{t('monitoring.column_time')}</th>
              <th>{t('monitoring.this_call_usage')}</th>
              <th>{t('monitoring.this_call_cost')}</th>
            </tr>
          </thead>
          <tbody>
            {pagination.pageItems.map((row) => {
              const sourceDisplay = buildRealtimeSourceDisplay(row, t);
              const apiKeyDisplay = buildRealtimeApiKeyDisplay(row, t);
              const showResolvedModel =
                row.resolvedModel &&
                row.resolvedModel.trim() &&
                row.resolvedModel.trim() !== row.model;
              const reasoningEffort = formatOptionalText(row.reasoningEffort);
              const serviceTier = formatOptionalText(row.serviceTier);
              const failureDetails = buildFailureDetails(row, t);
              const failureTooltipId = failureDetails
                ? `${tooltipIdPrefix}-failure-tooltip-${row.id}`
                : undefined;
              const timeParts = formatRealtimeDateParts(row.timestampMs, locale);
              const hasTtftMs = row.ttftMs !== null && row.ttftMs !== undefined;
              const ttftToneClass = getRealtimeDurationToneClass(row.ttftMs);
              const latencyToneClass = getRealtimeDurationToneClass(row.latencyMs);
              return (
                <tr key={row.id} className={row.failed ? styles.logRowFailed : undefined}>
                  <td>
                    <div className={styles.logTypeCell}>
                      <div className={styles.primaryCell}>
                        <span>{sourceDisplay.primary}</span>
                        {sourceDisplay.meta ? <small>{sourceDisplay.meta}</small> : null}
                        {apiKeyDisplay ? (
                          <small className={styles.realtimeApiKeyLine} title={apiKeyDisplay.title}>
                            {`${t('monitoring.realtime_api_key_label')}: ${apiKeyDisplay.display}`}
                          </small>
                        ) : null}
                      </div>
                    </div>
                  </td>
                  <td>
                    <div className={styles.primaryCell}>
                      <span className={styles.monoCell}>{row.model}</span>
                      {showResolvedModel ? (
                        <small className={styles.monoCell}>{row.resolvedModel}</small>
                      ) : null}
                    </div>
                  </td>
                  <td>
                    <div className={styles.primaryCell}>
                      {reasoningEffort !== '-' ? (
                        <span className={styles.realtimeReasoningBadge}>{reasoningEffort}</span>
                      ) : (
                        <span className={styles.mutedCell}>-</span>
                      )}
                      {serviceTier !== '-' ? (
                        <small>{`${t('monitoring.service_tier_short')}: ${serviceTier}`}</small>
                      ) : null}
                    </div>
                  </td>
                  <td>
                    <div className={styles.recentStatusCell}>
                      <RecentPattern pattern={row.recentPattern} variant="plain" />
                    </div>
                  </td>
                  <td>
                    <div className={styles.primaryCell}>
                      {failureDetails ? (
                        <span
                          className={styles.realtimeFailureStatus}
                          tabIndex={0}
                          aria-describedby={failureTooltipId}
                          aria-label={failureDetails.label}
                        >
                          <span
                            className={`${styles.realtimeRequestStatus} ${styles.realtimeRequestStatusBad}`}
                          >
                            {t('monitoring.result_failed')}
                          </span>
                          <span
                            id={failureTooltipId}
                            role="tooltip"
                            className={styles.realtimeFailureTooltip}
                          >
                            <button
                              type="button"
                              className={styles.realtimeFailureCopyButton}
                              onClick={(event) => {
                                event.preventDefault();
                                event.stopPropagation();
                                void handleCopyFailureDetails(failureDetails.copyText);
                              }}
                              title={t('common.copy')}
                              aria-label={t('common.copy')}
                            >
                              <IconCopy size={13} />
                            </button>
                            {failureDetails.statusCode ? (
                              <span className={styles.realtimeFailureTooltipStatus}>
                                {failureDetails.statusText}
                              </span>
                            ) : null}
                            {failureDetails.summary ? (
                              <span className={styles.realtimeFailureTooltipBody}>
                                {failureDetails.summary}
                              </span>
                            ) : null}
                          </span>
                        </span>
                      ) : (
                        <span
                          className={[
                            styles.realtimeRequestStatus,
                            row.failed
                              ? styles.realtimeRequestStatusBad
                              : styles.realtimeRequestStatusGood,
                          ]
                            .filter(Boolean)
                            .join(' ')}
                        >
                          {row.failed
                            ? t('monitoring.result_failed')
                            : t('monitoring.result_success')}
                        </span>
                      )}
                    </div>
                  </td>
                  <td
                    className={
                      row.successRate >= 0.95
                        ? styles.goodText
                        : row.successRate >= 0.85
                          ? styles.warnText
                          : styles.badText
                    }
                  >
                    {formatPercent(row.successRate)}
                  </td>
                  <td>{formatCompactNumber(row.requestCount)}</td>
                  <td className={styles.realtimeTpsColumn}>
                    <span className={styles.realtimeTpsCell}>
                      {formatTokensPerSecond(row.tokensPerSecond, locale)}
                    </span>
                  </td>
                  <td className={styles.realtimeLatencyColumn}>
                    <div className={styles.realtimeMetricCell}>
                      <span
                        className={[
                          styles.realtimeMetricText,
                          styles.realtimeMetricLeft,
                          ttftToneClass,
                        ]
                          .filter(Boolean)
                          .join(' ')}
                      >
                        {hasTtftMs ? formatRealtimeCompactDuration(row.ttftMs, locale) : '--'}
                      </span>
                      <span className={styles.realtimeMetricSeparator}>｜</span>
                      <span
                        className={[
                          styles.realtimeMetricText,
                          styles.realtimeMetricRight,
                          latencyToneClass,
                        ]
                          .filter(Boolean)
                          .join(' ')}
                      >
                        {formatRealtimeCompactDuration(row.latencyMs, locale)}
                      </span>
                    </div>
                  </td>
                  <td>
                    <div className={styles.realtimeTimeCell}>
                      <span className={styles.realtimeTimeLine}>{timeParts.date}</span>
                      <span className={styles.realtimeTimeLine}>{timeParts.time}</span>
                    </div>
                  </td>
                  <td>
                    <div className={styles.primaryCell}>
                      <span>{formatCompactNumber(row.totalTokens)}</span>
                      <small>{`I ${formatCompactNumber(row.inputTokens)} · O ${formatCompactNumber(row.outputTokens)} · C ${formatCompactNumber(row.cachedTokens)}`}</small>
                    </div>
                  </td>
                  <td>{hasPrices ? formatUsd(row.totalCost) : '--'}</td>
                </tr>
              );
            })}
            {rows.length === 0 ? (
              <tr>
                <td colSpan={12}>{emptyState}</td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </div>
      <PaginationControls
        count={rows.length}
        currentPage={pagination.currentPage}
        totalPages={pagination.totalPages}
        startItem={pagination.startItem}
        endItem={pagination.endItem}
        pageSize={pageSize}
        pageSizeOptions={REALTIME_PAGE_SIZE_OPTIONS}
        onPageChange={onPageChange}
        onPageSizeChange={onPageSizeChange}
        t={t}
      />
      {rows.length > 0 ? (
        <div className={styles.loadMoreEventsBar}>
          {eventsHasMore ? (
            <Button
              variant="secondary"
              size="sm"
              onClick={onLoadMoreEvents}
              disabled={eventsLoadingMore || overallLoading}
            >
              {eventsLoadingMore ? t('common.loading') : t('monitoring.load_more_events')}
            </Button>
          ) : (
            <span>{t('monitoring.no_more_events')}</span>
          )}
        </div>
      ) : null}
    </>
  );

  if (embedded) {
    return content;
  }

  return (
    <MonitoringPanel
      title={t('monitoring.realtime_table_title')}
      subtitle={t('monitoring.realtime_table_desc')}
      className={styles.realtimePanel}
      extra={actions}
    >
      {content}
    </MonitoringPanel>
  );
}
