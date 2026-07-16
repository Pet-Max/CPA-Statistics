import { describe, expect, it } from 'vitest';
import {
  applyCandidatePrice,
  buildPriceFromDraft,
  buildModelPriceRows,
  buildModelPriceSummary,
  buildSyncPriceModelsFromUsage,
  filterModelPriceRows,
} from './modelPricesPageModel';

const usage = {
  apis: {
    'POST /v1/chat/completions': {
      models: {
        'alias-fast': {
          details: [
            {
              timestamp: '2026-05-22T00:00:00Z',
              source: 'source',
              resolved_model: 'gpt-5.5',
              tokens: {},
            },
          ],
        },
      },
    },
  },
};

describe('modelPricesPageModel', () => {
  it('uses resolved models as the price-sync scope and hides legacy alias keys', () => {
    expect(
      buildSyncPriceModelsFromUsage(usage, {
        'alias-fast': { prompt: 9, completion: 9, cache: 9 },
        'manual-model': { prompt: 1, completion: 2, cache: 0.5 },
      })
    ).toEqual(['gpt-5.5', 'manual-model']);
  });

  it('keeps aliases as metadata on the actual billing-model row', () => {
    const rows = buildModelPriceRows(usage, {
      'gpt-5.5': { prompt: 1, completion: 2, cache: 0.5 },
    });

    expect(rows).toHaveLength(1);
    expect(rows[0]).toMatchObject({
      model: 'gpt-5.5',
      hasPrice: true,
      calls: 1,
      requestedCalls: 1,
      resolvedCalls: 1,
      aliases: ['alias-fast'],
    });
    expect(buildModelPriceSummary(rows)).toMatchObject({
      total: 1,
      saved: 1,
      missing: 0,
    });
  });

  it('offers a legacy price candidate on the actual model, never on the alias', () => {
    const rows = buildModelPriceRows(usage, {}, [
      {
        model: 'gpt-5.5',
        candidates: [
          {
            sourceModelId: 'alias-fast',
            score: 1,
            reason: 'legacy_alias_price',
            price: { prompt: 1, completion: 2, cache: 0.5, source: 'migration' },
          },
        ],
      },
    ]);

    expect(rows).toHaveLength(1);
    expect(rows[0]).toMatchObject({
      model: 'gpt-5.5',
      hasPrice: false,
      candidateCount: 1,
      aliases: ['alias-fast'],
    });
    expect(filterModelPriceRows(rows, 'candidates', '')).toHaveLength(1);
  });

  it('applies a candidate under the actual billing model name', () => {
    const next = applyCandidatePrice({}, 'gpt-5.5', {
      sourceModelId: 'alias-fast',
      score: 1,
      reason: 'legacy_alias_price',
      price: { prompt: 1, completion: 2, cache: 0.5, source: 'migration' },
    });

    expect(next['gpt-5.5']).toMatchObject({
      prompt: 1,
      completion: 2,
      cache: 0.5,
      source: 'migration',
      sourceModelId: 'alias-fast',
    });
    expect(next['alias-fast']).toBeUndefined();
  });

  it('marks manually entered prices with a manual source', () => {
    expect(
      buildPriceFromDraft({
        model: 'manual-model',
        prompt: '1',
        completion: '2',
        cache: '',
      })
    ).toMatchObject({
      prompt: 1,
      completion: 2,
      cache: 1,
      source: 'manual',
    });
  });
});
