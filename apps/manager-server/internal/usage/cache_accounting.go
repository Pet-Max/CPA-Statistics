package usage

import (
	"encoding/json"
	"strings"
)

const (
	CacheInputModeIncluded = "included_in_input"
	CacheInputModeSeparate = "separate_from_input"
)

// CacheAccounting is the canonical token split used for monetary cost. Raw
// event fields remain unchanged for compatibility and request inspection.
type CacheAccounting struct {
	Mode                string
	UncachedInputTokens int64
	TotalInputTokens    int64
}

type CacheInputContext struct {
	ExplicitMode     string
	ExecutorType     string
	Provider         string
	ProviderSnapshot string
	ResolvedModel    string
	RequestedModel   string
	DisplayModel     string
}

func NormalizeCacheAccounting(context CacheInputContext, inputTokens, cachedTokens, cacheTokens, cacheReadTokens, cacheCreationTokens int64) CacheAccounting {
	mode := InferCacheInputMode(context, cacheReadTokens, cacheCreationTokens)
	input := maxInt64(inputTokens, 0)
	legacyCached := CompatibleCachedTokens(cachedTokens, cacheTokens, cacheReadTokens, cacheCreationTokens)
	cacheRead := maxInt64(cacheReadTokens, 0)
	cacheCreation := maxInt64(cacheCreationTokens, 0)
	if mode == CacheInputModeSeparate {
		return CacheAccounting{
			Mode:                mode,
			UncachedInputTokens: input,
			TotalInputTokens:    input + legacyCached + cacheRead + cacheCreation,
		}
	}
	return CacheAccounting{
		Mode:                CacheInputModeIncluded,
		UncachedInputTokens: maxInt64(input-legacyCached-cacheRead-cacheCreation, 0),
		TotalInputTokens:    input,
	}
}

func InferCacheInputMode(context CacheInputContext, cacheReadTokens, cacheCreationTokens int64) string {
	if mode := normalizeCacheInputMode(context.ExplicitMode); mode != "" {
		return mode
	}
	if mode, ok := classifyExecutorCacheInputMode(context.ExecutorType); ok {
		return mode
	}
	for _, provider := range []string{context.Provider, context.ProviderSnapshot} {
		if mode, ok := classifyProviderCacheInputMode(provider); ok {
			return mode
		}
	}
	for _, model := range []string{context.ResolvedModel, context.RequestedModel, context.DisplayModel} {
		if mode, ok := classifyModelCacheInputMode(model); ok {
			return mode
		}
	}
	if cacheReadTokens > 0 || cacheCreationTokens > 0 {
		return CacheInputModeSeparate
	}
	return CacheInputModeIncluded
}

func CacheInputModeFromRawJSON(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	var payload map[string]any
	if json.Unmarshal([]byte(raw), &payload) != nil {
		return ""
	}
	return cacheInputModeFromRecord(payload)
}

func cacheInputModeFromRecord(record map[string]any) string {
	for _, parent := range []string{"tokens", "usage"} {
		if nested, ok := record[parent].(map[string]any); ok {
			if mode := normalizeCacheInputMode(readString(nested, "cache_input_mode", "cacheInputMode")); mode != "" {
				return mode
			}
		}
	}
	return normalizeCacheInputMode(readString(record, "cache_input_mode", "cacheInputMode"))
}

func normalizeCacheInputMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case CacheInputModeIncluded:
		return CacheInputModeIncluded
	case CacheInputModeSeparate:
		return CacheInputModeSeparate
	default:
		return ""
	}
}

func classifyExecutorCacheInputMode(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "", false
	}
	if strings.Contains(value, "claude") {
		return CacheInputModeSeparate, true
	}
	for _, marker := range []string{"openaicompat", "openai_compat", "openai-compat", "openai", "codex", "gemini", "aistudio", "ai_studio", "ai-studio", "antigravity", "xai", "kimi"} {
		if strings.Contains(value, marker) {
			return CacheInputModeIncluded, true
		}
	}
	return "", false
}

func classifyProviderCacheInputMode(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "", false
	}
	if strings.Contains(value, "anthropic") || strings.Contains(value, "claude") {
		return CacheInputModeSeparate, true
	}
	for _, marker := range []string{"openai", "codex", "gemini", "vertex", "aistudio", "ai_studio", "ai-studio", "interaction", "antigravity", "xai", "kimi", "moonshot"} {
		if strings.Contains(value, marker) {
			return CacheInputModeIncluded, true
		}
	}
	return "", false
}

func classifyModelCacheInputMode(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "", false
	}
	if strings.Contains(value, "anthropic") || strings.Contains(value, "claude") {
		return CacheInputModeSeparate, true
	}
	for _, marker := range []string{"gpt-", "openai", "codex", "gemini", "vertex", "aistudio", "antigravity", "grok", "xai", "kimi", "moonshot"} {
		if strings.Contains(value, marker) {
			return CacheInputModeIncluded, true
		}
	}
	return "", false
}
