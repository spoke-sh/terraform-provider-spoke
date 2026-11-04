package platformclient

// These types mirror the pinned Platform schema. All fields are required on
// the wire; null is allowed only for the explicitly optional target/policy/ID.
type GovernorPolicy struct {
	Name                string   `json:"name"`
	AdmissionPolicyID   string   `json:"admission_policy_id"`
	BudgetPolicyID      string   `json:"budget_policy_id"`
	CachePolicyID       string   `json:"cache_policy_id"`
	ExecutionProfileIDs []string `json:"execution_profile_ids"`
	MaxActiveJobs       uint64   `json:"max_active_jobs"`
	MaxAllocatedVCPUs   uint64   `json:"max_allocated_vcpus"`
	MaxWarmRunners      uint64   `json:"max_warm_runners"`
	MaxJobSeconds       uint64   `json:"max_job_seconds"`
	MaxQueueSeconds     uint64   `json:"max_queue_seconds"`
	RetirementPolicy    string   `json:"retirement_policy"`
	DeletionProtection  bool     `json:"deletion_protection"`
}
type GovernorIntent struct {
	Kind             string          `json:"kind"`
	TargetID         *string         `json:"target_id"`
	ExpectedRevision uint64          `json:"expected_revision"`
	Policy           *GovernorPolicy `json:"policy"`
}
type GovernorRequest struct {
	ManagementKey string         `json:"management_key"`
	Sequence      uint64         `json:"sequence"`
	Intent        GovernorIntent `json:"intent"`
}
type GovernorOperation struct {
	Scope         string         `json:"scope"`
	PrincipalID   string         `json:"principal_id"`
	ID            string         `json:"id"`
	ManagementKey string         `json:"management_key"`
	Sequence      uint64         `json:"sequence"`
	Fingerprint   string         `json:"fingerprint"`
	Intent        GovernorIntent `json:"intent"`
	Status        string         `json:"status"`
	ResourceID    *string        `json:"resource_id"`
	Reason        string         `json:"reason"`
}

func (o GovernorOperation) Request() GovernorRequest {
	return GovernorRequest{o.ManagementKey, o.Sequence, o.Intent}
}

type Governor struct {
	Scope          string         `json:"scope"`
	PrincipalID    string         `json:"principal_id"`
	ID             string         `json:"id"`
	ManagementKey  string         `json:"management_key"`
	Revision       uint64         `json:"revision"`
	ActiveRevision uint64         `json:"active_revision"`
	Policy         GovernorPolicy `json:"policy"`
	State          string         `json:"state"`
	Blockers       []string       `json:"blockers"`
	ActiveJobs     uint64         `json:"active_jobs"`
	AllocatedVCPUs uint64         `json:"allocated_vcpus"`
	UnsettledUsage uint64         `json:"unsettled_usage"`
}
type GovernorCapabilities struct {
	Scope        string            `json:"scope"`
	PrincipalID  string            `json:"principal_id"`
	Version      string            `json:"version"`
	Profile      string            `json:"profile"`
	Status       string            `json:"status"`
	Capabilities map[string]bool   `json:"capabilities"`
	Limits       map[string]uint64 `json:"limits"`
}
type GovernorPage struct {
	Scope       string     `json:"scope"`
	PrincipalID string     `json:"principal_id"`
	Snapshot    uint64     `json:"snapshot"`
	Complete    bool       `json:"complete"`
	NextCursor  string     `json:"next_cursor"`
	Items       []Governor `json:"items"`
}
type GovernorPendingPage struct {
	Scope       string              `json:"scope"`
	PrincipalID string              `json:"principal_id"`
	Snapshot    uint64              `json:"snapshot"`
	Complete    bool                `json:"complete"`
	NextCursor  string              `json:"next_cursor"`
	Items       []GovernorOperation `json:"items"`
}
type GovernorError struct {
	Code   string `json:"code"`
	Status int    `json:"-"`
}

func (e *GovernorError) Error() string { return "governor HTTP request failed: " + e.Code }

type GovernorOutcomeError struct{ Status, Reason string }

func (e *GovernorOutcomeError) Error() string {
	return "governor operation " + e.Status + ": " + e.Reason
}
