package usage

import (
	"errors"
	"strings"
	"testing"
)

const legacyUsageExportFixture = `{
  "version": 1,
  "exported_at": "2026-01-02T03:04:05Z",
  "usage": {
    "total_requests": 2,
    "success_count": 1,
    "failure_count": 1,
    "total_tokens": 66,
    "apis": {
      "POST /v1/chat/completions": {
        "models": {
          "gpt-4o": {
            "details": [
              {
                "timestamp": "2026-01-02T03:04:05Z",
                "source": "alice@example.com",
                "auth_index": "auth-1",
                "tokens": {
                  "input_tokens": 10,
                  "output_tokens": 20,
                  "cached_tokens": 3,
                  "total_tokens": 33
                },
                "failed": false,
                "latency_ms": 123
              },
              {
                "timestamp": "2026-01-02T03:05:05Z",
                "source": "fake-test-api-key",
                "authIndex": "auth-2",
                "tokens": {
                  "inputTokens": 5,
                  "outputTokens": 6,
                  "reasoningTokens": 7,
                  "cacheTokens": 8
                },
                "failed": true
              }
            ]
          }
        }
      }
    }
  }
}`

func TestParseImportPayloadLegacyUsageExport(t *testing.T) {
	result, err := ParseImportPayload([]byte(legacyUsageExportFixture))
	if err != nil {
		t.Fatalf("parse legacy export: %v", err)
	}
	if result.Format != ImportFormatLegacyExport {
		t.Fatalf("format = %q", result.Format)
	}
	if len(result.Events) != 2 || result.Failed != 0 || result.Unsupported != 0 {
		t.Fatalf("summary = %#v", result)
	}
	if len(result.Warnings) == 0 {
		t.Fatalf("expected legacy warnings")
	}

	first := result.Events[0]
	if first.Model != "gpt-4o" || first.Endpoint != "POST /v1/chat/completions" {
		t.Fatalf("first event target = %#v", first)
	}
	if first.Method != "POST" || first.Path != "/v1/chat/completions" {
		t.Fatalf("first endpoint parts = %#v", first)
	}
	if first.Source != "ali***@example.com" || first.AuthIndex != "auth-1" {
		t.Fatalf("first source = %#v", first)
	}
	if first.TotalTokens != 33 || first.LatencyMS == nil || *first.LatencyMS != 123 {
		t.Fatalf("first metrics = %#v", first)
	}
	if first.EventHash == "" || !strings.HasPrefix(first.RequestID, "legacy:") {
		t.Fatalf("first ids = %#v", first)
	}

	second := result.Events[1]
	if second.TotalTokens != 26 || !second.Failed || second.AuthIndex != "auth-2" {
		t.Fatalf("second event = %#v", second)
	}

	again, err := ParseImportPayload([]byte(legacyUsageExportFixture))
	if err != nil {
		t.Fatalf("parse legacy export again: %v", err)
	}
	if again.Events[0].EventHash != first.EventHash || again.Events[1].EventHash != second.EventHash {
		t.Fatalf("legacy event hashes are not stable")
	}
}

func TestParseImportPayloadRejectsLegacySummaryWithoutDetails(t *testing.T) {
	payload := `{
	  "usage": {
	    "total_requests": 1,
	    "apis": {
	      "GET /v1/models": {
	        "models": {
	          "gpt-4o": {
	            "requests": 1
	          }
	        }
	      }
	    }
	  }
	}`
	result, err := ParseImportPayload([]byte(payload))
	if !errors.Is(err, ErrLegacyUsageNoDetails) {
		t.Fatalf("err = %v, result = %#v", err, result)
	}
	if result.Format != ImportFormatLegacyExport || result.Unsupported != 1 {
		t.Fatalf("summary = %#v", result)
	}
}

func TestParseImportPayloadPreservesExportedEventHash(t *testing.T) {
	payload := `{
	  "request_id": "req-1",
	  "event_hash": "stable-hash",
	  "timestamp_ms": 1760000000000,
	  "timestamp": "2025-10-09T08:53:20Z",
	  "model": "gpt-4o",
	  "endpoint": "POST /v1/chat/completions",
	  "source": "m:sk-t...alue",
	  "source_hash": "source-hash",
	  "api_key_hash": "key-hash",
	  "input_tokens": 1,
	  "output_tokens": 2,
	  "total_tokens": 3,
	  "created_at_ms": 1760000000001
	}`
	result, err := ParseImportPayload([]byte(payload))
	if err != nil {
		t.Fatalf("parse exported event: %v", err)
	}
	if result.Format != ImportFormatJSONL || len(result.Events) != 1 {
		t.Fatalf("result = %#v", result)
	}
	event := result.Events[0]
	if event.EventHash != "stable-hash" || event.SourceHash != "source-hash" || event.APIKeyHash != "key-hash" {
		t.Fatalf("event hashes = %#v", event)
	}
}

