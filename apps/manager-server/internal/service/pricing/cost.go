// Package pricing converts token aggregates into monetary cost given a model price book.
package pricing

import "github.com/seakee/cpa-statistics/apps/manager-server/internal/model"

// PerMillion divides by one million to convert token-priced units (per 1M tokens).
const PerMillion = 1_000_000.0

// ModelTokens represents the token totals consumed by a single model.
// CachedTokens is the remaining legacy/OpenAI-style cached input after any
// fine-grained cache_read/cache_creation values have already been removed.
type ModelTokens struct {
	InputTokens         int64
	OutputTokens        int64
	CachedTokens        int64
	CacheReadTokens     int64
	CacheCreationTokens int64
}

// CostForModel computes the dollar cost for a single (model, tokens) pair.
// When CPA provides fine-grained cache read/create tokens, input tokens are
// treated as non-cache input and those cache dimensions are priced separately.
// Any residual CachedTokens are still charged at the legacy cache price; callers
// must pass the compatibility cached value, not CPA's Claude mirror copy.
// Older payloads keep the OpenAI-style cached-in-input behavior.
func CostForModel(modelName string, tokens ModelTokens, prices map[string]model.ModelPrice) float64 {
	price, ok := prices[modelName]
	if !ok {
		return 0
	}
	inputTokens := maxInt64(tokens.InputTokens, 0)
	outputTokens := maxInt64(tokens.OutputTokens, 0)
	cachedTokens := maxInt64(tokens.CachedTokens, 0)
	cacheReadTokens := maxInt64(tokens.CacheReadTokens, 0)
	cacheCreationTokens := maxInt64(tokens.CacheCreationTokens, 0)
	if cacheReadTokens > 0 || cacheCreationTokens > 0 {
		promptTokens := maxInt64(inputTokens-cachedTokens, 0)
		cacheReadPrice := fallbackPrice(price.CacheRead, price.Cache)
		cacheCreationPrice := fallbackPrice(price.CacheCreation, price.Prompt)
		return float64(promptTokens)*price.Prompt/PerMillion +
			float64(outputTokens)*price.Completion/PerMillion +
			float64(cachedTokens)*price.Cache/PerMillion +
			float64(cacheReadTokens)*cacheReadPrice/PerMillion +
			float64(cacheCreationTokens)*cacheCreationPrice/PerMillion
	}

	promptTokens := maxInt64(inputTokens-cachedTokens, 0)

	return float64(promptTokens)*price.Prompt/PerMillion +
		float64(outputTokens)*price.Completion/PerMillion +
		float64(cachedTokens)*price.Cache/PerMillion
}

// SumCost folds CostForModel over a slice of (model, tokens) tuples.
type Item struct {
	Model  string
	Tokens ModelTokens
}

// SumCost adds up the cost across multiple items.
func SumCost(items []Item, prices map[string]model.ModelPrice) float64 {
	total := 0.0
	for _, item := range items {
		total += CostForModel(item.Model, item.Tokens, prices)
	}
	return total
}

func maxInt64(left, right int64) int64 {
	if left > right {
		return left
	}
	return right
}

func fallbackPrice(value float64, fallback float64) float64 {
	if value > 0 {
		return value
	}
	return fallback
}
