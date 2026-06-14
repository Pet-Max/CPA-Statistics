import { describe, expect, it } from 'vitest';
import dashboardPageSource from './DashboardPage.tsx?raw';
import dashboardSummaryHookSource from './hooks/useDashboardUsageSummary.ts?raw';

describe('dashboard usage isolation', () => {
  it('keeps dashboard summary requests lightweight for overview health cards', () => {
    expect(dashboardSummaryHookSource).toContain('topModels: 0');
    expect(dashboardSummaryHookSource).not.toContain('topModels: summary?.top_models_today ?? []');
    expect(dashboardSummaryHookSource).not.toContain('modelCostRank: summary?.model_cost_rank ?? []');
    expect(dashboardSummaryHookSource).not.toContain('trafficTimeline: summary?.traffic_timeline ?? []');
    expect(dashboardSummaryHookSource).not.toContain('todayRequestHealthTimeline');
    expect(dashboardSummaryHookSource).not.toContain('tokenMix: summary?.token_mix ?? []');
  });

  it('does not render usage traffic or model cost rank cards on overview', () => {
    expect(dashboardPageSource).toContain("from './components/RollingRateCard'");
    expect(dashboardPageSource).toContain('ChannelHealthCard');
    expect(dashboardPageSource).toContain('RecentFailuresCard');
    expect(dashboardPageSource).toContain('<RollingRateCard');
    expect(dashboardPageSource).toContain('<ChannelHealthCard');
    expect(dashboardPageSource).toContain('<RecentFailuresCard');
    expect(dashboardPageSource).not.toContain("from './components/TrafficOverviewCard'");
    expect(dashboardPageSource).not.toContain("from './components/UsageMetricsCard'");
    expect(dashboardPageSource).not.toContain('<UsageMetricsCard');
    expect(dashboardPageSource).not.toContain('mode="metrics-only"');
    expect(dashboardPageSource).not.toContain('mode="rank-only"');
    expect(dashboardPageSource).not.toContain('<TrafficOverviewCard');
    expect(dashboardPageSource).not.toContain('trafficTimeline');
    expect(dashboardPageSource).not.toContain('trafficNowMs');
    expect(dashboardPageSource).not.toContain('todayRequestHealthTimeline');
    expect(dashboardPageSource).not.toContain('tokenMix');
  });
});
