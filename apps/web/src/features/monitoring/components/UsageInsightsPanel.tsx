import { useMemo, type CSSProperties, type ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import {
  IconChartLine,
  IconDollarSign,
  IconInbox,
  IconTimer,
  IconTrendingUp,
  IconX,
} from '@/components/ui/icons';
import type {
  MonitoringModelShareRow,
  MonitoringSummary,
  MonitoringTimelinePoint,
} from '@/features/monitoring/hooks/useMonitoringData';
import { formatCompactNumber, formatDurationMs, formatUsd } from '@/utils/usage';
import styles from './UsageInsightsPanel.module.scss';

type UsageInsightsPanelProps = {
  summary: MonitoringSummary;
  timeline: MonitoringTimelinePoint[];
  modelShareRows: MonitoringModelShareRow[];
  loading: boolean;
  locale: string;
};

type MetricCard = {
  key: string;
  label: string;
  value: string;
  meta: string;
  tone: 'clay' | 'sage' | 'ochre' | 'brick' | 'umber';
  icon: ReactNode;
};

type TrafficGridStyle = CSSProperties & Record<'--bucket-count', number>;
type TrafficBarStyle = CSSProperties &
  Record<'--metric-share' | '--metric-min-height', number | string>;
type TokenStyle = CSSProperties & Record<'--token-share', string>;
type RankStyle = CSSProperties & Record<'--rank-share', string>;

type RankedModelCostRow = MonitoringModelShareRow & {
  costShare: number;
  rankShare: number;
  shareLabel: string;
};

const formatPercent = (value: number) => `${(value * 100).toFixed(2)}%`;

const clampShare = (value: number | undefined) => {
  if (value === undefined || !Number.isFinite(value)) return 0;
  return Math.max(0, Math.min(1, value));
};

const formatCoord = (value: number) =>
  Number.isInteger(value) ? String(value) : value.toFixed(3).replace(/\.?0+$/, '');

const buildTrafficAxisTickIndexes = (pointCount: number, maxTicks = 6) => {
  if (pointCount <= 0) return [];
  if (pointCount <= maxTicks) return Array.from({ length: pointCount }, (_, index) => index);

  const tickCount = Math.max(2, maxTicks);
  const indexes = new Set<number>();
  for (let index = 0; index < tickCount; index += 1) {
    indexes.add(Math.round((index * (pointCount - 1)) / (tickCount - 1)));
  }
  return Array.from(indexes).sort((left, right) => left - right);
};

const getTrafficPointX = (index: number, pointCount: number) =>
  pointCount <= 0 ? 0 : ((index + 0.5) / pointCount) * 100;

const getTimelineShare = (
  point: MonitoringTimelinePoint,
  metric: 'requests' | 'tokens',
  maxRequests: number,
  maxTokens: number
) => {
  if (metric === 'requests') {
    return clampShare(point.requestsShare ?? (maxRequests > 0 ? point.requests / maxRequests : 0));
  }
  return clampShare(point.tokensShare ?? (maxTokens > 0 ? point.tokens / maxTokens : 0));
};

const buildCallsLinePath = (
  timeline: MonitoringTimelinePoint[],
  maxRequests: number,
  maxTokens: number
) =>
  timeline
    .map((point, index) => {
      const x = getTrafficPointX(index, timeline.length);
      const y = 100 - getTimelineShare(point, 'requests', maxRequests, maxTokens) * 100;
      return `${index === 0 ? 'M' : 'L'} ${formatCoord(x)} ${formatCoord(y)}`;
    })
    .join(' ');

const getTimelineKey = (point: MonitoringTimelinePoint, index: number) =>
  point.bucketMs ?? `${point.label}-${index}`;

const sortModelCostRows = (left: MonitoringModelShareRow, right: MonitoringModelShareRow) =>
  right.totalCost - left.totalCost ||
  right.totalTokens - left.totalTokens ||
  right.requests - left.requests ||
  left.model.localeCompare(right.model);

export function UsageInsightsPanel({
  summary,
  timeline,
  modelShareRows,
  loading,
  locale,
}: UsageInsightsPanelProps) {
  const { t } = useTranslation();
  const maxTimelineRequests = useMemo(
    () => Math.max(...timeline.map((item) => item.requests), 0),
    [timeline]
  );
  const maxTimelineTokens = useMemo(
    () => Math.max(...timeline.map((item) => item.tokens), 0),
    [timeline]
  );
  const hasTrafficData = timeline.some((point) => point.requests > 0 || point.tokens > 0);
  const trafficGridStyle = {
    '--bucket-count': Math.max(timeline.length, 1),
  } as TrafficGridStyle;
  const trafficAxisTickIndexes = useMemo(
    () => buildTrafficAxisTickIndexes(timeline.length),
    [timeline.length]
  );
  const callsLinePath = useMemo(
    () => buildCallsLinePath(timeline, maxTimelineRequests, maxTimelineTokens),
    [maxTimelineRequests, maxTimelineTokens, timeline]
  );
  const tokenTotal = Math.max(summary.totalTokens, 0);
  const tokenSegments = useMemo(
    () => [
      {
        key: 'input',
        label: t('dashboard.token_mix_input'),
        value: summary.inputTokens,
        tone: 'clay',
      },
      {
        key: 'output',
        label: t('dashboard.token_mix_output'),
        value: summary.outputTokens,
        tone: 'sage',
      },
      {
        key: 'reasoning',
        label: t('dashboard.token_mix_reasoning'),
        value: summary.reasoningTokens,
        tone: 'ochre',
      },
      {
        key: 'cached',
        label: t('dashboard.token_mix_cached'),
        value: summary.cachedTokens + summary.cacheReadTokens + summary.cacheCreationTokens,
        tone: 'umber',
      },
    ],
    [summary, t]
  );

  const modelCostRank = useMemo<RankedModelCostRow[]>(() => {
    const totalCostSum = modelShareRows.reduce(
      (sum, row) => sum + Math.max(row.totalCost, 0),
      0
    );
    const totalTokenSum = modelShareRows.reduce(
      (sum, row) => sum + Math.max(row.totalTokens, 0),
      0
    );
    const totalRequestSum = modelShareRows.reduce(
      (sum, row) => sum + Math.max(row.requests, 0),
      0
    );

    return [...modelShareRows]
      .sort(sortModelCostRows)
      .slice(0, 5)
      .map((row) => {
        const costShare = totalCostSum > 0 ? Math.max(row.totalCost, 0) / totalCostSum : 0;
        const tokenShare = totalTokenSum > 0 ? Math.max(row.totalTokens, 0) / totalTokenSum : 0;
        const requestShare =
          totalRequestSum > 0 ? Math.max(row.requests, 0) / totalRequestSum : 0;
        const rankShare = totalCostSum > 0 ? costShare : totalTokenSum > 0 ? tokenShare : requestShare;
        const shareLabel =
          totalCostSum > 0
            ? `${(costShare * 100).toFixed(1)}%`
            : totalTokenSum > 0
              ? `${(tokenShare * 100).toFixed(1)}% Token`
              : `${(requestShare * 100).toFixed(1)}% ${t('dashboard.traffic_calls')}`;

        return {
          ...row,
          costShare,
          rankShare,
          shareLabel,
        };
      });
  }, [modelShareRows, t]);

  const cachedTokenTotal = summary.cachedTokens + summary.cacheReadTokens + summary.cacheCreationTokens;
  const failureRate = summary.totalCalls > 0 ? summary.failureCalls / summary.totalCalls : 0;
  const metrics: MetricCard[] = [
    {
      key: 'totalRequests',
      label: t('usage_stats.total_requests'),
      value: formatCompactNumber(summary.totalCalls),
      meta: `${t('monitoring.success_calls')}: ${formatCompactNumber(summary.successCalls)}`,
      tone: 'clay',
      icon: <IconInbox size={18} />,
    },
    {
      key: 'successRate',
      label: t('dashboard.success_rate'),
      value: formatPercent(summary.successRate),
      meta: `${formatCompactNumber(summary.successCalls)} / ${formatCompactNumber(summary.totalCalls)}`,
      tone: 'sage',
      icon: <IconTrendingUp size={18} />,
    },
    {
      key: 'failureCalls',
      label: t('usage_page.failure_amount', { defaultValue: '失败量' }),
      value: formatCompactNumber(summary.failureCalls),
      meta: `${formatCompactNumber(summary.failureCalls)} / ${formatCompactNumber(summary.totalCalls)}`,
      tone: 'brick',
      icon: <IconX size={18} />,
    },
    {
      key: 'failureRate',
      label: t('usage_page.failure_rate', { defaultValue: '失败率' }),
      value: formatPercent(failureRate),
      meta: `${t('usage_page.failure_amount', { defaultValue: '失败量' })}: ${formatCompactNumber(summary.failureCalls)}`,
      tone: 'brick',
      icon: <IconTrendingUp size={18} />,
    },
    {
      key: 'avgLatency',
      label: t('dashboard.avg_latency'),
      value: formatDurationMs(summary.averageLatencyMs, { locale }),
      meta: `${t('monitoring.zero_token_model_calls')}: ${formatCompactNumber(summary.zeroTokenCalls)}`,
      tone: 'ochre',
      icon: <IconTimer size={18} />,
    },
    {
      key: 'totalCost',
      label: t('monitoring.estimated_cost'),
      value: formatUsd(summary.totalCost),
      meta: modelShareRows.length
        ? t('model_prices.sync_model_count', { count: modelShareRows.length })
        : t('dashboard.no_usage_rank_data'),
      tone: 'umber',
      icon: <IconDollarSign size={18} />,
    },
    {
      key: 'totalTokens',
      label: t('dashboard.total_tokens'),
      value: formatCompactNumber(summary.totalTokens),
      meta: t('dashboard.metric_zero_token_calls', { value: formatCompactNumber(summary.zeroTokenCalls) }),
      tone: 'clay',
      icon: <IconChartLine size={18} />,
    },
    {
      key: 'inputTokens',
      label: t('monitoring.input_tokens'),
      value: formatCompactNumber(summary.inputTokens),
      meta: `${t('usage_page.token_mix', { defaultValue: 'Token 构成' })}: ${tokenTotal > 0 ? ((summary.inputTokens / tokenTotal) * 100).toFixed(1) : '0.0'}%`,
      tone: 'clay',
      icon: <IconChartLine size={18} />,
    },
    {
      key: 'outputTokens',
      label: t('monitoring.output_tokens'),
      value: formatCompactNumber(summary.outputTokens),
      meta: `${t('usage_page.token_mix', { defaultValue: 'Token 构成' })}: ${tokenTotal > 0 ? ((summary.outputTokens / tokenTotal) * 100).toFixed(1) : '0.0'}%`,
      tone: 'sage',
      icon: <IconTrendingUp size={18} />,
    },
    {
      key: 'cachedTokens',
      label: t('monitoring.cached_tokens'),
      value: formatCompactNumber(cachedTokenTotal),
      meta: `${t('monitoring.cache_reuse_rate')}: ${tokenTotal > 0 ? ((cachedTokenTotal / tokenTotal) * 100).toFixed(1) : '0.0'}%`,
      tone: 'umber',
      icon: <IconInbox size={18} />,
    },
  ];

  return (
    <section className={styles.panel} aria-label={t('nav.usage')}>
      <div className={styles.metricsGrid}>
        {metrics.map((metric) => (
          <article key={metric.key} className={`${styles.metricCard} ${styles[metric.tone]}`}>
            <div className={styles.metricHeader}>
              <span className={styles.metricIcon}>{metric.icon}</span>
              <span>{metric.label}</span>
            </div>
            <strong>{loading ? '...' : metric.value}</strong>
            <small>{metric.meta}</small>
          </article>
        ))}
      </div>

      <div className={styles.chartsGrid}>
        <article className={styles.chartCard}>
          <div className={styles.cardHeader}>
            <div>
              <h2>{t('usage_page.request_health', { defaultValue: '请求健康' })}</h2>
            </div>
          </div>
          <div className={styles.healthGrid}>
            <div className={styles.healthMetric}>
              <span>{t('dashboard.success_rate')}</span>
              <strong className={styles.healthGood}>{formatPercent(summary.successRate)}</strong>
            </div>
            <div className={styles.healthMetric}>
              <span>{t('usage_page.failure_rate', { defaultValue: '失败率' })}</span>
              <strong className={styles.healthBad}>{formatPercent(failureRate)}</strong>
            </div>
            <div className={styles.healthMetric}>
              <span>{t('dashboard.avg_latency')}</span>
              <strong className={styles.healthLatency}>{formatDurationMs(summary.averageLatencyMs, { locale })}</strong>
            </div>
            <div className={styles.healthMetric}>
              <span>{t('usage_page.first_token', { defaultValue: '首字' })}</span>
              <strong className={styles.healthFirstToken}>{formatDurationMs(summary.averageTTFTMs, { locale })}</strong>
            </div>
          </div>
        </article>

        <article className={styles.chartCard}>
          <div className={styles.cardHeader}>
            <div>
              <h2>{t('usage_page.token_mix', { defaultValue: 'Token 构成' })}</h2>
              <p>{t('usage_page.token_desc', { defaultValue: '输入、输出、推理与缓存 Token 的构成。' })}</p>
            </div>
          </div>
          <div className={styles.tokenList}>
            {tokenSegments.map((segment) => {
              const share = tokenTotal > 0 ? segment.value / tokenTotal : 0;
              return (
                <div key={segment.key} className={styles.tokenRow}>
                  <div>
                    <span>{segment.label}</span>
                    <strong>{formatCompactNumber(segment.value)}</strong>
                  </div>
                  <span className={`${styles.tokenTrack} ${styles[segment.tone]}`}>
                    <i style={{ '--token-share': `${Math.max(share, 0)}` } as TokenStyle} />
                  </span>
                </div>
              );
            })}
          </div>
        </article>

        <article className={styles.chartCard}>
          <div className={styles.cardHeader}>
            <div>
              <h2>{t('dashboard.model_cost_rank')}</h2>
              <p>{t('usage_page.model_cost_rank_desc', { defaultValue: '按当前筛选范围的估算成本排序。' })}</p>
            </div>
          </div>
          <div className={styles.rankList}>
            {modelCostRank.length > 0 ? (
              modelCostRank.map((model, index) => (
                <div key={model.model} className={styles.rankItem}>
                  <div className={styles.rankIndex} data-rank={index + 1}>
                    {index + 1}
                  </div>
                  <div className={styles.rankInfo}>
                    <div className={styles.modelName}>{model.model}</div>
                    <div className={styles.rankTrack}>
                      <div
                        className={styles.rankBar}
                        style={{ '--rank-share': `${clampShare(model.rankShare)}` } as RankStyle}
                      />
                    </div>
                    <div className={styles.rankMeta}>
                      {formatCompactNumber(model.totalTokens)} Token · {formatCompactNumber(model.requests)}{' '}
                      {t('dashboard.traffic_calls')}
                    </div>
                  </div>
                  <div className={styles.rankValue}>
                    <div className={styles.cost}>{formatUsd(model.totalCost)}</div>
                    <div className={styles.share}>{model.shareLabel}</div>
                  </div>
                </div>
              ))
            ) : (
              <div className={styles.empty}>{loading ? '...' : t('dashboard.no_usage_rank_data')}</div>
            )}
          </div>
        </article>

        <article className={`${styles.chartCard} ${styles.timelineCard}`}>
          <div className={styles.cardHeader}>
            <div>
              <h2>{t('usage_page.traffic_trend', { defaultValue: '流量趋势' })}</h2>
              <p>{t('usage_page.traffic_desc', { defaultValue: '请求量与 Token 量随筛选范围变化。' })}</p>
            </div>
            <div className={styles.legend}>
              <span className={styles.legendItem}>
                <span className={styles.dot} style={{ background: '#3b82f6' }} />
                {t('dashboard.traffic_calls')}
              </span>
              <span className={styles.legendItem}>
                <span className={styles.dot} style={{ background: '#10b981' }} />
                {t('dashboard.traffic_tokens')}
              </span>
            </div>
          </div>
          <div className={styles.trafficChart}>
            <div className={styles.trafficPlot}>
              <div className={styles.trafficGridLines} aria-hidden="true">
                {Array.from({ length: 5 }, (_, index) => <span key={index} />)}
              </div>
              <div className={styles.trafficBars} style={trafficGridStyle}>
                {timeline.map((point, index) => {
                  const tokenShare = getTimelineShare(
                    point,
                    'tokens',
                    maxTimelineRequests,
                    maxTimelineTokens
                  );
                  return (
                    <div
                      key={getTimelineKey(point, index)}
                      className={styles.trafficBucket}
                      title={`${point.label} · ${t('dashboard.traffic_calls')}: ${formatCompactNumber(point.requests)} · ${t('dashboard.traffic_tokens')}: ${formatCompactNumber(point.tokens)}`}
                    >
                      <div
                        className={`${styles.trafficBar} ${styles.tokensBar}`}
                        style={
                          {
                            '--metric-share': tokenShare,
                            '--metric-min-height': tokenShare > 0 ? '2px' : '0px',
                          } as TrafficBarStyle
                        }
                      />
                    </div>
                  );
                })}
              </div>
              {hasTrafficData ? (
                <svg className={styles.callsLineLayer} viewBox="0 0 100 100" preserveAspectRatio="none" aria-hidden="true">
                  <path d={callsLinePath} />
                </svg>
              ) : null}
              <div className={styles.trafficYAxis} aria-hidden="true">
                <span>{t('dashboard.traffic_tokens')}</span>
                <span>{t('dashboard.traffic_calls')}</span>
              </div>
              {!hasTrafficData ? (
                <div className={styles.empty}>{loading ? '...' : t('dashboard.no_traffic_data')}</div>
              ) : null}
            </div>
            <div className={styles.trafficAxis} style={trafficGridStyle}>
              {trafficAxisTickIndexes.map((index) => {
                const point = timeline[index];
                return point ? (
                  <span key={getTimelineKey(point, index)} style={{ gridColumn: index + 1 }}>
                    {point.label}
                  </span>
                ) : null;
              })}
            </div>
          </div>
        </article>
      </div>
    </section>
  );
}
