package platformclient

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"sync"
)

const BucketProfile = "spoke.platform.http.bucket.v1"
const MaxProfileCounter int64 = 1<<53 - 1

// ProfileIdentity pins a separately owned, bounded implementation of the wire
// inventory. Development metadata evidence cannot publish the full contract.
type ProfileIdentity struct {
	Name             string `json:"name"`
	Version          string `json:"version"`
	Status           string `json:"status"`
	PlatformRevision string `json:"platform_revision"`
	SchemaSHA256     string `json:"schema_sha256"`
	VectorsSHA256    string `json:"vectors_sha256"`
}

//go:embed profile.json
var profileJSON []byte

func CurrentProfile() ProfileIdentity {
	var p ProfileIdentity
	d := json.NewDecoder(bytes.NewReader(profileJSON))
	d.DisallowUnknownFields()
	if d.Decode(&p) != nil || d.Decode(new(any)) != io.EOF || p.Name != BucketProfile || p.Version != APIVersion || p.Status != "development-metadata" || !regexp.MustCompile(`^[a-f0-9]{40}$`).MatchString(p.PlatformRevision) || !fingerprintPattern.MatchString(p.SchemaSHA256) || !fingerprintPattern.MatchString(p.VectorsSHA256) {
		panic("invalid embedded Platform profile")
	}
	return p
}

// Shared across scoped client copies, so an endpoint cannot switch its declared
// profile during concurrent observations. Empty identifies the provisional mock.
type profileState struct {
	mu    sync.Mutex
	bound bool
	name  string
}

func (p *profileState) bind(name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if (name != "" && name != BucketProfile) || (p.bound && name != p.name) {
		return &APIError{Code: "incompatible_api"}
	}
	p.name, p.bound = name, true
	return nil
}

func (c *Client) bucketProfile() bool {
	c.profile.mu.Lock()
	defer c.profile.mu.Unlock()
	return c.profile.name == BucketProfile
}

var bucketProfileName = regexp.MustCompile(`^[a-z][a-z0-9-]{1,61}[a-z0-9]$`)

func validBucketProfileSpec(s *Spec) bool {
	return s != nil && bucketProfileName.MatchString(s.Name) && s.Valid("bucket")
}

func validBucketProfileIntent(i Intent) bool {
	if i.ResourceKind != "bucket" || i.ExpectedRevision < 0 || i.ExpectedRevision > MaxProfileCounter {
		return false
	}
	if i.Kind == "create" && (i.TargetID != "" || i.ExpectedRevision != 0) {
		return false
	}
	if i.Kind == "delete" {
		return i.Spec == nil
	}
	return validBucketProfileSpec(i.Spec)
}

// PendingPage is the bounded, complete current pending index, not receipt history.
type PendingPage struct {
	Scope
	ResourceKind string      `json:"resource_kind"`
	Items        []Operation `json:"items"`
	NextToken    string      `json:"next_token"`
	Complete     bool        `json:"complete"`
	Snapshot     int64       `json:"snapshot"`
}

func (p *PendingPage) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if json.Unmarshal(data, &fields) != nil {
		return errors.New("invalid pending page")
	}
	for _, key := range []string{"organization_id", "project_id", "environment_id", "resource_kind", "items", "next_token", "complete", "snapshot"} {
		v, ok := fields[key]
		if !ok || bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
			return errors.New("incomplete pending page")
		}
	}
	type plain PendingPage
	var value plain
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if d.Decode(&value) != nil {
		return errors.New("invalid pending page")
	}
	*p = PendingPage(value)
	return nil
}

// PendingBuckets observes a single checked cut without advancing accepted work.
// It is intentionally not exposed as a Terraform resource or terminal history.
func (c *Client) PendingBuckets(ctx context.Context) (PendingPage, error) {
	var page PendingPage
	caps, err := c.Capabilities(ctx)
	if err != nil {
		return page, err
	}
	if !c.bucketProfile() || !c.scope.AllowsKind("bucket") {
		return page, errors.New("pending discovery requires the Platform bucket profile")
	}
	if err = c.request(ctx, "GET", c.base()+"/management/bucket?state=accepted", nil, &page); err != nil {
		return PendingPage{}, err
	}
	if page.Scope != c.scope || page.ResourceKind != "bucket" || page.Items == nil || len(page.Items) > 32 || len(page.Items) > c.maxItems || !page.Complete || page.NextToken != "" || page.Snapshot < 1 {
		return PendingPage{}, errors.New("incomplete or invalid pending snapshot")
	}
	ids, keys := map[string]bool{}, map[string]bool{}
	for _, op := range page.Items {
		if !c.validOperation(op, op.ManagementKey) || !ValidID(op.ManagementKey) || op.ResourceKind != "bucket" || op.OwnerID != caps.PrincipalID || op.Status != "accepted" || op.ErrorCode != "" || ids[op.ID] || keys[op.ManagementKey] {
			return PendingPage{}, errors.New("invalid pending operation identity")
		}
		ids[op.ID], keys[op.ManagementKey] = true, true
	}
	return page, nil
}
