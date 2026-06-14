import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import type { DashboardChannelHealth, DashboardRecentFailure } from '@/services/api/usageService';
import type { CredentialInfo } from '@/types/sourceInfo';
import type { MonitoringAuthMeta, MonitoringChannelMeta } from '@/features/monitoring/model/types';
import {
  buildMonitoringSourceDisplay,
  type MonitoringSourceDisplay,
} from '@/features/monitoring/model/sourceDisplay';
import { buildSourceInfoMap } from '@/utils/sourceResolver';
import { maskSensitiveText, truncateText } from '@/utils/format';
import { formatDurationMs } from '@/utils/usage';
import styles from './HealthAlertsCard.module.scss';

interface HealthAlertsSourceDisplayProps {
  authMetaMap: Map<string, MonitoringAuthMeta>;
  authFileMap: Map<string, CredentialInfo>;
  sourceInfoMap: ReturnType<typeof buildSourceInfoMap>;
  channelByAuthIndex: Map<string, MonitoringChannelMeta>;
  apiKeyAliasMap: Map<string, string>;
}

interface ChannelHealthCardProps extends HealthAlertsSourceDisplayProps {
  loading: boolean;
  channelHealth: DashboardChannelHealth[];
}

interface RecentFailuresCardProps extends HealthAlertsSourceDisplayProps {
  loading: boolean;
  recentFailures: DashboardRecentFailure[];
}

interface HealthAlertsCardProps extends HealthAlertsSourceDisplayProps {
  loading: boolean;
  recentFailures: DashboardRecentFailure[];
  channelHealth: DashboardChannelHealth[];
}

export function ChannelHealthCard({
  loading,
  channelHealth,
  authMetaMap,
  authFileMap,
  sourceInfoMap,
  channelByAuthIndex,
  apiKeyAliasMap,
}: ChannelHealthCardProps) {
  const { t, i18n } = useTranslation();

  const formatPercent = useMemo(
    () =>
      new Intl.NumberFormat(i18n.language, {
        style: 'percent',
        maximumFractionDigits: 1,
      }).format,
    [i18n.language]
  );

  const sourceDisplayContext = {
    authMetaMap,
    authFileMap,
    sourceInfoMap,
    channelByAuthIndex,
    apiKeyAliasMap,
  };

  const channelRows = channelHealth.slice(0, 5).map((channel, index) => ({
    channel,
    key: `${channel.auth_index}-${index}`,
    display: buildMonitoringSourceDisplay(
      {
        source: channel.source,
        authIndex: channel.auth_index,
        accountSnapshot: channel.account_snapshot,
        authLabelSnapshot: channel.auth_label_snapshot,
        authProviderSnapshot: channel.auth_provider_snapshot,
        authLabel: channel.auth_label,
        account: channel.account,
        channel: channel.channel,
      },
      sourceDisplayContext
    ),
  }));
  const channelNameCounts = channelRows.reduce((map, row) => {
    map.set(row.display.primary, (map.get(row.display.primary) ?? 0) + 1);
    return map;
  }, new Map<string, number>());

  return (
    <section className={styles.dataCard}>
      <div className={styles.cardHeader}>
        <h3>{t('dashboard.channel_health_status')}</h3>
      </div>
      <div className={styles.list}>
        {channelRows.map(({ channel, key, display }) => {
          const isDuplicateName = (channelNameCounts.get(display.primary) ?? 0) > 1;
          const label =
            channel.auth_index === '-'
              ? t('dashboard.health_unlinked_channel')
              : isDuplicateName && display.meta
                ? `${display.primary} · ${display.meta}`
                : display.primary;

          return (
            <div key={key} className={styles.listItem}>
              <span className={`${styles.statusDot} ${styles[channel.tone] || ''}`} />
              <span
                className={styles.label}
                title={channel.auth_index === '-' ? undefined : display.title}
              >
                {label}
              </span>
              <span className={styles.value}>{formatPercent(channel.success_rate)}</span>
            </div>
          );
        })}
        {channelHealth.length === 0 ? (
          <div className={styles.empty}>{loading ? '...' : t('dashboard.no_channel_health_data')}</div>
        ) : null}
      </div>
    </section>
  );
}

