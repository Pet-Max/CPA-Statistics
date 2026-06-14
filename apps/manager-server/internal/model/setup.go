package model

import "fmt"

type Setup struct {
	CPAUpstreamURL       string        `json:"cpaBaseUrl"`
	ManagementKey        string        `json:"managementKey,omitempty"`
	CPAUpstreamURLDigest *SecretDigest `json:"cpaBaseUrlDigest,omitempty"`
	ManagementKeyDigest  *SecretDigest `json:"managementKeyDigest,omitempty"`
	Queue                string        `json:"queue,omitempty"`
	PopSide              string        `json:"popSide,omitempty"`
}

func (s Setup) String() string {
	return fmt.Sprintf("upstream=%s queue=%s popSide=%s", s.CPAUpstreamURL, s.Queue, s.PopSide)
}
