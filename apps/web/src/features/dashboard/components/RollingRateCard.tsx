import { useTranslation } from 'react-i18next';
import type { DashboardSummaryResponse } from '@/services/api/usageService';
import { formatCompactNumber } from '@/utils/usage';
import styles from './RollingRateCard.module.scss';

interface RollingRateCardProps {
  summary: DashboardSummaryResponse | null;
  loading: boolean;
  error?: string;
}

const formatRate = (value: number | undefined, locale: string) => {
  if (value === undefined || !Number.isFinite(value)) return '-';
  return value.toLocaleString(locale, {
    maximumFractionDigits: 1,
    minimumFractionDigits: 1,
  });
};

export function RollingRateCard({ summary, loading, error }: RollingRateCardProps) {
  const { t, i18n } = useTranslation();
  const rolling = summary?.rolling_30m;
  const loadingText = loading ? '...' : '-';
  const hasError = Boolean(error && !loading && !rolling);

  return (
    <section className={styles.dataCard}>
      <div className={styles.cardHeader}>
        <h3>{t('usage_stats.rate_30m')}</h3>
      </div>
      <div className={styles.metricsGrid}>
        <div className={styles.metricItem}>
          <span className={styles.metricLabel}>{t('dashboard.tpm_30m')}</span>
          <strong className={styles.metricValue}>
            {rolling ? formatCompactNumber(rolling.tpm) : loadingText}
          </strong>
          <small className={styles.metricSubValue}>
            {rolling
              ? t('dashboard.metric_rolling_tokens', {
                  value: formatCompactNumber(rolling.total_tokens),
                })
              : hasError
                ? error
                : t('dashboard.no_usage_rank_data')}
          </small>
        </div>
        <div className={styles.metricItem}>
          <span className={styles.metricLabel}>{t('dashboard.rpm_30m')}</span>
          <strong className={styles.metricValue}>
            {rolling ? formatRate(rolling.rpm, i18n.language) : loadingText}
          </strong>
          <small className={styles.metricSubValue}>
            {rolling
              ? t('dashboard.metric_rolling_calls', {
                  value: rolling.total_calls.toLocaleString(i18n.language),
                })
              : hasError
                ? error
                : t('dashboard.no_usage_rank_data')}
          </small>
        </div>
      </div>
    </section>
  );
}
