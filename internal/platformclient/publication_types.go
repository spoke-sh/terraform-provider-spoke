package platformclient

// These types follow Platform's owned HTTP schema. Fingerprints are opaque
// server commitments; the client does not implement Platform's native codec.
type PublicationPolicy struct {
	Name               string `json:"name"`
	DeletionProtection bool   `json:"deletion_protection"`
	RetentionDays      int64  `json:"retention_days"`
}

type PublicationApplication struct {
	PublicationPolicy
	ID        string `json:"id"`
	Revision  string `json:"revision"`
	Readiness string `json:"readiness"`
	Reason    string `json:"reason"`
}

type PublicationBinding struct {
	ID            string  `json:"id"`
	Revision      string  `json:"revision"`
	ApplicationID string  `json:"application_id"`
	EnvironmentID string  `json:"environment_id"`
	TargetID      *string `json:"target_id"`
	State         string  `json:"state"`
}

type PublicationTarget struct {
	PublicationApplication
	ApplicationID               *string `json:"application_id"`
	BindingID                   string  `json:"binding_id"`
	State                       string  `json:"state"`
	DesiredGeneration           string  `json:"desired_generation"`
	ObservedGeneration          *string `json:"observed_generation"`
	ConfigurationDigest         string  `json:"configuration_digest"`
	DesiredConfigurationDigest  *string `json:"desired_configuration_digest"`
	ObservedConfigurationDigest *string `json:"observed_configuration_digest"`
	ObservedBundleDigest        *string `json:"observed_bundle_digest"`
	BundleDigest                *string `json:"bundle_digest"`
	LastObservationMS           *string `json:"last_observation_ms"`
	Endpoint                    *string `json:"endpoint"`
	ExpiresAtMS                 *string `json:"expires_at_ms"`
}

type PublicationOperation struct {
	ID                  string  `json:"id"`
	Key                 string  `json:"operation_key"`
	Kind                string  `json:"kind"`
	PrincipalID         string  `json:"principal_id"`
	Fingerprint         string  `json:"fingerprint"`
	Status              string  `json:"status"`
	ResourceID          *string `json:"resource_id"`
	Location            string  `json:"location"`
	Generation          string  `json:"generation"`
	BundleDigest        *string `json:"bundle_digest"`
	ConfigurationDigest *string `json:"configuration_digest"`
	EffectStatus        string  `json:"effect_status"`
	ObservationRevision string  `json:"observation_revision"`
	LastObservationMS   *string `json:"last_observation_ms"`
	Readiness           string  `json:"readiness"`
	Reason              string  `json:"reason"`
	ExpiresAtMS         *string `json:"expires_at_ms"`
	// Only bounded history may contain an expired tombstone. Exact lookup
	// returns receipt_expired and must never permit key reuse.
	Code string `json:"code,omitempty"`
}

type PublicationCapabilities struct {
	Profile         string           `json:"profile"`
	Version         string           `json:"version"`
	Scope           string           `json:"scope"`
	PrincipalID     string           `json:"principal_id"`
	DeploymentClass string           `json:"deployment_class"`
	Capabilities    map[string]bool  `json:"capabilities"`
	Limits          map[string]int64 `json:"limits"`
	ReceiptPolicy   string           `json:"receipt_policy"`
}

type PublicationError struct {
	Code   string `json:"code"`
	Key    string `json:"operation_key,omitempty"`
	Lookup string `json:"lookup,omitempty"`
	Status int    `json:"-"`
}

func (e *PublicationError) Error() string { return "publication API: " + e.Code }

type PublicationOutcomeError struct{ Status string }

func (e *PublicationOutcomeError) Error() string { return "publication operation: " + e.Status }
