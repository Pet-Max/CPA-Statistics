package model

type AdminCredential struct {
	Version     int    `json:"version"`
	Algorithm   string `json:"algorithm,omitempty"`
	Salt        string `json:"salt"`
	KeyHash     string `json:"keyHash"`
	Iterations  int    `json:"iterations"`
	CreatedAtMS int64  `json:"createdAtMs"`
	RotatedAtMS int64  `json:"rotatedAtMs,omitempty"`
	Source      string `json:"source,omitempty"`
}

type SecretDigest struct {
	Version     int    `json:"version"`
	Algorithm   string `json:"algorithm"`
	Salt        string `json:"salt"`
	Hash        string `json:"hash"`
	Iterations  int    `json:"iterations"`
	CreatedAtMS int64  `json:"createdAtMs"`
}

type BootstrapState struct {
	Version            int    `json:"version"`
	Status             string `json:"status"`
	AdminReady         bool   `json:"adminReady"`
	ProjectInitialized bool   `json:"projectInitialized"`
	DataKeyReady       bool   `json:"dataKeyReady"`
	MigratedLegacy     bool   `json:"migratedLegacy"`
	HasHistoricalData  bool   `json:"hasHistoricalData"`
	UpdatedAtMS        int64  `json:"updatedAtMs"`
}
