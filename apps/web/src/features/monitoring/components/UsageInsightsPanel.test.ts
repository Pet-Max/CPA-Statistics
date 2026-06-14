import { describe, expect, it } from 'vitest';
import usageInsightsPanelSource from './UsageInsightsPanel.tsx?raw';

describe('UsageInsightsPanel source contract', () => {
  it('derives the model cost rank from filtered model share rows', () => {
    expect(usageInsightsPanelSource).toContain('const modelCostRank = useMemo<RankedModelCostRow[]>');
    expect(usageInsightsPanelSource).toContain('modelShareRows.reduce');
    expect(usageInsightsPanelSource).toContain('sort(sortModelCostRows)');
    expect(usageInsightsPanelSource).toContain('slice(0, 5)');
    expect(usageInsightsPanelSource).toContain('const rankShare = totalCostSum > 0 ? costShare : totalTokenSum > 0 ? tokenShare : requestShare');
  });

  it('uses filter-aware usage labels instead of today-only dashboard labels', () => {
    expect(usageInsightsPanelSource).toContain("t('usage_page.traffic_trend'");
    expect(usageInsightsPanelSource).toContain("t('dashboard.model_cost_rank')");
    expect(usageInsightsPanelSource).not.toContain('dashboard.traffic_trend_today');
    expect(usageInsightsPanelSource).not.toContain('model_cost_rank_today');
    expect(usageInsightsPanelSource).not.toContain('trafficNowMs');
  });

  it('renders chart cards in the intended usage page order', () => {
    const requestHealthIndex = usageInsightsPanelSource.indexOf("<h2>{t('usage_page.request_health'");
    const tokenMixIndex = usageInsightsPanelSource.indexOf("<h2>{t('usage_page.token_mix'");
    const modelCostRankIndex = usageInsightsPanelSource.indexOf("<h2>{t('dashboard.model_cost_rank')");
    const trafficTrendIndex = usageInsightsPanelSource.indexOf("<h2>{t('usage_page.traffic_trend'");

    expect(requestHealthIndex).toBeGreaterThan(-1);
    expect(tokenMixIndex).toBeGreaterThan(requestHealthIndex);
    expect(modelCostRankIndex).toBeGreaterThan(tokenMixIndex);
    expect(trafficTrendIndex).toBeGreaterThan(modelCostRankIndex);
  });
});
