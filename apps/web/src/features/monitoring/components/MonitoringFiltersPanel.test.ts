import { describe, expect, it } from 'vitest';
import filtersPanelSource from './MonitoringFiltersPanel.tsx?raw';
import monitoringStyles from '../MonitoringCenterPage.module.scss';

describe('MonitoringFiltersPanel mobile time range labels', () => {
  it('wraps time range text in a stable label element', () => {
    expect(filtersPanelSource).toContain('styles.segmentButtonLabel');
    expect(filtersPanelSource).toContain('<span className={styles.segmentButtonLabel}>{t(option.labelKey)}</span>');
  });

  it('keeps mobile time range labels vertically centered and repaint-safe', () => {
    expect(filtersPanelSource).toContain('styles.segmentButtonLabel');
    expect(monitoringStyles.segmentButtonLabel).toBeTypeOf('string');
    expect(monitoringStyles.segmentedControl).toBeTypeOf('string');
  });
});
