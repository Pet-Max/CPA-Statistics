package usage

import "testing"

func TestNormalizeCacheAccountingHandlesBothInputConventions(t *testing.T) {
	included := NormalizeCacheAccounting(CacheInputContext{ExplicitMode: CacheInputModeIncluded}, 1_000, 400, 0, 200, 100)
	if included.Mode != CacheInputModeIncluded || included.UncachedInputTokens != 600 || included.TotalInputTokens != 1_000 {
		t.Fatalf("included accounting = %#v", included)
	}

	separate := NormalizeCacheAccounting(CacheInputContext{ExplicitMode: CacheInputModeSeparate}, 1_000, 400, 0, 200, 100)
	if separate.Mode != CacheInputModeSeparate || separate.UncachedInputTokens != 1_000 || separate.TotalInputTokens != 1_400 {
		t.Fatalf("separate accounting = %#v", separate)
	}
}

func TestInferCacheInputModePrecedence(t *testing.T) {
	if got := InferCacheInputMode(CacheInputContext{ExplicitMode: CacheInputModeIncluded, Provider: "anthropic"}, 1, 1); got != CacheInputModeIncluded {
		t.Fatalf("explicit mode = %q", got)
	}
	if got := InferCacheInputMode(CacheInputContext{ExecutorType: "claude"}, 1, 1); got != CacheInputModeSeparate {
		t.Fatalf("executor mode = %q", got)
	}
	if got := InferCacheInputMode(CacheInputContext{Provider: "openai"}, 1, 1); got != CacheInputModeIncluded {
		t.Fatalf("provider mode = %q", got)
	}
	if got := InferCacheInputMode(CacheInputContext{ResolvedModel: "claude-sonnet"}, 1, 1); got != CacheInputModeSeparate {
		t.Fatalf("model mode = %q", got)
	}
	if got := InferCacheInputMode(CacheInputContext{}, 1, 0); got != CacheInputModeSeparate {
		t.Fatalf("fine-cache fallback = %q", got)
	}
}
