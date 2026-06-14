import i18n from '@/i18n';
import { maskApiKey } from './format';
import { normalizeAuthIndex } from './authIndex';
import { parseTimestampMs } from './timestamp';

export { normalizeAuthIndex };

export interface ModelPrice {
  prompt: number;
  completion: number;
  cache: number;
  cacheRead?: number;
  cacheCreation?: number;
  source?: string;
  sourceModelId?: string;
  rawJson?: string;
  updatedAtMs?: number;
  syncedAtMs?: number;
}

export interface UsageTokens {
  input_tokens?: number;
  output_tokens?: number;
  reasoning_tokens?: number;
  cached_tokens?: number;
  cache_tokens?: number;
  cache_read_tokens?: number;
  cache_creation_tokens?: number;
  total_tokens?: number;
}

export interface UsageDetail {
  timestamp: string;
  source: string;
  auth_index: string | number | null;
  api_key_hash?: string;
  apiKeyHash?: string;
  account_snapshot?: string;
  accountSnapshot?: string;
  auth_label_snapshot?: string;
  authLabelSnapshot?: string;
  auth_file_snapshot?: string;
  authFileSnapshot?: string;
  auth_provider_snapshot?: string;
  authProviderSnapshot?: string;
  auth_project_id_snapshot?: string;
  authProjectIdSnapshot?: string;
  auth_snapshot_at_ms?: number;
  authSnapshotAtMs?: number;
  reasoning_effort?: string;
  reasoningEffort?: string;
  service_tier?: string;
  serviceTier?: string;
  executor_type?: string;
  executorType?: string;
  latency_ms?: number;
  ttft_ms?: number;
  tokens: UsageTokens;
  failed: boolean;
  fail_status_code?: number | null;
  failStatusCode?: number | null;
  fail_summary?: string;
  failSummary?: string;
  fail_body?: string;
  failBody?: string;
  __modelName?: string;
  __resolvedModel?: string;
  __timestampMs?: number;
}

export interface UsageDetailWithEndpoint extends UsageDetail {
  __endpoint: string;
  __endpointMethod?: string;
  __endpointPath?: string;
  __timestampMs: number;
}

export interface DurationFormatOptions {
  maxUnits?: number;
  invalidText?: string;
  secondDecimals?: number | 'auto';
  locale?: string;
}

const TOKENS_PER_PRICE_UNIT = 1_000_000;
const MODEL_PRICE_STORAGE_KEY = 'cli-proxy-model-prices-v2';
const USAGE_ENDPOINT_METHOD_REGEX = /^(GET|POST|PUT|PATCH|DELETE|OPTIONS|HEAD)\s+(\S+)/i;
const USAGE_SOURCE_PREFIX_KEY = 'k:';
const USAGE_SOURCE_PREFIX_MASKED = 'm:';
const USAGE_SOURCE_PREFIX_TEXT = 't:';
const KEY_LIKE_TOKEN_REGEX =
  /(sk-proj-[A-Za-z0-9-_]{6,}|sk-ant-[A-Za-z0-9-_]{6,}|sk-[A-Za-z0-9-_]{6,}|sess-[A-Za-z0-9-_]{6,}|ghp_[A-Za-z0-9]{6,}|github_pat_[A-Za-z0-9_]{20,}|AIza[0-9A-Za-z-_]{8,}|hf_[A-Za-z0-9]{6,}|pk_[A-Za-z0-9]{6,}|rk_[A-Za-z0-9]{6,})/;
const MASKED_TOKEN_HINT_REGEX = /^[^\s]{1,24}(\*{2,}|\.{3})[^\s]{1,24}$/;
const BACKEND_MASKED_SOURCE_REGEX = /^m:(\*{4}|[^\s/\\]{4}\.\.\.[^\s/\\]{4})$/;

const keyFingerprintCache = new Map<string, string>();
const usageDetailsCache = new WeakMap<object, UsageDetail[]>();
const usageDetailsWithEndpointCache = new WeakMap<object, UsageDetailWithEndpoint[]>();

const isRecord = (value: unknown): value is Record<string, unknown> =>
  value !== null && typeof value === 'object' && !Array.isArray(value);

