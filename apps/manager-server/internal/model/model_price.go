package model

type ModelPrice struct {
	Prompt        float64 `json:"prompt"`
	Completion    float64 `json:"completion"`
	Cache         float64 `json:"cache"`
	CacheRead     float64 `json:"cacheRead,omitempty"`
	CacheCreation float64 `json:"cacheCreation,omitempty"`
	Source        string  `json:"source,omitempty"`
	SourceModelID string  `json:"sourceModelId,omitempty"`
	RawJSON       string  `json:"rawJson,omitempty"`
	UpdatedAtMS   int64   `json:"updatedAtMs,omitempty"`
	SyncedAtMS    *int64  `json:"syncedAtMs,omitempty"`
}

type ModelPriceSyncResult struct {
	Imported int `json:"imported"`
	Skipped  int `json:"skipped"`
}

type ModelPriceSyncRequest struct {
	Models []string `json:"models,omitempty"`
}

type ModelPriceSyncCandidate struct {
	SourceModelID string     `json:"sourceModelId"`
	Score         float64    `json:"score"`
	Reason        string     `json:"reason"`
	Price         ModelPrice `json:"price"`
}

type ModelPriceSyncCandidateSet struct {
	Model      string                    `json:"model"`
	Candidates []ModelPriceSyncCandidate `json:"candidates"`
}

type ModelPriceSyncSourceResult struct {
	Source  string `json:"source"`
	Models  int    `json:"models"`
	Skipped int    `json:"skipped"`
	Error   string `json:"error,omitempty"`
}

type ModelPriceSyncResponse struct {
	Prices        map[string]ModelPrice          `json:"prices"`
	Imported      int                            `json:"imported"`
	Skipped       int                            `json:"skipped"`
	Candidates    []ModelPriceSyncCandidateSet   `json:"candidates"`
	Unmatched     []string                       `json:"unmatched"`
	SourceResults []ModelPriceSyncSourceResult   `json:"sourceResults"`
	ProxyUsed     bool                           `json:"proxyUsed"`
}
