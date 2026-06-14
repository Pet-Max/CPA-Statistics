package model

type ManagerConfig struct {
	CPAConnection                 ManagerCPAConnectionConfig        `json:"cpaConnection"`
	Collector                     ManagerCollectorConfig            `json:"collector"`
	ExternalUsageService          ManagerExternalUsageServiceConfig `json:"externalUsageService"`
	ExternalUsageServiceSubmitted bool                              `json:"-"`
	UpdatedAtMS                   int64                             `json:"updatedAtMs,omitempty"`
}

type ManagerCPAConnectionConfig struct {
	CPABaseURL          string        `json:"cpaBaseUrl"`
	ManagementKey       string        `json:"managementKey,omitempty"`
	CPABaseURLDigest    *SecretDigest `json:"cpaBaseUrlDigest,omitempty"`
	ManagementKeyDigest *SecretDigest `json:"managementKeyDigest,omitempty"`
}

type ManagerCollectorConfig struct {
	Enabled        *bool  `json:"enabled,omitempty"`
	CollectorMode  string `json:"collectorMode,omitempty"`
	Queue          string `json:"queue,omitempty"`
	PopSide        string `json:"popSide,omitempty"`
	BatchSize      int    `json:"batchSize,omitempty"`
	PollIntervalMS int    `json:"pollIntervalMs,omitempty"`
	QueryLimit     int    `json:"queryLimit,omitempty"`
	TLSSkipVerify  bool   `json:"tlsSkipVerify,omitempty"`
}

type ManagerExternalUsageServiceConfig struct {
	Enabled     bool   `json:"enabled"`
	ServiceBase string `json:"serviceBase,omitempty"`
}
