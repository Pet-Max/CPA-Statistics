import { describe, expect, it } from 'vitest';
import { buildTimelineFromAnalytics } from './analyticsAdapters';

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
