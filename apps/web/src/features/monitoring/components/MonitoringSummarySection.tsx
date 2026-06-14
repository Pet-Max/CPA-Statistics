import { SummaryCard, type SummaryCardProps } from '@/features/monitoring/components/MonitoringShared';
import styles from '../MonitoringCenterPage.module.scss';

type MonitoringSummarySectionProps = {
  primaryCards: SummaryCardProps[];
  secondaryCards: SummaryCardProps[];
};

export function MonitoringSummarySection({
  primaryCards,
  secondaryCards,
}: MonitoringSummarySectionProps) {
  return (
    <section className={styles.summarySection}>
      <div className={styles.summaryHero}>
        {primaryCards.map((card) => (
          <SummaryCard key={card.label} {...card} />
        ))}
      </div>
      <div className={styles.summarySub}>
        {secondaryCards.map((card) => (
          <SummaryCard key={card.label} {...card} />
        ))}
      </div>
    </section>
  );
}
