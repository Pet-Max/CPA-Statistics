import type { TFunction } from 'i18next';
import { LoadingSpinner } from '@/components/ui/LoadingSpinner';
import type { MonitoringStatusTone } from '@/features/monitoring/hooks/useMonitoringData';
import { formatCompactNumber } from '@/utils/usage';
import styles from '../MonitoringCenterPage.module.scss';

type MonitoringStatusHeaderProps = {
  showLoadingOverlay: boolean;
  monitoringUnavailable: boolean;
  monitoringUnavailableTitle: string;
  monitoringUnavailableBody: string;
  t: TFunction;
};

type MonitoringStatusSummaryProps = {
  connectionTone: MonitoringStatusTone;
  connectionLabel: string;
  lastRefreshedAt: Date | null;
  locale: string;
  scopedFailureCount: number;
  totalCalls: number;
  t: TFunction;
};

export function MonitoringStatusSummary({
  connectionTone,
  connectionLabel,
  lastRefreshedAt,
  locale,
  scopedFailureCount,
  totalCalls,
  t,
}: MonitoringStatusSummaryProps) {
  return (
    <div className={styles.statusBar}>
      <span className={`${styles.statusBadge} ${styles[`tone${connectionTone}`]}`}>
        <span className={styles.statusDot} aria-hidden="true" />
        {connectionLabel}
      </span>
      <div className={styles.statusMeta}>
        <span>
          {t('monitoring.last_sync')}:{' '}
          {lastRefreshedAt ? lastRefreshedAt.toLocaleTimeString(locale) : '--'}
        </span>
        <span className={scopedFailureCount > 0 ? styles.statusMetaWarn : undefined}>
          {`${t('monitoring.recent_failures')}: ${scopedFailureCount}`}
        </span>
        <span>{`${t('monitoring.total_calls')}: ${formatCompactNumber(totalCalls)}`}</span>
      </div>
    </div>
  );
}

export function MonitoringStatusHeader({
  showLoadingOverlay,
  monitoringUnavailable,
  monitoringUnavailableTitle,
  monitoringUnavailableBody,
  t,
}: MonitoringStatusHeaderProps) {
  return (
    <>
      {showLoadingOverlay ? (
        <div className={styles.loadingOverlay} aria-busy="true">
          <div className={styles.loadingOverlayContent}>
            <LoadingSpinner size={28} />
            <span>{t('common.loading')}</span>
          </div>
        </div>
      ) : null}

      {monitoringUnavailable ? (
        <div className={styles.callout}>
          <strong>{monitoringUnavailableTitle}</strong>
          <span>{monitoringUnavailableBody}</span>
        </div>
      ) : null}
    </>
  );
}