const toFiniteNumber = (value: unknown): number => {
  const numberValue = typeof value === 'number' ? value : Number(value);
  return Number.isFinite(numberValue) ? numberValue : 0;
};

const toPositiveNumber = (value: unknown): number | undefined => {
  const numberValue = toFiniteNumber(value);
  return numberValue > 0 ? numberValue : undefined;
};

const toOptionalNumber = (value: unknown): number | undefined => {
  if (value === null || value === undefined || (typeof value === 'string' && value.trim() === '')) {
    return undefined;
  }
  const numberValue = toFiniteNumber(value);
  return Number.isFinite(numberValue) ? numberValue : undefined;
};

const readDetailString = (value: unknown): string | undefined => {
  if (value === null || value === undefined) return undefined;
  const text = String(value).trim();
  return text || undefined;
};

export const compatibleCachedTokens = (
  cachedTokens: unknown,
  cacheTokens: unknown,
  cacheReadTokens: unknown,
  cacheCreationTokens: unknown
): number => {
  const cached = Math.max(
    Math.max(toFiniteNumber(cachedTokens), 0),
    Math.max(toFiniteNumber(cacheTokens), 0)
  );
  if (cached <= 0) return 0;
  const fineGrained =
    Math.max(toFiniteNumber(cacheReadTokens), 0) + Math.max(toFiniteNumber(cacheCreationTokens), 0);
  return Math.max(cached - fineGrained, 0);
};

const getApisRecord = (usageData: unknown): Record<string, unknown> | null => {
  const usageRecord = isRecord(usageData) ? usageData : null;
  const apisRaw = usageRecord ? usageRecord.apis : null;
  return isRecord(apisRaw) ? apisRaw : null;
};

const fnv1a64Hex = (value: string): string => {
  const cached = keyFingerprintCache.get(value);
  if (cached) return cached;

  const FNV_OFFSET_BASIS = 0xcbf29ce484222325n;
  const FNV_PRIME = 0x100000001b3n;

  let hash = FNV_OFFSET_BASIS;
  for (let i = 0; i < value.length; i += 1) {
    hash ^= BigInt(value.charCodeAt(i));
    hash = (hash * FNV_PRIME) & 0xffffffffffffffffn;
  }

  const hex = hash.toString(16).padStart(16, '0');
  keyFingerprintCache.set(value, hex);
  return hex;
};

const looksLikeRawSecret = (text: string): boolean => {
  if (!text || /\s/.test(text)) return false;

  const lower = text.toLowerCase();
  if (lower.endsWith('.json')) return false;
  if (lower.startsWith('http://') || lower.startsWith('https://')) return false;
  if (/[\\/]/.test(text)) return false;
  if (KEY_LIKE_TOKEN_REGEX.test(text)) return true;
  if (text.length >= 32 && text.length <= 512) return true;
  if (text.length >= 16 && text.length < 32 && /^[A-Za-z0-9._=-]+$/.test(text)) {
    return /[A-Za-z]/.test(text) && /\d/.test(text);
  }
  return false;
};