func TestParseImportPayloadJSONLCountsBadLines(t *testing.T) {
	payload := `{"timestamp":"2026-01-02T03:04:05Z","model":"gpt-4o","endpoint":"GET /v1/models","tokens":{"input_tokens":1}}
not-json`
	result, err := ParseImportPayload([]byte(payload))
	if err != nil {
		t.Fatalf("parse jsonl: %v", err)
	}
	if result.Format != ImportFormatJSONL || len(result.Events) != 1 || result.Failed != 1 {
		t.Fatalf("result = %#v", result)
	}
}

func TestParseImportPayloadPreservesAuthProjectIDSnapshot(t *testing.T) {
	payload := `{
	  "event_hash": "hash-project",
	  "timestamp_ms": 1760000000000,
	  "timestamp": "2025-10-09T08:53:20Z",
	  "model": "gemini-2.5",
	  "endpoint": "POST /v1/chat/completions",
	  "auth_project_id_snapshot": "vertex-project-42",
	  "input_tokens": 1,
	  "total_tokens": 1
	}`
	result, err := ParseImportPayload([]byte(payload))
	if err != nil {
		t.Fatalf("parse exported event: %v", err)
	}
	if len(result.Events) != 1 {
		t.Fatalf("result = %#v", result)
	}
	if got := result.Events[0].AuthProjectIDSnapshot; got != "vertex-project-42" {
		t.Fatalf("auth_project_id_snapshot = %q", got)
	}
}

func TestNormalizeRawReadsProjectID(t *testing.T) {
	payload := `{
	  "timestamp": "2026-05-19T10:00:00Z",
	  "model": "gemini-2.5",
	  "endpoint": "POST /v1/chat/completions",
	  "project_id": "vertex-project-42",
	  "input_tokens": 1,
	  "total_tokens": 1
	}`
	event, err := NormalizeRaw([]byte(payload))
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if event.AuthProjectIDSnapshot != "vertex-project-42" {
		t.Fatalf("auth_project_id_snapshot = %q", event.AuthProjectIDSnapshot)
	}
}

func TestNormalizeRawReadsCPA7118UsageFields(t *testing.T) {
	payload := `{
	  "timestamp": "2026-04-25T00:00:00Z",
	  "latency_ms": 1500,
	  "ttft_ms": 450,
	  "source": "user@example.com",
	  "auth_index": "0",
	  "tokens": {
	    "input_tokens": 10,
	    "output_tokens": 20,
	    "reasoning_tokens": 3,
	    "cached_tokens": 5,
	    "cache_read_tokens": 4,
	    "cache_creation_tokens": 1,
	    "total_tokens": 33
	  },
	  "failed": true,
	  "fail": {
	    "status_code": 429,
	    "body": "rate limit exceeded"
	  },
	  "provider": "openai",
	  "model": "gpt-5.4",
	  "alias": "client-gpt",
	  "endpoint": "POST /v1/chat/completions",
	  "auth_type": "apikey",
	  "api_key": "test-key",
	  "request_id": "ctx-request-id",
	  "reasoning_effort": "medium",
	  "service_tier": "priority",
	  "executor_type": "codex",
	  "response_headers": {
	    "Retry-After": ["30"]
	  }
	}`
	event, err := NormalizeRaw([]byte(payload))
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if event.RequestID != "ctx-request-id" || event.ReasoningEffort != "medium" ||
		event.ServiceTier != "priority" || event.ExecutorType != "codex" {
		t.Fatalf("event identity/metadata = %#v", event)
	}
	if event.InputTokens != 10 || event.OutputTokens != 20 || event.ReasoningTokens != 3 ||
		event.CachedTokens != 5 || event.CacheReadTokens != 4 || event.CacheCreationTokens != 1 ||
		event.TotalTokens != 33 {
		t.Fatalf("event tokens = %#v", event)
	}
	if !event.Failed || event.FailStatusCode != 429 || event.FailBody != "rate limit exceeded" {
		t.Fatalf("event failure = %#v", event)
	}
	if event.FailSummary != "rate limit exceeded" {
		t.Fatalf("fail summary = %q", event.FailSummary)
	}
	if event.LatencyMS == nil || *event.LatencyMS != 1500 {
		t.Fatalf("latency = %#v", event.LatencyMS)
	}
	if event.TTFTMS == nil || *event.TTFTMS != 450 {
		t.Fatalf("ttft = %#v", event.TTFTMS)
	}

	legacyPayload := BuildPayload([]Event{event})
	api := legacyPayload.APIs["POST /v1/chat/completions"]
	if api == nil {
		t.Fatalf("missing endpoint aggregate")
	}
	modelEntry := api.Models["client-gpt"]
	if modelEntry == nil || len(modelEntry.Details) != 1 {
		t.Fatalf("model details = %#v", api.Models)
	}
	detail := modelEntry.Details[0]
	if detail.ReasoningEffort != "medium" || detail.ServiceTier != "priority" ||
		detail.ExecutorType != "codex" || detail.Tokens.CacheReadTokens != 4 ||
		detail.Tokens.CacheCreationTokens != 1 || detail.FailStatusCode != 429 ||
		detail.Tokens.CachedTokens != 0 || detail.Tokens.CacheTokens != 0 ||
		detail.FailSummary != "rate limit exceeded" || detail.TTFTMS == nil ||
		*detail.TTFTMS != 450 {
		t.Fatalf("detail = %#v", detail)
	}
}

