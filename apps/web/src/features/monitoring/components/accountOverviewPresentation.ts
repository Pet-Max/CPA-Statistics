import type { TFunction } from 'i18next';
import type { MonitoringAccountAuthState } from '@/features/monitoring/accountOverviewState';
import type { MonitoringAccountQuotaProvider } from '@/features/monitoring/accountOverviewQuotaTargets';
import type { MonitoringAccountRow } from '@/features/monitoring/hooks/useMonitoringData';
import { normalizePlanType } from '@/utils/quota';
import { formatCompactNumber, formatUsd } from '@/utils/usage';
import styles from '../MonitoringCenterPage.module.scss';

const PREMIUM_CODEX_PLAN_TYPES = new Set(['pro', 'prolite', 'pro-lite', 'pro_lite']);

export type AccountQuotaWindow = {
  id: string;
  label: string;
  remainingPercent: number | null;
  resetLabel: string;
  usageLabel: string | null;
};

export type AccountQuotaEntry = {
  key: string;
  provider: MonitoringAccountQuotaProvider;
  providerLabel: string;
  authLabel: string;
  fileName: string;
  planType: string | null;
  metaLabels?: string[];
  emptyMessage?: string;
  windows: AccountQuotaWindow[];
  error?: string;
};

export type AccountQuotaState = {
  status: 'idle' | 'loading' | 'success' | 'error';
  targetKey: string;
  entries: AccountQuotaEntry[];
  error?: string;
  lastRefreshedAt?: number;
};

export type AccountSummaryMetric = {
  key: string;
  label: string;
  value: string;
  valueClassName?: string;
};

export const formatPercent = (value: number) => `${(value * 100).toFixed(1)}%`;

const joinShort = (values: string[], limit = 2) => {
  if (values.length <= limit) {
    return values.join(', ');
  }
  return `${values.slice(0, limit).join(', ')} +${values.length - limit}`;
};

export const getCodexPlanLabel = (
  planType: string | null | undefined,
  t: TFunction
): string | null => {
  const normalized = normalizePlanType(planType);
  if (!normalized) return null;
  if (normalized === 'pro') return t('codex_quota.plan_pro');
  if (PREMIUM_CODEX_PLAN_TYPES.has(normalized) && normalized !== 'pro') {
    return t('codex_quota.plan_prolite');
  }
  if (normalized === 'plus') return t('codex_quota.plan_plus');
  if (normalized === 'team') return t('codex_quota.plan_team');
  if (normalized === 'free') return t('codex_quota.plan_free');
  return planType || normalized;
};

export const buildAccountSecondaryText = (row: MonitoringAccountRow) => {
  const primaryText = row.displayAccount || row.account;
  if (row.account && row.account !== primaryText) {
    return row.account;
  }

  const extraAuthLabels = row.authLabels.filter((label) => label && label !== primaryText);
  if (extraAuthLabels.length > 0) {
    return joinShort(extraAuthLabels, 2);
  }
  const extraChannels = row.channels.filter(
    (label) => label && label !== '-' && label !== primaryText
  );
  if (extraChannels.length > 0) {
    return joinShort(extraChannels, 2);
  }
  return '';
};

export const buildAccountSummaryMetrics = (
  row: MonitoringAccountRow,
  hasPrices: boolean,
  locale: string,
  t: TFunction
): AccountSummaryMetric[] => [
  {
    key: 'total-calls',
    label: t('monitoring.total_calls'),
    value: formatCompactNumber(row.totalCalls),
  },
  {
    key: 'success-calls',
    label: t('monitoring.success_calls'),
    value: formatCompactNumber(row.successCalls),
    valueClassName: styles.goodText,
  },
  {
    key: 'failure-calls',
    label: t('monitoring.failure_calls'),
    value: formatCompactNumber(row.failureCalls),
    valueClassName: row.failureCalls > 0 ? styles.badText : undefined,
  },
  {
    key: 'total-tokens',
    label: t('monitoring.total_tokens'),
    value: formatCompactNumber(row.totalTokens),
  },
  {
    key: 'input-tokens',
    label: t('monitoring.input_tokens'),
    value: formatCompactNumber(row.inputTokens),
  },
  {
    key: 'output-tokens',
    label: t('monitoring.output_tokens'),
    value: formatCompactNumber(row.outputTokens),
  },
  {
    key: 'cached-tokens',
    label: t('monitoring.cached_tokens'),
    value: formatCompactNumber(row.cachedTokens),
  },
  {
    key: 'estimated-cost',
    label: t('monitoring.estimated_cost'),
    value: hasPrices ? formatUsd(row.totalCost) : '--',
  },
  {
    key: 'latest-request-time',
    label: t('monitoring.latest_request_time'),
    value: new Date(row.lastSeenAt).toLocaleString(locale),
  },
];

export const getAccountStatusTone = (authState: MonitoringAccountAuthState) => {
  switch (authState.enabledState) {
    case 'enabled':
      return 'enabled';
    case 'disabled':
      return 'disabled';
    case 'mixed':
      return 'mixed';
    case 'unavailable':
    default:
      return 'unavailable';
  }
};

export const getAccountStatusLabel = (authState: MonitoringAccountAuthState, t: TFunction) => {
  switch (authState.enabledState) {
    case 'enabled':
      return t('monitoring.account_overview_enabled_state_enabled');
    case 'disabled':
      return t('monitoring.account_overview_enabled_state_disabled');
    case 'mixed':
      return t('monitoring.account_overview_enabled_state_mixed');
    case 'unavailable':
    default:
      return t('monitoring.account_overview_enabled_state_unavailable');
  }
};

export const getAccountStatusDotClassName = (tone: string) => {
  switch (tone) {
    case 'enabled':
      return styles.accountStatusDotEnabled;
    case 'disabled':
      return styles.accountStatusDotDisabled;
    case 'mixed':
      return styles.accountStatusDotMixed;
    case 'unavailable':
    default:
      return styles.accountStatusDotUnavailable;
  }
};

export const getSuccessRateClassName = (rate: number) =>
  rate >= 0.95 ? styles.goodText : rate >= 0.85 ? styles.warnText : styles.badText;