export function RecentFailuresCard({
  loading,
  recentFailures,
  authMetaMap,
  authFileMap,
  sourceInfoMap,
  channelByAuthIndex,
  apiKeyAliasMap,
}: RecentFailuresCardProps) {
  const { t, i18n } = useTranslation();

  const sourceDisplayContext = {
    authMetaMap,
    authFileMap,
    sourceInfoMap,
    channelByAuthIndex,
    apiKeyAliasMap,
  };

  const buildRecentFailureDisplay = (failure: DashboardRecentFailure) =>
    buildMonitoringSourceDisplay(
      {
        source: failure.source,
        sourceHash: failure.source_hash,
        apiKeyHash: failure.api_key_hash,
        authIndex: failure.auth_index,
        accountSnapshot: failure.account_snapshot,
        authLabelSnapshot: failure.auth_label_snapshot,
        authProviderSnapshot: failure.auth_provider_snapshot,
        channel: failure.channel,
        authLabel: failure.auth_label,
        account: failure.account,
        apiKeyAlias: failure.api_key_alias,
      },
      sourceDisplayContext
    );

  const buildFailureTooltip = (failure: DashboardRecentFailure) => {
    const statusCode = failure.fail_status_code;
    const summary = maskSensitiveText(failure.fail_summary || '');
    if (!statusCode && !summary) return null;
    const statusText = statusCode ? `${t('monitoring.fail_status_code_short')} ${statusCode}` : '';
    const compactSummary = summary ? truncateText(summary, 160) : '';
    return {
      statusText,
      summary: compactSummary,
      title: [statusText, compactSummary].filter(Boolean).join(' · '),
    };
  };

  const renderFailureName = (
    display: MonitoringSourceDisplay,
    tooltip: ReturnType<typeof buildFailureTooltip>
  ) => (
    <span
      className={tooltip ? styles.failureSourceWithTooltip : undefined}
      tabIndex={tooltip ? 0 : undefined}
      title={tooltip ? undefined : display.title}
    >
      <span className={styles.failureSourceText}>{display.primary}</span>
      {tooltip ? (
        <span role="tooltip" className={styles.failureTooltip}>
          {tooltip.statusText ? (
            <span className={styles.failureTooltipStatus}>{tooltip.statusText}</span>
          ) : null}
          {tooltip.summary ? (
            <span className={styles.failureTooltipBody}>{tooltip.summary}</span>
          ) : null}
        </span>
      ) : null}
    </span>
  );

  return (
    <section className={styles.dataCard}>
      <div className={styles.cardHeader}>
        <h3>{t('dashboard.recent_failed_requests')}</h3>
      </div>
      <div className={styles.list}>
        {recentFailures.slice(0, 3).map((failure) => {
          const display = buildRecentFailureDisplay(failure);
          const tooltip = buildFailureTooltip(failure);
          return (
            <div
              key={`${failure.timestamp_ms}-${failure.source_hash}-${failure.model}`}
              className={styles.failureItem}
            >
              <div className={styles.failureMeta}>
                <span className={styles.time}>
                  {new Date(failure.timestamp_ms).toLocaleTimeString(i18n.language)}
                </span>
                <span className={styles.model}>{failure.model}</span>
              </div>
              <div className={styles.failureDetail}>
                {renderFailureName(display, tooltip)}
                <span>{formatDurationMs(failure.duration_ms, { locale: i18n.language })}</span>
              </div>
            </div>
          );
        })}
        {recentFailures.length === 0 ? (
          <div className={styles.empty}>{loading ? '...' : t('dashboard.no_recent_failures')}</div>
        ) : null}
      </div>
    </section>
  );
}

export function HealthAlertsCard({
  loading,
  recentFailures,
  channelHealth,
  authMetaMap,
  authFileMap,
  sourceInfoMap,
  channelByAuthIndex,
  apiKeyAliasMap,
}: HealthAlertsCardProps) {
  const commonProps = {
    loading,
    authMetaMap,
    authFileMap,
    sourceInfoMap,
    channelByAuthIndex,
    apiKeyAliasMap,
  };

  return (
    <>
      <ChannelHealthCard {...commonProps} channelHealth={channelHealth} />
      <RecentFailuresCard {...commonProps} recentFailures={recentFailures} />
    </>
  );
}