func TestCompatibleCachedTokensDoesNotDoubleCountFineGrainedCache(t *testing.T) {
	if got := CompatibleCachedTokens(5, 0, 4, 1); got != 0 {
		t.Fatalf("fully mirrored cached tokens = %d, want 0", got)
	}
	if got := CompatibleCachedTokens(10, 0, 4, 1); got != 5 {
		t.Fatalf("partial compatible cached tokens = %d, want 5", got)
	}
	if got := CompatibleCachedTokens(0, 8, 3, 0); got != 5 {
		t.Fatalf("cache_tokens compatible fallback = %d, want 5", got)
	}
}

func TestNormalizeRawFallbackTotalIncludesFineGrainedCache(t *testing.T) {
	payload := `{
	  "timestamp": "2026-04-25T00:00:00Z",
	  "source": "user@example.com",
	  "tokens": {
	    "input_tokens": 10,
	    "output_tokens": 20,
	    "reasoning_tokens": 3,
	    "cached_tokens": 10,
	    "cache_read_tokens": 4,
	    "cache_creation_tokens": 1
	  },
	  "model": "gpt-5.4",
	  "endpoint": "POST /v1/chat/completions"
	}`
	event, err := NormalizeRaw([]byte(payload))
	if err != nil {
		t.Fatalf("normalize fallback total: %v", err)
	}
	if event.TotalTokens != 43 {
		t.Fatalf("total tokens = %d, want 43", event.TotalTokens)
	}
}

func TestNormalizeRawSanitizesFailBodyForSummaryAndRawJSON(t *testing.T) {
	longBody := strings.Repeat("x", maxFailSummaryBytes+128)
	payload := `{
	  "timestamp": "2026-04-25T00:00:00Z",
	  "source": "user@example.com",
	  "tokens": {"input_tokens": 1, "total_tokens": 1},
	  "failed": true,
	  "fail": {
	    "status_code": 500,
	    "body": "Authorization: Bearer demo-token\napi_key=fake-test-api-key\naccess_token=demo-access-token\nCookie: session=demo-session\nalice@example.com ` + longBody + `"
	  },
	  "model": "gpt-5.4",
	  "endpoint": "POST /v1/chat/completions"
	}`
	event, err := NormalizeRaw([]byte(payload))
	if err != nil {
		t.Fatalf("normalize sensitive fail body: %v", err)
	}
	if event.FailBody == "" || !strings.Contains(event.FailBody, "fake-test-api-key") {
		t.Fatalf("raw fail body should be preserved internally = %q", event.FailBody)
	}
	if event.FailSummary == "" || len(event.FailSummary) > maxFailSummaryBytes+3 {
		t.Fatalf("fail summary length/content = %d %q", len(event.FailSummary), event.FailSummary)
	}
	for _, secret := range []string{
		"Bearer demo-token",
		"fake-test-api-key",
		"demo-access-token",
		"session=demo-session",
		"alice@example.com",
	} {
		if strings.Contains(event.FailSummary, secret) {
			t.Fatalf("fail summary contains secret %q: %q", secret, event.FailSummary)
		}
		if strings.Contains(event.RawJSON, secret) {
			t.Fatalf("raw json contains secret %q: %q", secret, event.RawJSON)
		}
	}
	if !strings.Contains(event.FailSummary, "[redacted]") {
		t.Fatalf("fail summary missing redaction marker: %q", event.FailSummary)
	}
}