const extractRawSecretFromText = (text: string): string | null => {
  if (!text) return null;
  if (looksLikeRawSecret(text)) return text;

  const keyLikeMatch = text.match(KEY_LIKE_TOKEN_REGEX);
  if (keyLikeMatch?.[0]) return keyLikeMatch[0];

  const queryMatch = text.match(
    /(?:[?&])(api[-_]?key|key|token|access_token|authorization)=([^&#\s]+)/i
  );
  const queryValue = queryMatch?.[2];
  if (queryValue && looksLikeRawSecret(queryValue)) return queryValue;

  const headerMatch = text.match(
    /(api[-_]?key|key|token|access[-_]?token|authorization)\s*[:=]\s*([A-Za-z0-9._=-]+)/i
  );
  const headerValue = headerMatch?.[2];
  if (headerValue && looksLikeRawSecret(headerValue)) return headerValue;

  const bearerMatch = text.match(/\bBearer\s+([A-Za-z0-9._=-]{6,})/i);
  const bearerValue = bearerMatch?.[1];
  return bearerValue && looksLikeRawSecret(bearerValue) ? bearerValue : null;
};

export function maskUsageSecretSource(secret: string): string {
  const trimmed = String(secret || '').trim();
  if (!trimmed) return '';

  if (trimmed.length <= 8) return '****';
  return `${trimmed.slice(0, 4)}...${trimmed.slice(-4)}`;
}

export function normalizeUsageSourceId(
  value: unknown,
  masker: (val: string) => string = maskApiKey
): string {
  const raw =
    typeof value === 'string' ? value : value === null || value === undefined ? '' : String(value);
  const trimmed = raw.trim();
  if (!trimmed) return '';
  if (trimmed.startsWith(USAGE_SOURCE_PREFIX_KEY)) return trimmed;
  if (trimmed.startsWith(USAGE_SOURCE_PREFIX_MASKED)) {
    if (BACKEND_MASKED_SOURCE_REGEX.test(trimmed)) return trimmed;
    const maskedValue = trimmed.slice(USAGE_SOURCE_PREFIX_MASKED.length).trim();
    const extracted = extractRawSecretFromText(maskedValue) || extractRawSecretFromText(trimmed);
    return extracted ? `${USAGE_SOURCE_PREFIX_KEY}${fnv1a64Hex(extracted)}` : trimmed;
  }
  if (trimmed.startsWith(USAGE_SOURCE_PREFIX_TEXT)) {
    const textSource = trimmed.slice(USAGE_SOURCE_PREFIX_TEXT.length).trim();
    const extracted = extractRawSecretFromText(textSource);
    return extracted ? `${USAGE_SOURCE_PREFIX_KEY}${fnv1a64Hex(extracted)}` : trimmed;
  }

  const extracted = extractRawSecretFromText(trimmed);
  if (extracted) return `${USAGE_SOURCE_PREFIX_KEY}${fnv1a64Hex(extracted)}`;
  if (MASKED_TOKEN_HINT_REGEX.test(trimmed)) {
    return `${USAGE_SOURCE_PREFIX_MASKED}${masker(trimmed)}`;
  }
  return `${USAGE_SOURCE_PREFIX_TEXT}${trimmed}`;
}

export function buildCandidateUsageSourceIds(input: {
  apiKey?: string;
  prefix?: string;
}): string[] {
  const result: string[] = [];
  const prefix = input.prefix?.trim();
  if (prefix) result.push(`${USAGE_SOURCE_PREFIX_TEXT}${prefix}`);

  const apiKey = input.apiKey?.trim();
  if (apiKey) {
    result.push(normalizeUsageSourceId(apiKey));
    result.push(`${USAGE_SOURCE_PREFIX_MASKED}${maskUsageSecretSource(apiKey)}`);
    result.push(`${USAGE_SOURCE_PREFIX_MASKED}${maskApiKey(apiKey)}`);
    result.push(`${USAGE_SOURCE_PREFIX_TEXT}${maskApiKey(apiKey)}`);
  }

  return Array.from(new Set(result.filter(Boolean)));
}

export function extractLatencyMs(detail: unknown): number | null {
  const record = isRecord(detail) ? detail : null;
  const rawValue = record?.latency_ms ?? record?.latencyMs;
  if (
    rawValue === null ||
    rawValue === undefined ||
    (typeof rawValue === 'string' && rawValue.trim() === '')
  ) {
    return null;
  }

  const parsed = Number(rawValue);
  return Number.isFinite(parsed) && parsed >= 0 ? parsed : null;
}

export function extractTTFTMs(detail: unknown): number | null {
  const record = isRecord(detail) ? detail : null;
  const rawValue =
    record?.ttft_ms ??
    record?.ttftMs ??
    record?.time_to_first_token_ms ??
    record?.timeToFirstTokenMs;
  if (
    rawValue === null ||
    rawValue === undefined ||
    (typeof rawValue === 'string' && rawValue.trim() === '')
  ) {
    return null;
  }

  const parsed = Number(rawValue);
  return Number.isFinite(parsed) && parsed >= 0 ? parsed : null;
}

const readTokens = (detail: Record<string, unknown>): UsageTokens => {
  const tokensRaw = isRecord(detail.tokens) ? detail.tokens : {};
  const cacheReadTokens = toFiniteNumber(tokensRaw.cache_read_tokens ?? tokensRaw.cacheReadTokens);
  const cacheCreationTokens = toFiniteNumber(
    tokensRaw.cache_creation_tokens ?? tokensRaw.cacheCreationTokens
  );
  const cachedTokens = compatibleCachedTokens(
    tokensRaw.cached_tokens ?? tokensRaw.cachedTokens,
    tokensRaw.cache_tokens ?? tokensRaw.cacheTokens,
    cacheReadTokens,
    cacheCreationTokens
  );
  const inputTokens = toFiniteNumber(tokensRaw.input_tokens ?? tokensRaw.inputTokens);
  const outputTokens = toFiniteNumber(tokensRaw.output_tokens ?? tokensRaw.outputTokens);
  const reasoningTokens = toFiniteNumber(tokensRaw.reasoning_tokens ?? tokensRaw.reasoningTokens);
  const explicitTotalTokens = toFiniteNumber(tokensRaw.total_tokens ?? tokensRaw.totalTokens);
  const totalTokens =
    explicitTotalTokens > 0
      ? explicitTotalTokens
      : inputTokens +
        outputTokens +
        reasoningTokens +
        cachedTokens +
        cacheReadTokens +
        cacheCreationTokens;
  return {
    input_tokens: inputTokens,
    output_tokens: outputTokens,
    reasoning_tokens: reasoningTokens,
    cached_tokens: cachedTokens,
    cache_tokens: cachedTokens,
    cache_read_tokens: cacheReadTokens,
    cache_creation_tokens: cacheCreationTokens,
    total_tokens: totalTokens,
  };
};

const normalizeSourceWithCache = (sourceCache: Map<string, string>, value: unknown): string => {
  const raw =
    typeof value === 'string' ? value : value === null || value === undefined ? '' : String(value);
  const trimmed = raw.trim();
  if (!trimmed) return '';

  const cached = sourceCache.get(trimmed);
  if (cached !== undefined) return cached;

  const normalized = normalizeUsageSourceId(trimmed);
  sourceCache.set(trimmed, normalized);
  return normalized;
};

export function collectUsageDetails(usageData: unknown): UsageDetail[] {
  const cacheKey = isRecord(usageData) ? (usageData as object) : null;
  if (cacheKey) {
    const cached = usageDetailsCache.get(cacheKey);
    if (cached) return cached;
  }

  const apis = getApisRecord(usageData);
  if (!apis) return [];

  const details: UsageDetail[] = [];
  const sourceCache = new Map<string, string>();

  Object.values(apis).forEach((apiEntry) => {
    if (!isRecord(apiEntry)) return;
    const models = isRecord(apiEntry.models) ? apiEntry.models : null;
    if (!models) return;

    Object.entries(models).forEach(([modelName, modelEntry]) => {
      if (!isRecord(modelEntry)) return;
      const modelDetails = Array.isArray(modelEntry.details) ? modelEntry.details : [];

      modelDetails.forEach((detailRaw) => {
        if (!isRecord(detailRaw) || typeof detailRaw.timestamp !== 'string') return;
        const timestamp = detailRaw.timestamp;
        const timestampMs = parseTimestampMs(timestamp);
        const latencyMs = extractLatencyMs(detailRaw);
        const ttftMs = extractTTFTMs(detailRaw);
        const failRaw = isRecord(detailRaw.fail) ? detailRaw.fail : {};
        details.push({
          timestamp,
          source: normalizeSourceWithCache(sourceCache, detailRaw.source),
          auth_index: (detailRaw.auth_index ??
            detailRaw.authIndex ??
            detailRaw.AuthIndex ??
            null) as UsageDetail['auth_index'],
          api_key_hash: readDetailString(detailRaw.api_key_hash ?? detailRaw.apiKeyHash),
          account_snapshot: readDetailString(
            detailRaw.account_snapshot ?? detailRaw.accountSnapshot
          ),
          auth_label_snapshot: readDetailString(
            detailRaw.auth_label_snapshot ?? detailRaw.authLabelSnapshot
          ),
          auth_file_snapshot: readDetailString(
            detailRaw.auth_file_snapshot ?? detailRaw.authFileSnapshot
          ),
          auth_provider_snapshot: readDetailString(
            detailRaw.auth_provider_snapshot ?? detailRaw.authProviderSnapshot
          ),
          auth_project_id_snapshot: readDetailString(
            detailRaw.auth_project_id_snapshot ?? detailRaw.authProjectIdSnapshot
          ),
          auth_snapshot_at_ms: toPositiveNumber(
            detailRaw.auth_snapshot_at_ms ?? detailRaw.authSnapshotAtMs
          ),
          reasoning_effort: readDetailString(
            detailRaw.reasoning_effort ?? detailRaw.reasoningEffort
          ),
          service_tier: readDetailString(detailRaw.service_tier ?? detailRaw.serviceTier),
          executor_type: readDetailString(detailRaw.executor_type ?? detailRaw.executorType),
          latency_ms: latencyMs ?? undefined,
          ttft_ms: ttftMs ?? undefined,
          tokens: readTokens(detailRaw),
          failed: detailRaw.failed === true,
          fail_status_code:
            toOptionalNumber(
              detailRaw.fail_status_code ??
                detailRaw.failStatusCode ??
                failRaw.status_code ??
                failRaw.statusCode
            ) ?? null,
          fail_summary: readDetailString(detailRaw.fail_summary ?? detailRaw.failSummary),
          fail_body: readDetailString(detailRaw.fail_body ?? detailRaw.failBody ?? failRaw.body),
          __modelName: modelName,
          __resolvedModel: readDetailString(detailRaw.resolved_model ?? detailRaw.resolvedModel),
          __timestampMs: Number.isNaN(timestampMs) ? 0 : timestampMs,
        });
      });
    });
  });

  if (cacheKey) usageDetailsCache.set(cacheKey, details);
  return details;
}

export function collectUsageDetailsWithEndpoint(usageData: unknown): UsageDetailWithEndpoint[] {
  const cacheKey = isRecord(usageData) ? (usageData as object) : null;
  if (cacheKey) {
    const cached = usageDetailsWithEndpointCache.get(cacheKey);
    if (cached) return cached;
  }

  const apis = getApisRecord(usageData);
  if (!apis) return [];

  const details: UsageDetailWithEndpoint[] = [];
  const sourceCache = new Map<string, string>();

  Object.entries(apis).forEach(([endpoint, apiEntry]) => {
    if (!isRecord(apiEntry)) return;
    const models = isRecord(apiEntry.models) ? apiEntry.models : null;
    if (!models) return;

    const endpointMatch = endpoint.match(USAGE_ENDPOINT_METHOD_REGEX);
    const endpointMethod = endpointMatch?.[1]?.toUpperCase();
    const endpointPath = endpointMatch?.[2];

    Object.entries(models).forEach(([modelName, modelEntry]) => {
      if (!isRecord(modelEntry)) return;
      const modelDetails = Array.isArray(modelEntry.details) ? modelEntry.details : [];

      modelDetails.forEach((detailRaw) => {
        if (!isRecord(detailRaw) || typeof detailRaw.timestamp !== 'string') return;
        const timestamp = detailRaw.timestamp;
        const timestampMs = parseTimestampMs(timestamp);
        const latencyMs = extractLatencyMs(detailRaw);
        const ttftMs = extractTTFTMs(detailRaw);
        const failRaw = isRecord(detailRaw.fail) ? detailRaw.fail : {};
        details.push({
          timestamp,
          source: normalizeSourceWithCache(sourceCache, detailRaw.source),
          auth_index: (detailRaw.auth_index ??
            detailRaw.authIndex ??
            detailRaw.AuthIndex ??
            null) as UsageDetail['auth_index'],
          api_key_hash: readDetailString(detailRaw.api_key_hash ?? detailRaw.apiKeyHash),
          account_snapshot: readDetailString(
            detailRaw.account_snapshot ?? detailRaw.accountSnapshot
          ),
          auth_label_snapshot: readDetailString(
            detailRaw.auth_label_snapshot ?? detailRaw.authLabelSnapshot
          ),
          auth_file_snapshot: readDetailString(
            detailRaw.auth_file_snapshot ?? detailRaw.authFileSnapshot
          ),
          auth_provider_snapshot: readDetailString(
            detailRaw.auth_provider_snapshot ?? detailRaw.authProviderSnapshot
          ),
          auth_project_id_snapshot: readDetailString(
            detailRaw.auth_project_id_snapshot ?? detailRaw.authProjectIdSnapshot
          ),
          auth_snapshot_at_ms: toPositiveNumber(
            detailRaw.auth_snapshot_at_ms ?? detailRaw.authSnapshotAtMs
          ),
          reasoning_effort: readDetailString(
            detailRaw.reasoning_effort ?? detailRaw.reasoningEffort
          ),
          service_tier: readDetailString(detailRaw.service_tier ?? detailRaw.serviceTier),
          executor_type: readDetailString(detailRaw.executor_type ?? detailRaw.executorType),
          latency_ms: latencyMs ?? undefined,
          ttft_ms: ttftMs ?? undefined,
          tokens: readTokens(detailRaw),
          failed: detailRaw.failed === true,
          fail_status_code:
            toOptionalNumber(
              detailRaw.fail_status_code ??
                detailRaw.failStatusCode ??
                failRaw.status_code ??
                failRaw.statusCode
            ) ?? null,
          fail_summary: readDetailString(detailRaw.fail_summary ?? detailRaw.failSummary),
          fail_body: readDetailString(detailRaw.fail_body ?? detailRaw.failBody ?? failRaw.body),
          __modelName: modelName,
          __resolvedModel: readDetailString(detailRaw.resolved_model ?? detailRaw.resolvedModel),
          __endpoint: endpoint,
          __endpointMethod: endpointMethod,
          __endpointPath: endpointPath,
          __timestampMs: Number.isNaN(timestampMs) ? 0 : timestampMs,
        });
      });
    });
  });

  if (cacheKey) usageDetailsWithEndpointCache.set(cacheKey, details);
  return details;
}

export function extractTotalTokens(detail: unknown): number {
  const record = isRecord(detail) ? detail : null;
  const tokens = record && isRecord(record.tokens) ? record.tokens : {};
  const explicitTotal = toFiniteNumber(tokens.total_tokens ?? tokens.totalTokens);
  if (explicitTotal > 0) return explicitTotal;

  const inputTokens = toFiniteNumber(tokens.input_tokens ?? tokens.inputTokens);
  const outputTokens = toFiniteNumber(tokens.output_tokens ?? tokens.outputTokens);
  const reasoningTokens = toFiniteNumber(tokens.reasoning_tokens ?? tokens.reasoningTokens);
  const cachedTokens = compatibleCachedTokens(
    tokens.cached_tokens ?? tokens.cachedTokens,
    tokens.cache_tokens ?? tokens.cacheTokens,
    tokens.cache_read_tokens ?? tokens.cacheReadTokens,
    tokens.cache_creation_tokens ?? tokens.cacheCreationTokens
  );
  const cacheReadTokens = Math.max(
    toFiniteNumber(tokens.cache_read_tokens ?? tokens.cacheReadTokens),
    0
  );
  const cacheCreationTokens = Math.max(
    toFiniteNumber(tokens.cache_creation_tokens ?? tokens.cacheCreationTokens),
    0
  );

  return (
    inputTokens +
    outputTokens +
    reasoningTokens +
    cachedTokens +
    cacheReadTokens +
    cacheCreationTokens
  );
}

export function calculateCost(
  detail: Pick<UsageDetail, 'tokens' | '__modelName' | '__resolvedModel'>,
  modelPrices: Record<string, ModelPrice>
): number {
  const resolvedModel = detail.__resolvedModel || '';
  const requestedModel = detail.__modelName || '';
  const price = modelPrices[resolvedModel] || modelPrices[requestedModel];
  if (!price) return 0;

  const inputTokens = Math.max(toFiniteNumber(detail.tokens.input_tokens), 0);
  const completionTokens = Math.max(toFiniteNumber(detail.tokens.output_tokens), 0);
  const cachedTokens = Math.max(
    Math.max(toFiniteNumber(detail.tokens.cached_tokens), 0),
    Math.max(toFiniteNumber(detail.tokens.cache_tokens), 0)
  );
  const cacheReadTokens = Math.max(toFiniteNumber(detail.tokens.cache_read_tokens), 0);
  const cacheCreationTokens = Math.max(toFiniteNumber(detail.tokens.cache_creation_tokens), 0);
  const promptPrice = Number(price.prompt) || 0;
  const completionPrice = Number(price.completion) || 0;
  if (cacheReadTokens > 0 || cacheCreationTokens > 0) {
    const cacheReadPrice = Number(price.cacheRead) || Number(price.cache) || 0;
    const cacheCreationPrice = Number(price.cacheCreation) || promptPrice;
    const promptTokens = Math.max(inputTokens - cachedTokens, 0);
    const total =
      (promptTokens / TOKENS_PER_PRICE_UNIT) * promptPrice +
      (completionTokens / TOKENS_PER_PRICE_UNIT) * completionPrice +
      (cachedTokens / TOKENS_PER_PRICE_UNIT) * (Number(price.cache) || 0) +
      (cacheReadTokens / TOKENS_PER_PRICE_UNIT) * cacheReadPrice +
      (cacheCreationTokens / TOKENS_PER_PRICE_UNIT) * cacheCreationPrice;
    return Number.isFinite(total) && total > 0 ? total : 0;
  }

  const promptTokens = Math.max(inputTokens - cachedTokens, 0);
  const promptCost = (promptTokens / TOKENS_PER_PRICE_UNIT) * promptPrice;
  const completionCost = (completionTokens / TOKENS_PER_PRICE_UNIT) * completionPrice;
  const cachedCost = (cachedTokens / TOKENS_PER_PRICE_UNIT) * (Number(price.cache) || 0);
  const total = promptCost + cachedCost + completionCost;
  return Number.isFinite(total) && total > 0 ? total : 0;
}

export function loadModelPrices(): Record<string, ModelPrice> {
  try {
    if (typeof localStorage === 'undefined') return {};
    const raw = localStorage.getItem(MODEL_PRICE_STORAGE_KEY);
    if (!raw) return {};

    const parsed: unknown = JSON.parse(raw);
    if (!isRecord(parsed)) return {};

    const normalized: Record<string, ModelPrice> = {};
    Object.entries(parsed).forEach(([model, price]) => {
      if (!model || !isRecord(price)) return;

      const prompt = toFiniteNumber(price.prompt);
      const completion = toFiniteNumber(price.completion);
      const cacheRaw = Number(price.cache);
      const cache = Number.isFinite(cacheRaw) && cacheRaw >= 0 ? cacheRaw : prompt;
      const cacheReadRaw = Number(price.cacheRead);
      const cacheRead = Number.isFinite(cacheReadRaw) && cacheReadRaw >= 0 ? cacheReadRaw : 0;
      const cacheCreationRaw = Number(price.cacheCreation);
      const cacheCreation =
        Number.isFinite(cacheCreationRaw) && cacheCreationRaw >= 0 ? cacheCreationRaw : 0;

      if (prompt < 0 || completion < 0 || cache < 0 || cacheRead < 0 || cacheCreation < 0) return;
      normalized[model] = {
        prompt,
        completion,
        cache,
        cacheRead,
        cacheCreation,
        source: readDetailString(price.source),
        sourceModelId: readDetailString(price.sourceModelId),
        rawJson: readDetailString(price.rawJson),
        updatedAtMs: toPositiveNumber(price.updatedAtMs),
        syncedAtMs: toPositiveNumber(price.syncedAtMs),
      };
    });

    return normalized;
  } catch {
    return {};
  }
}

export function saveModelPrices(prices: Record<string, ModelPrice>): void {
  try {
    if (typeof localStorage === 'undefined') return;
    localStorage.setItem(MODEL_PRICE_STORAGE_KEY, JSON.stringify(prices));
  } catch {
    // Ignore storage failures; pricing is an optional browser-side aid.
  }
}

export function clearModelPrices(): void {
  try {
    if (typeof localStorage === 'undefined') return;
    localStorage.removeItem(MODEL_PRICE_STORAGE_KEY);
  } catch {
    // Ignore storage failures; pricing is optional fallback data.
  }
}

export function formatCompactNumber(value: number): string {
  const num = Number(value);
  if (!Number.isFinite(num)) return '0';

  const abs = Math.abs(num);
  if (abs === 0) return '0';
  if (abs >= 1_000_000) return `${(num / 1_000_000).toFixed(1)}M`;
  if (abs >= 1_000) return `${(num / 1_000).toFixed(1)}K`;
  return abs >= 1 ? num.toFixed(0) : num.toFixed(2);
}

export function formatUsd(value: number): string {
  const num = Number(value);
  if (!Number.isFinite(num)) return '$0.00';

  const fixed = num.toFixed(2);
  const parts = Number(fixed).toLocaleString(undefined, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });
  return `$${parts}`;
}

const resolveDurationLocale = (locale?: string): string | undefined =>
  locale?.trim() || i18n.resolvedLanguage || i18n.language || undefined;

const formatDurationNumber = (
  value: number,
  locale: string | undefined,
  options: Intl.NumberFormatOptions = {}
): string => {
  try {
    return new Intl.NumberFormat(locale, {
      useGrouping: false,
      ...options,
    }).format(value);
  } catch {
    return String(value);
  }
};

const getDurationUnitLabel = (unit: 'd' | 'h' | 'm' | 's' | 'ms'): string =>
  i18n.t(`usage_stats.duration_unit_${unit}`, { defaultValue: unit });

const formatDurationPart = (
  value: number,
  unit: 'd' | 'h' | 'm' | 's' | 'ms',
  locale: string | undefined,
  options: Intl.NumberFormatOptions = {}
): string => `${formatDurationNumber(value, locale, options)}${getDurationUnitLabel(unit)}`;

const normalizeDurationMaxUnits = (value: number | undefined): number => {
  const parsed = Number(value);
  if (!Number.isFinite(parsed) || parsed <= 0) return 2;
  return Math.min(Math.floor(parsed), 4);
};

const resolveSecondDecimalPlaces = (
  seconds: number,
  secondDecimals: number | 'auto' | undefined
): number => {
  if (secondDecimals === 'auto' || secondDecimals === undefined) return seconds < 10 ? 2 : 1;

  const parsed = Math.floor(Number(secondDecimals));
  if (!Number.isFinite(parsed) || parsed < 0) return seconds < 10 ? 2 : 1;
  return Math.min(parsed, 3);
};

export function formatDurationMs(
  value: number | null | undefined,
  options: DurationFormatOptions = {}
): string {
  const invalidText = options.invalidText ?? '--';
  if (value === null || value === undefined) return invalidText;

  const parsed = Number(value);
  if (!Number.isFinite(parsed) || parsed < 0) return invalidText;

  const locale = resolveDurationLocale(options.locale);
  if (parsed < 1000) return formatDurationPart(Math.round(parsed), 'ms', locale);

  const seconds = parsed / 1000;
  if (seconds < 60) {
    const secondDecimalPlaces = resolveSecondDecimalPlaces(seconds, options.secondDecimals);
    return formatDurationPart(seconds, 's', locale, {
      minimumFractionDigits: 0,
      maximumFractionDigits: secondDecimalPlaces,
    });
  }

  const totalSeconds = Math.floor(seconds);
  let remainingSeconds = totalSeconds;
  const days = Math.floor(remainingSeconds / 86_400);
  remainingSeconds -= days * 86_400;
  const hours = Math.floor(remainingSeconds / 3_600);
  remainingSeconds -= hours * 3_600;
  const minutes = Math.floor(remainingSeconds / 60);
  remainingSeconds -= minutes * 60;

  const parts = [
    { unit: 'd' as const, value: days },
    { unit: 'h' as const, value: hours },
    { unit: 'm' as const, value: minutes },
    { unit: 's' as const, value: remainingSeconds },
  ].filter((part) => part.value > 0);

  if (!parts.length) return formatDurationPart(0, 's', locale);

  return parts
    .slice(0, normalizeDurationMaxUnits(options.maxUnits))
    .map((part, index) =>
      formatDurationPart(part.value, part.unit, locale, {
        minimumIntegerDigits: index > 0 && (part.unit === 'm' || part.unit === 's') ? 2 : 1,
        maximumFractionDigits: 0,
      })
    )
    .join(' ');
}
