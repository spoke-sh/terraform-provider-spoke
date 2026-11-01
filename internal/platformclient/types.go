// Package platformclient implements the provisional mock-v5 management contract.
// It is not a client for an existing Spoke Platform production API.
package platformclient

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
)

const APIVersion = "mock-v5"

var identifier = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,127}$`)

func ValidID(id string) bool { return identifier.MatchString(id) }

type Scope struct {
	OrganizationID string `json:"organization_id"`
	ProjectID      string `json:"project_id"`
	EnvironmentID  string `json:"environment_id"`
}

func (s Scope) Valid() bool {
	return ValidID(s.OrganizationID) && (s.ProjectID == "" || ValidID(s.ProjectID)) && (s.EnvironmentID == "" || (ValidID(s.EnvironmentID) && s.ProjectID != ""))
}

func (s Scope) Path() string {
	p := "/v5/organizations/" + s.OrganizationID
	if s.ProjectID != "" {
		p += "/projects/" + s.ProjectID
	}
	if s.EnvironmentID != "" {
		p += "/environments/" + s.EnvironmentID
	}
	return p
}

type Project struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
}

type Environment struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	ProjectID      string `json:"project_id"`
	Name           string `json:"name"`
}

type Spec struct {
	Name               string          `json:"name"`
	DeletionProtection bool            `json:"deletion_protection"`
	SubjectID          string          `json:"subject_id,omitempty"`
	Target             string          `json:"target,omitempty"`
	Permission         string          `json:"permission,omitempty"`
	Registry           *RegistrySpec   `json:"registry,omitempty"`
	Storage            *StorageSpec    `json:"storage,omitempty"`
	Secret             *SecretSpec     `json:"secret,omitempty"`
	Binding            *BindingSpec    `json:"binding,omitempty"`
	Container          *ContainerSpec  `json:"container,omitempty"`
	Endpoint           *EndpointSpec   `json:"endpoint,omitempty"`
	Job                *JobSpec        `json:"job,omitempty"`
	Schedule           *ScheduleSpec   `json:"schedule,omitempty"`
	Stream             *StreamSpec     `json:"stream,omitempty"`
	Queue              *QueueSpec      `json:"queue,omitempty"`
	Volume             *VolumeSpec     `json:"volume,omitempty"`
	Attachment         *AttachmentSpec `json:"attachment,omitempty"`
	Domain             *DomainSpec     `json:"domain,omitempty"`
	Connection         *ConnectionSpec `json:"connection,omitempty"`
	Backup             *BackupSpec     `json:"backup,omitempty"`
	Clone              *CloneSpec      `json:"clone,omitempty"`
	Scaling            *ScalingSpec    `json:"scaling,omitempty"`
	Placement          *PlacementSpec  `json:"placement,omitempty"`
}

type Bucket struct {
	Scope
	Spec
	ID            string `json:"id"`
	ManagementKey string `json:"management_key"`
	Revision      int64  `json:"revision"`
	OwnerID       string `json:"owner_id"`
}

// Object is the common management envelope; each kind has separate spec rules.
type Object struct {
	Bucket
	ResourceKind string         `json:"resource_kind"`
	Status       *ServiceStatus `json:"status,omitempty"`
}

// Missing or null protection must never be decoded as permission to delete.
func (o *Object) UnmarshalJSON(data []byte) error {
	if err := validateSettingsWire(data); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return errors.New("invalid object envelope")
	}
	for _, key := range []string{"organization_id", "project_id", "environment_id", "id", "resource_kind", "owner_id", "management_key", "revision", "name", "deletion_protection"} {
		value, ok := fields[key]
		if !ok || bytes.Equal(value, []byte("null")) {
			return errors.New("incomplete object envelope")
		}
	}
	type plain Object
	var value plain
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&value); err != nil {
		return errors.New("invalid object envelope")
	}
	*o = Object(value)
	return nil
}

func Collection(kind string) string {
	if collection, ok := LifecycleCollections[kind]; ok {
		return collection
	}
	if collection, ok := ServiceCollections[kind]; ok {
		return collection
	}
	switch kind {
	case "bucket":
		return "buckets"
	case "project":
		return "projects"
	case "environment":
		return "environments"
	case "service_account":
		return "service_accounts"
	case "grant":
		return "grants"
	}
	return ""
}

func (s Scope) AllowsKind(kind string) bool {
	if ServiceKind(kind) {
		return s.Valid() && s.EnvironmentID != ""
	}
	switch kind {
	case "project", "service_account", "grant":
		return s.Valid() && s.ProjectID == ""
	case "environment":
		return s.Valid() && s.ProjectID != "" && s.EnvironmentID == ""
	case "bucket":
		return s.Valid() && s.EnvironmentID != ""
	}
	return false
}

type Capabilities struct {
	Scope
	Version       string   `json:"version"`
	PrincipalID   string   `json:"principal_id"`
	Resources     []string `json:"resources"`
	ReceiptPolicy string   `json:"receipt_policy"`
}

type Page struct {
	Scope
	ResourceKind string   `json:"resource_kind"`
	Items        []Object `json:"items"`
	NextToken    string   `json:"next_token"`
	Complete     bool     `json:"complete"`
	Snapshot     int64    `json:"snapshot"`
}

// Intent is independent of the current operation generation. A revision fence
// distinguishes returning to an earlier configuration from replaying old work.
type Intent struct {
	Kind             string `json:"kind"`
	ManagementKey    string `json:"management_key"`
	TargetID         string `json:"target_id"`
	ExpectedRevision int64  `json:"expected_revision"`
	Spec             *Spec  `json:"spec,omitempty"`
	ResourceKind     string `json:"resource_kind"`
}

func (i Intent) Fingerprint() string {
	// Required binding slots form a set. Canonicalize a copy, never caller-owned input.
	if i.Spec != nil && i.Spec.Job != nil {
		spec, job := *i.Spec, *i.Spec.Job
		job.Workload.RequiredBindings = slices.Clone(job.Workload.RequiredBindings)
		slices.Sort(job.Workload.RequiredBindings)
		spec.Job = &job
		i.Spec = &spec
	}
	if i.Spec != nil && i.Spec.Container != nil {
		spec, container := *i.Spec, *i.Spec.Container
		container.RequiredBindings = slices.Clone(container.RequiredBindings)
		slices.Sort(container.RequiredBindings)
		spec.Container = &container
		i.Spec = &spec
	}
	b, _ := json.Marshal(i)
	h := sha256.Sum256(append([]byte(APIVersion+"\n"), b...))
	return hex.EncodeToString(h[:])
}

type Mutation struct {
	Intent
	ExpectedGeneration int64 `json:"expected_generation"`
}

type Operation struct {
	Scope
	ID            string `json:"id"`
	ManagementKey string `json:"management_key"`
	Fingerprint   string `json:"fingerprint"`
	Kind          string `json:"kind"`
	TargetID      string `json:"target_id"`
	Generation    int64  `json:"generation"`
	Status        string `json:"status"`
	ErrorCode     string `json:"error_code,omitempty"`
	ResourceKind  string `json:"resource_kind"`
	OwnerID       string `json:"owner_id"`
}

type Management struct {
	Scope
	ManagementKey string     `json:"management_key"`
	Generation    int64      `json:"generation"`
	ResourceID    string     `json:"resource_id"`
	Operation     *Operation `json:"operation"`
	OwnerID       string     `json:"owner_id"`
	ResourceKind  string     `json:"resource_kind"`
}

type APIError struct {
	Code string `json:"code"`
	// Never print the server's message: it may contain credentials or payloads.
	Status int `json:"-"`
}

func (e *APIError) Error() string { return fmt.Sprintf("platform API: %s (HTTP %d)", e.Code, e.Status) }

func IsCode(err error, code string) bool {
	var e *APIError
	if errors.As(err, &e) {
		return e.Code == code
	}
	var operation *OperationError
	return errors.As(err, &operation) && operation.Code == code
}

// OperationError is a terminal service failure observed over a successful read.
type OperationError struct {
	Operation Operation
	Code      string
}

func (e *OperationError) Error() string {
	return fmt.Sprintf("operation %s failed: %s", e.Operation.ID, e.Code)
}

// PendingError preserves a known operation when waiting fails. Timeout is not
// rollback; another process can recover it through the same management key.
type PendingError struct {
	Operation Operation
	Cause     error
}

func (e *PendingError) Error() string {
	return fmt.Sprintf("operation %s for management key %s remains recoverable: %v", e.Operation.ID, e.Operation.ManagementKey, e.Cause)
}
func (e *PendingError) Unwrap() error { return e.Cause }