func TestFailSummaryRedactionPreservesDiagnosticText(t *testing.T) {
	body := `AImproved fallback AIServer down {"cookie":"session=secret","status":"401","detail":"upstream denied","retry_after":30}`
	summary := FailSummaryFromBody(body)
	for _, want := range []string{
		"AImproved fallback",
		"AIServer down",
		`"status":"401"`,
		`"detail":"upstream denied"`,
		`"retry_after":30`,
	} {
		if !strings.Contains(summary, want) {
			t.Fatalf("summary missing %q: %q", want, summary)
		}
	}
	if strings.Contains(summary, "session=secret") {
		t.Fatalf("summary leaked cookie value: %q", summary)
	}
}

func TestNormalizeRawAcceptsPre7118UsagePayload(t *testing.T) {
	payload := `{
	  "timestamp": "2026-04-25T00:00:00Z",
	  "latency_ms": 1500,
	  "source": "user@example.com",
	  "auth_index": "0",
	  "tokens": {
	    "input_tokens": 10,
	    "output_tokens": 20,
	    "reasoning_tokens": 3,
	    "cached_tokens": 5,
	    "total_tokens": 33
	  },
	  "failed": false,
	  "provider": "openai",
	  "model": "gpt-5.4",
	  "endpoint": "POST /v1/chat/completions",
	  "auth_type": "apikey",
	  "api_key": "test-key",
	  "request_id": "ctx-request-id"
	}`
	event, err := NormalizeRaw([]byte(payload))
	if err != nil {
		t.Fatalf("normalize old payload: %v", err)
	}
	if event.ReasoningEffort != "" || event.CacheReadTokens != 0 ||
		event.CacheCreationTokens != 0 || event.FailStatusCode != 0 || event.FailBody != "" ||
		event.FailSummary != "" {
		t.Fatalf("old payload defaults = %#v", event)
	}
	if event.InputTokens != 10 || event.OutputTokens != 20 || event.ReasoningTokens != 3 ||
		event.CachedTokens != 5 || event.TotalTokens != 33 {
		t.Fatalf("old payload tokens = %#v", event)
	}
}

func TestNormalizeRawSplitsAliasAndResolvedModel(t *testing.T) {
	payload := `{
	  "timestamp": "2026-05-19T10:00:00Z",
	  "model": "gpt-5.5",
	  "alias": "gpt-5.4",
	  "endpoint": "POST /v1/chat/completions",
	  "input_tokens": 1,
	  "total_tokens": 1
	}`
	event, err := NormalizeRaw([]byte(payload))
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if event.RequestedModel != "gpt-5.4" {
		t.Fatalf("requested_model = %q, want gpt-5.4", event.RequestedModel)
	}
	if event.ResolvedModel != "gpt-5.5" {
		t.Fatalf("resolved_model = %q, want gpt-5.5", event.ResolvedModel)
	}
	if event.Model != "gpt-5.4" {
		t.Fatalf("model = %q, want gpt-5.4", event.Model)
	}
}

func TestNormalizeRawFallsBackToResolvedModelWhenAliasMissing(t *testing.T) {
	payload := `{
	  "timestamp": "2026-05-19T10:00:00Z",
	  "model": "gpt-4.1",
	  "endpoint": "POST /v1/chat/completions",
	  "input_tokens": 1,
	  "total_tokens": 1
	}`
	event, err := NormalizeRaw([]byte(payload))
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if event.RequestedModel != "" {
		t.Fatalf("requested_model = %q, want empty", event.RequestedModel)
	}
	if event.ResolvedModel != "gpt-4.1" {
		t.Fatalf("resolved_model = %q, want gpt-4.1", event.ResolvedModel)
	}
	if event.Model != "gpt-4.1" {
		t.Fatalf("model = %q, want gpt-4.1", event.Model)
	}
}

func TestBuildPayloadExposesResolvedModelOnDetails(t *testing.T) {
	event := Event{
		Timestamp:      "2026-05-19T10:00:00Z",
		Endpoint:       "POST /v1/chat/completions",
		Model:          "gpt-5.4",
		RequestedModel: "gpt-5.4",
		ResolvedModel:  "gpt-5.5",
	}
	payload := BuildPayload([]Event{event})
	api := payload.APIs["POST /v1/chat/completions"]
	if api == nil {
		t.Fatalf("missing endpoint aggregate")
	}
	modelEntry := api.Models["gpt-5.4"]
	if modelEntry == nil {
		t.Fatalf("aggregation key should be requested model gpt-5.4, got %#v", api.Models)
	}
	if len(modelEntry.Details) != 1 || modelEntry.Details[0].ResolvedModel != "gpt-5.5" {
		t.Fatalf("detail resolved_model = %#v", modelEntry.Details)
	}
}
