// Package liveplatform owns explicit live qualification configuration and evidence.
// It never chooses a backend from ambient credentials or falls back to a mock.
package liveplatform

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spoke-sh/terraform-provider-spoke/internal/nativecontract"
)

type Config struct {
	Profile          string               `json:"profile"`
	Gate             string               `json:"gate"`
	Endpoint         string               `json:"endpoint"`
	CAFile           string               `json:"ca_file"`
	CredentialFile   string               `json:"credential_file"`
	JournalDirectory string               `json:"journal_directory"`
	Artifact         string               `json:"artifact"`
	ArtifactSHA256   string               `json:"artifact_sha256"`
	PlatformRevision string               `json:"platform_revision"`
	TransitRevision  string               `json:"transit_revision"`
	Scope            nativecontract.Scope `json:"scope"`
	RunID            string               `json:"run_id"`
	DeadlineSeconds  int                  `json:"deadline_seconds"`
	MaxObjects       int                  `json:"max_objects"`
}

var hexDigest = regexp.MustCompile(`^[0-9a-f]{64}$`)
var runID = regexp.MustCompile(`^[a-z][a-z0-9-]{2,63}$`)

func RequiredCases(gate string) []string {
	cases := []string{"lifecycle_import_empty_plan", "drift_revision_owner_denial", "protected_deletion", "provider_crash_exact_recovery", "management_restart"}
	if gate == "service-graph" {
		cases = append(cases, "oci_push_pull", "bucket_bytes_database_rows", "secret_workload_tls", "revoke_restore_rollout", "retirement_retains_data")
	}
	return cases
}
func Load(path string) (Config, error) {
	var c Config
	if !privatePath(path) {
		return c, errors.New("SPOKE_LIVE_CONFIG must select an absolute external configuration file")
	}
	f, err := os.Open(path)
	if err != nil {
		return c, errors.New("cannot read live configuration")
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 16385))
	if err != nil || len(data) > 16384 {
		return c, errors.New("invalid live configuration size")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&c) != nil || decoder.Decode(new(any)) != io.EOF {
		return c, errors.New("invalid live configuration schema")
	}
	return c, c.Validate()
}
func privatePath(path string) bool {
	return filepath.IsAbs(path) && !strings.HasPrefix(filepath.Clean(path)+"/", "/nix/store/")
}
func (c Config) Validate() error {
	if c.Profile != nativecontract.Profile || (c.Gate != "metadata" && c.Gate != "service-graph") {
		return errors.New("explicit native profile and metadata or service-graph gate required")
	}
	u, err := url.Parse(c.Endpoint)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return errors.New("live endpoint must be an explicit HTTPS origin")
	}
	if c.PlatformRevision != nativecontract.PlatformRevision || c.TransitRevision != nativecontract.TransitRevision {
		return errors.New("live dependency revisions are not the inspected contract pins")
	}
	if c.Scope.Organization == 0 || c.Scope.Project == 0 || c.Scope.Environment == 0 || !runID.MatchString(c.RunID) || c.DeadlineSeconds < 1 || c.DeadlineSeconds > 1800 || c.MaxObjects < 1 || c.MaxObjects > 64 {
		return errors.New("live scope, isolated run ID and bounded budgets are required")
	}
	if !hexDigest.MatchString(c.ArtifactSHA256) || !filepath.IsAbs(c.Artifact) {
		return errors.New("exact installed artifact and SHA-256 required")
	}
	for _, path := range []string{c.CAFile, c.CredentialFile, c.JournalDirectory} {
		if !privatePath(path) {
			return errors.New("live credential, CA and journal paths must be explicit and outside the Nix store")
		}
	}
	return nil
}

type Case struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}
type Report struct {
	Profile          string   `json:"profile"`
	Gate             string   `json:"gate"`
	ArtifactSHA256   string   `json:"artifact_sha256"`
	PlatformRevision string   `json:"platform_revision"`
	TransitRevision  string   `json:"transit_revision"`
	Status           string   `json:"status"`
	Cases            []Case   `json:"cases"`
	Blockers         []string `json:"blockers"`
}

// Preflight establishes artifact identity without reading credentials or sending
// requests. No current SDK API can dispatch Serving commands or sign these read
// proofs. Preserve that explicit blocker until the upstream boundary is usable.
func Preflight(c Config) (Report, error) {
	r := Report{Profile: c.Profile, Gate: c.Gate, ArtifactSHA256: c.ArtifactSHA256, PlatformRevision: c.PlatformRevision, TransitRevision: c.TransitRevision, Status: "blocked", Cases: []Case{}}
	if err := c.Validate(); err != nil {
		return r, err
	}
	f, err := os.Open(c.Artifact)
	if err != nil {
		return r, errors.New("cannot inspect installed artifact")
	}
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil || hex.EncodeToString(h.Sum(nil)) != c.ArtifactSHA256 {
		return r, errors.New("installed artifact digest mismatch")
	}
	for _, name := range RequiredCases(c.Gate) {
		r.Cases = append(r.Cases, Case{name, "blocked"})
	}
	r.Blockers = []string{"transit_go_sdk_missing_serving_and_subject_proofs", "native_adapter_and_durable_journal_not_qualified"}
	if c.Gate == "service-graph" {
		r.Blockers = append(r.Blockers, "platform_metadata_only_no_service_data_planes")
	}
	return r, errors.New("live qualification blocked: shared SDK Serving/proof support and installed native recovery are unavailable; see qualification/native-boundary.md")
}

// ValidateEvidence rejects empty, partial, duplicated, skipped or mismatched runs.
func (r Report) ValidateEvidence(c Config) error {
	if err := c.Validate(); err != nil {
		return err
	}
	if r.Status != "passed" || len(r.Blockers) != 0 || r.Profile != c.Profile || r.Gate != c.Gate || r.ArtifactSHA256 != c.ArtifactSHA256 || r.PlatformRevision != c.PlatformRevision || r.TransitRevision != c.TransitRevision {
		return errors.New("qualification identity or outcome mismatch")
	}
	required := map[string]bool{}
	for _, name := range RequiredCases(c.Gate) {
		required[name] = true
	}
	for _, test := range r.Cases {
		if !required[test.Name] || test.Status != "passed" {
			return errors.New("unexpected, duplicate or non-passing live case")
		}
		delete(required, test.Name)
	}
	if len(required) != 0 {
		return errors.New("missing required live cases")
	}
	return nil
}
