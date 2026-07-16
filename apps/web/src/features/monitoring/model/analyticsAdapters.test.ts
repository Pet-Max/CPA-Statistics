import { describe, expect, it } from 'vitest';
import type { MonitoringAnalyticsSummary } from '@/services/api/usageService';
import { buildSummaryFromAnalytics, buildTimelineFromAnalytics } from './analyticsAdapters';

describe('analyticsAdapters timeline', () => {
  it('maps bucket timestamps and relative request/token shares', () => {
    const timeline = buildTimelineFromAnalytics(
      [
        {
          bucket_ms: 1_800_000,
          label: 'first',
          calls: 10,
          tokens: 50,
          success: 9,
          failure: 1,
        },
        {
          bucket_ms: 5_400_000,
          label: 'second',
          calls: 40,
          tokens: 200,
          success: 40,
          failure: 0,
        },
      ],
      'hour'
    );

    expect(timeline[0]).toMatchObject({
      bucketMs: 1_800_000,
      requests: 10,
      tokens: 50,
      cost: 0,
      requestsShare: 0.25,
      tokensShare: 0.25,
    });
    expect(timeline[1]).toMatchObject({
      bucketMs: 5_400_000,
      requests: 40,
      tokens: 200,
      cost: 0,
      requestsShare: 1,
      tokensShare: 1,
    });
  });

  it('uses zero shares when all analytics buckets are empty', () => {
    expect(
      buildTimelineFromAnalytics(
        [
          {
            bucket_ms: 1_800_000,
            label: 'empty',
            calls: 0,
            tokens: 0,
            success: 0,
            failure: 0,
          },
        ],
        'hour'
      )[0]
    ).toMatchObject({
      bucketMs: 1_800_000,
      requestsShare: 0,
      tokensShare: 0,
    });
  });
});


describe('analyticsAdapters summary', () => {
  const baseSummary: MonitoringAnalyticsSummary = {
    total_calls: 3,
    success_calls: 2,
    failure_calls: 1,
    success_rate: 2 / 3,
    input_tokens: 10,
    output_tokens: 20,
    cached_tokens: 0,
    cache_read_tokens: 0,
    cache_creation_tokens: 0,
    reasoning_tokens: 0,
    total_tokens: 30,
    total_cost: 0,
    average_latency_ms: 500,
    average_ttft_ms: 120,
    zero_token_calls: 0,
    rpm_30m: 0.1,
    tpm_30m: 1,
    avg_daily_requests: 3,
    avg_daily_tokens: 30,
    approx_tasks: 3,
    approx_task_failures: 1,
    approx_task_success_rate: 2 / 3,
    zero_token_models: [],
  };

  it('maps average TTFT and preserves a missing value', () => {
    expect(buildSummaryFromAnalytics(baseSummary).averageTTFTMs).toBe(120);
    expect(buildSummaryFromAnalytics({ ...baseSummary, average_ttft_ms: null }).averageTTFTMs).toBeNull();
  });
});
