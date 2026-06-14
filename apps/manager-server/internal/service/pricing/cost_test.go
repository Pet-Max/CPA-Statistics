package pricing

import (
	"math"
	"testing"

	"github.com/seakee/cpa-statistics/apps/manager-server/internal/model"
)

func TestCostForModelSeparatesCachedInputTokens(t *testing.T) {
	prices := map[string]model.ModelPrice{
		"gpt-cached": {Prompt: 2, Completion: 4, Cache: 1},
	}

	cost := CostForModel("gpt-cached", ModelTokens{
		InputTokens:  1_000_000,
		OutputTokens: 500_000,
		CachedTokens: 250_000,
	}, prices)

	if math.Abs(cost-3.75) > 0.000001 {
		t.Fatalf("cost = %v, want 3.75", cost)
	}
}

func TestCostForModelDoesNotCreateNegativePromptCost(t *testing.T) {
	prices := map[string]model.ModelPrice{
		"gpt-cached": {Prompt: 10, Cache: 1},
	}

	cost := CostForModel("gpt-cached", ModelTokens{
		InputTokens:  100_000,
		CachedTokens: 250_000,
	}, prices)

	if math.Abs(cost-0.25) > 0.000001 {
		t.Fatalf("cost = %v, want 0.25", cost)
	}
}

func TestCostForModelPricesFineGrainedCacheOutsideInput(t *testing.T) {
	prices := map[string]model.ModelPrice{
		"claude-cached": {Prompt: 2, Completion: 4, Cache: 1, CacheRead: 1, CacheCreation: 3},
	}

	cost := CostForModel("claude-cached", ModelTokens{
		InputTokens:         500_000,
		OutputTokens:        250_000,
		CachedTokens:        0,
		CacheReadTokens:     2_000_000,
		CacheCreationTokens: 100_000,
	}, prices)

	if math.Abs(cost-4.3) > 0.000001 {
		t.Fatalf("cost = %v, want 4.3", cost)
	}
}

func TestCostForModelPricesResidualCompatCachedWithFineGrainedCache(t *testing.T) {
	prices := map[string]model.ModelPrice{
		"mixed-cache": {Prompt: 2, Completion: 4, Cache: 1, CacheRead: 0.5, CacheCreation: 3},
	}

	cost := CostForModel("mixed-cache", ModelTokens{
		InputTokens:         1_000_000,
		CachedTokens:        100_000,
		CacheReadTokens:     200_000,
		CacheCreationTokens: 100_000,
	}, prices)

	if math.Abs(cost-2.3) > 0.000001 {
		t.Fatalf("cost = %v, want 2.3", cost)
	}
}
