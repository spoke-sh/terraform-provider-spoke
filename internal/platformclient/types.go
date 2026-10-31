// Package platformclient implements the provisional mock-v1 management contract.
// It is not a client for an existing Spoke Platform production API.
package platformclient

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
)

const APIVersion = "mock-v1"

var identifier = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,127}$`)

func ValidID(id string) bool { return identifier.MatchString(id) }

type Scope struct {
	OrganizationID string `json:"organization_id"`
	ProjectID      string `json:"project_id"`
	EnvironmentID  string `json:"environment_id"`
}

func (s Scope) Valid() bool {
	return ValidID(s.OrganizationID) && ValidID(s.ProjectID) && ValidID(s.EnvironmentID)
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

type BucketSpec struct {
	Name               string `json:"name"`
	DeletionProtection bool   `json:"deletion_protection"`
}

type Bucket struct {
	Scope
	BucketSpec
	ID            string `json:"id"`
	ManagementKey string `json:"management_key"`
	Revision      int64  `json:"revision"`
}

// Intent is independent of the current operation generation. A revision fence
// distinguishes returning to an earlier configuration from replaying old work.
type Intent struct {
	Kind             string      `json:"kind"`
	ManagementKey    string      `json:"management_key"`
	TargetID         string      `json:"target_id"`
	ExpectedRevision int64       `json:"expected_revision"`
	Spec             *BucketSpec `json:"spec,omitempty"`
}

func (i Intent) Fingerprint() string {
	b, _ := json.Marshal(i)
	h := sha256.Sum256(b)
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
}

type Management struct {
	Scope
	ManagementKey string     `json:"management_key"`
	Generation    int64      `json:"generation"`
	ResourceID    string     `json:"resource_id"`
	Operation     *Operation `json:"operation"`
}

type APIError struct {
	Code string `json:"code"`
	// Never print the server's message: it may contain credentials or payloads.
	Status int `json:"-"`
}

func (e *APIError) Error() string { return fmt.Sprintf("platform API: %s (HTTP %d)", e.Code, e.Status) }

func IsCode(err error, code string) bool {
	var e *APIError
	return errors.As(err, &e) && e.Code == code
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
