package liveplatform

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

const HostedEvidence = "hosted-platform-metadata-consumer"

// Separate from cases.json: these consumer observations cannot qualify services.
var HostedCases = []string{
	"direct_lifecycle_import_discovery_empty_plan",
	"direct_a_b_a_drift_revision_conflict",
	"direct_invalid_credential_preserves_state",
	"direct_protection_authoritative_absence",
	"relay_observational_reads",
	"create_before_forward", "create_after_acceptance", "create_after_success",
	"update_before_forward", "update_after_acceptance", "update_after_success",
	"delete_before_forward", "delete_after_acceptance", "delete_after_success",
	"credential_confinement",
}

var ConsumerLimits = []string{
	"platform_revision_operator_supplied_not_attested",
	"server_restart_receipt_expiry_and_backend_durability_not_qualified",
	"metadata_only_no_storage_or_service_release_qualification",
}

type ConsumerEvidence struct {
	Endpoint         string    `json:"endpoint"`
	Scope            api.Scope `json:"scope"`
	Principal        string    `json:"principal"`
	RunID            string    `json:"run_id"`
	StartedAt        string    `json:"started_at"`
	AttemptStartedAt string    `json:"attempt_started_at"`
	Resumed          bool      `json:"resumed"`
	Limitations      []string  `json:"limitations"`
}

func (c Config) validateHosted() error {
	p, contract := api.CurrentProfile(), api.CurrentContract()
	if c.EvidenceKind != HostedEvidence || c.Gate != "hosted-metadata" || c.ManagementProfile != p.Name || c.ProfileSchemaSHA256 != p.SchemaSHA256 || c.APIVersion != contract.Version || c.APISchemaSHA256 != contract.SchemaSHA256 {
		return errors.New("hosted consumer metadata requires the exact bounded bucket profile and HTTP contract")
	}
	u, err := url.Parse(c.Endpoint)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" || u.Opaque != "" {
		return errors.New("consumer endpoint must be an explicit HTTPS origin without a trailing slash")
	}
	if !c.Scope.Valid() || c.Scope.EnvironmentID == "" || !api.ValidID(c.ExpectedPrincipal) || !runID.MatchString(c.RunID) || c.DeadlineSeconds < 1 || c.DeadlineSeconds > 1800 || c.MaxObjects != 32 {
		return errors.New("consumer scope, principal, run ID, deadline and 32-object bound required")
	}
	if !sourceRevision.MatchString(c.ProviderRevision) || !sourceRevision.MatchString(c.HarnessRevision) || !sourceRevision.MatchString(c.PlatformRevision) || !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+-dev\.g`+c.ProviderRevision+`$`).MatchString(c.ProviderVersion) {
		return errors.New("exact immutable artifact, harness and operator-attributed Platform revisions required")
	}
	for _, path := range []string{c.CAFile, c.CredentialFile, c.RunDirectory, c.HarnessCheckout} {
		if !privatePath(path) || filepath.Clean(path) != path || path == "/" {
			return errors.New("explicit external CA, credential, harness and private run paths required")
		}
	}
	if strings.HasPrefix(c.RunDirectory+"/", c.HarnessCheckout+"/") || c.FixtureCheckout != "" {
		return errors.New("consumer runs must be outside the harness and cannot select a Platform fixture")
	}
	for _, path := range []string{c.CAFile, c.CredentialFile} {
		if strings.HasPrefix(path+"/", c.RunDirectory+"/") {
			return errors.New("external consumer credential and CA must be outside the run directory")
		}
	}
	want := filepath.Join(c.Mirror, "registry.terraform.io/spoke-sh/spoke", c.ProviderVersion, "linux_amd64", "terraform-provider-spoke_v"+c.ProviderVersion)
	if !filepath.IsAbs(c.Mirror) || c.Artifact != want || !hexDigest.MatchString(c.ArtifactSHA256) {
		return errors.New("exact installed mirror layout and artifact digest required")
	}
	return nil
}

func (r Report) validateHostedEvidence(c Config) error {
	if r.HarnessRevision != c.HarnessRevision || r.ProviderRevision != c.ProviderRevision || r.ProviderVersion != c.ProviderVersion || r.TerraformVersion != "1.15.3" || r.System != "linux_amd64" || r.ExecutionReport != filepath.Join(c.RunDirectory, "execution.json") || r.Consumer == nil {
		return errors.New("invalid hosted consumer artifact or harness identity")
	}
	u := r.Consumer
	started, e1 := time.Parse(time.RFC3339Nano, u.StartedAt)
	checked, e2 := time.Parse(time.RFC3339Nano, r.CheckedAt)
	attempt, e3 := time.Parse(time.RFC3339Nano, u.AttemptStartedAt)
	if e1 != nil || e2 != nil || e3 != nil || attempt.Before(started) || checked.Before(attempt) || (!u.Resumed && !attempt.Equal(started)) || checked.After(time.Now().Add(time.Minute)) || u.Endpoint != c.Endpoint || u.Scope != c.Scope || u.Principal != c.ExpectedPrincipal || u.RunID != c.RunID || !slices.Equal(u.Limitations, ConsumerLimits) {
		return errors.New("invalid consumer target, freshness or evidence limits")
	}
	return nil
}

// CheckHostedExecution is called only after a successful child process. The
// checkpoint must have been rewritten by this attempt, not copied from a prior run.
func CheckHostedExecution(data []byte, c Config, started time.Time) (Report, error) {
	var r Report
	if len(data) > 65536 || json.Unmarshal(data, &r) != nil {
		return r, errors.New("invalid consumer execution report")
	}
	checked, err := time.Parse(time.RFC3339Nano, r.CheckedAt)
	if err != nil || checked.Before(started) {
		return r, errors.New("stale consumer execution report")
	}
	if r.Consumer == nil {
		return r, errors.New("missing consumer execution")
	}
	attempt, err := time.Parse(time.RFC3339Nano, r.Consumer.AttemptStartedAt)
	if err != nil || attempt.Before(started) {
		return r, errors.New("stale consumer attempt")
	}
	return r, r.ValidateEvidence(c)
}

// ConsumerClient checks the public HTTPS boundary before any mutating test.
// It never reads ambient credentials and does not reinterpret HTML or S3 errors.
func ConsumerClient(ctx context.Context, c Config) (*api.Client, error) {
	read := func(path string, secret bool) ([]byte, error) {
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > 1<<20 || (secret && info.Mode().Perm()&0077 != 0) {
			return nil, errors.New("consumer credential or CA must be a bounded regular file; credentials require private permissions")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, errors.New("cannot read external consumer credential or CA")
		}
		return data, nil
	}
	ca, err := read(c.CAFile, false)
	if err != nil {
		return nil, err
	}
	token, err := read(c.CredentialFile, true)
	if err != nil {
		return nil, err
	}
	client, err := api.New(api.Config{Endpoint: c.Endpoint, Scope: c.Scope, Token: strings.TrimSpace(string(token)), CAPEM: ca, ManagementProfile: api.BucketProfile, MaxPages: 1, MaxItems: 32})
	if err != nil {
		return nil, errors.New("invalid consumer transport configuration")
	}
	caps, err := client.Capabilities(ctx)
	if err != nil || caps.PrincipalID != c.ExpectedPrincipal {
		return nil, errors.New("consumer TLS, profile, scope or authenticated principal preflight failed")
	}
	return client, nil
}
