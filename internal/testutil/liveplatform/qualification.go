// Package liveplatform owns explicit live qualification configuration and evidence.
// It never chooses a backend from ambient credentials or falls back to a mock.
package liveplatform

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

type Config struct {
	APIVersion       string               `json:"api_version"`
	APISchemaSHA256  string               `json:"api_schema_sha256"`
	Gate             string               `json:"gate"`
	Endpoint         string               `json:"endpoint"`
	CAFile           string               `json:"ca_file"`
	CredentialFile   string               `json:"credential_file"`
	Artifact         string               `json:"artifact"`
	ArtifactSHA256   string               `json:"artifact_sha256"`
	PlatformRevision string               `json:"platform_revision"`
	Scope            platformclient.Scope `json:"scope"`
	RunID            string               `json:"run_id"`
	DeadlineSeconds  int                  `json:"deadline_seconds"`
	MaxObjects       int                  `json:"max_objects"`
}

var hexDigest = regexp.MustCompile(`^[0-9a-f]{64}$`)
var sourceRevision = regexp.MustCompile(`^[0-9a-f]{40}$`)
var runID = regexp.MustCompile(`^[a-z][a-z0-9-]{2,63}$`)

//go:embed cases.json
var caseManifest []byte

func RequiredCases(gate string) []string {
	var manifest map[string][]string
	if err := json.Unmarshal(caseManifest, &manifest); err != nil {
		panic(err)
	}
	if _, ok := manifest[gate]; !ok && gate != "service-graph" {
		return nil
	}
	cases := slices.Clone(manifest["metadata"])
	families := []string{"initial-graph", "jobs", "messaging", "volumes", "networking", "recovery", "policies"}
	for _, family := range families {
		if gate == "service-graph" || gate == family {
			cases = append(cases, manifest[family]...)
		}
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
	contract := platformclient.CurrentContract()
	if c.APIVersion != contract.Version || c.APISchemaSHA256 != contract.SchemaSHA256 || len(RequiredCases(c.Gate)) == 0 {
		return errors.New("explicit supported HTTP contract, schema digest and qualification gate required")
	}
	u, err := url.Parse(c.Endpoint)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return errors.New("live endpoint must be an explicit HTTPS origin")
	}
	if !sourceRevision.MatchString(c.PlatformRevision) || (contract.Status == "published" && c.PlatformRevision != contract.PlatformRevision) {
		return errors.New("explicit Platform revision matching the published HTTP contract required")
	}
	if !c.Scope.Valid() || c.Scope.EnvironmentID == "" || !runID.MatchString(c.RunID) || c.DeadlineSeconds < 1 || c.DeadlineSeconds > 1800 || c.MaxObjects < 1 || c.MaxObjects > 64 {
		return errors.New("live scope, isolated run ID and bounded budgets are required")
	}
	if !hexDigest.MatchString(c.ArtifactSHA256) || !filepath.IsAbs(c.Artifact) {
		return errors.New("exact installed artifact and SHA-256 required")
	}
	for _, path := range []string{c.CAFile, c.CredentialFile} {
		if !privatePath(path) {
			return errors.New("live credential and CA paths must be explicit and outside the Nix store")
		}
	}
	return nil
}

type Case struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}
type Report struct {
	Transport        string   `json:"transport"`
	APIVersion       string   `json:"api_version"`
	APISchemaSHA256  string   `json:"api_schema_sha256"`
	Gate             string   `json:"gate"`
	ArtifactSHA256   string   `json:"artifact_sha256"`
	PlatformRevision string   `json:"platform_revision"`
	Status           string   `json:"status"`
	Cases            []Case   `json:"cases"`
	Blockers         []string `json:"blockers"`
}

// Preflight verifies the explicit HTTP contract and artifact without reading
// credentials or contacting services. The only implemented wire schema is still
// provisional; neither a configured endpoint nor a fabricated passing report
// supplies the missing Platform-owned contract or installed live cases.
func Preflight(c Config) (Report, error) {
	contract := platformclient.CurrentContract()
	r := Report{Transport: contract.Transport, APIVersion: c.APIVersion, APISchemaSHA256: c.APISchemaSHA256, Gate: c.Gate, ArtifactSHA256: c.ArtifactSHA256, PlatformRevision: c.PlatformRevision, Status: "blocked", Cases: []Case{}}
	if err := c.Validate(); err != nil {
		return r, err
	}
	f, err := os.Open(c.Artifact)
	if err != nil {
		return r, errors.New("cannot inspect installed artifact")
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 || info.Size() < 1 || info.Size() > 128<<20 {
		return r, errors.New("installed artifact must be a regular executable no larger than 128 MiB")
	}
	h := sha256.New()
	if _, err = io.Copy(h, io.LimitReader(f, (128<<20)+1)); err != nil || hex.EncodeToString(h.Sum(nil)) != c.ArtifactSHA256 {
		return r, errors.New("installed artifact digest mismatch")
	}
	for _, name := range RequiredCases(c.Gate) {
		r.Cases = append(r.Cases, Case{name, "blocked"})
	}
	if contract.Status != "published" {
		r.Blockers = append(r.Blockers, "platform_http_management_contract_not_published")
	}
	r.Blockers = append(r.Blockers, "installed_http_live_cases_not_implemented")
	if c.Gate != "metadata" {
		r.Blockers = append(r.Blockers, "platform_service_backends_not_qualified")
	}
	return r, errors.New("live qualification blocked: Platform HTTP contract and installed real-service evidence are required; see docs/roadmap/milestones-10-17-implementation-plan.md")
}

// ValidateEvidence rejects empty, partial, duplicated, skipped or mismatched runs.
func (r Report) ValidateEvidence(c Config) error {
	if err := c.Validate(); err != nil {
		return err
	}
	contract := platformclient.CurrentContract()
	if contract.Status != "published" {
		return errors.New("provisional HTTP contract cannot qualify live services")
	}
	return r.validateCases(c)
}

// Keep structural validation separate from the published-contract prerequisite;
// passing this check alone never grants live qualification.
func (r Report) validateCases(c Config) error {
	if r.Status != "passed" || len(r.Blockers) != 0 || r.Transport != "https-json" || r.APIVersion != c.APIVersion || r.APISchemaSHA256 != c.APISchemaSHA256 || r.Gate != c.Gate || r.ArtifactSHA256 != c.ArtifactSHA256 || r.PlatformRevision != c.PlatformRevision {
		return errors.New("qualification identity or outcome mismatch")
	}
	if len(RequiredCases(c.Gate)) == 0 {
		return errors.New("unknown qualification gate")
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
