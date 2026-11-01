package liveplatform

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

func validConfig(t *testing.T) Config {
	t.Helper()
	dir := t.TempDir()
	artifact := filepath.Join(dir, "provider")
	if err := os.WriteFile(artifact, []byte("synthetic artifact"), 0700); err != nil {
		t.Fatal(err)
	}
	contract := platformclient.CurrentContract()
	return Config{APIVersion: contract.Version, APISchemaSHA256: contract.SchemaSHA256, Gate: "service-graph", Endpoint: "https://isolated.example.invalid", CAFile: filepath.Join(dir, "ca.pem"), CredentialFile: filepath.Join(dir, "credential"), Artifact: artifact, ArtifactSHA256: fmt.Sprintf("%x", sha256.Sum256([]byte("synthetic artifact"))), PlatformRevision: strings.Repeat("a", 40), Scope: platformclient.Scope{OrganizationID: "org", ProjectID: "project", EnvironmentID: "env"}, RunID: "isolated-test", DeadlineSeconds: 60, MaxObjects: 16}
}

func passedReport(c Config) Report {
	r := Report{Transport: "https-json", APIVersion: c.APIVersion, APISchemaSHA256: c.APISchemaSHA256, Gate: c.Gate, ArtifactSHA256: c.ArtifactSHA256, PlatformRevision: c.PlatformRevision, Status: "passed", Blockers: []string{}}
	for _, name := range RequiredCases(c.Gate) {
		r.Cases = append(r.Cases, Case{Name: name, Status: "passed"})
	}
	return r
}

func TestPreflightReportsHTTPPrerequisitesWithoutCredentials(t *testing.T) {
	c := validConfig(t)
	// Neither of these files exists. A prerequisite check must not read them or
	// contact the deliberately unavailable endpoint.
	r, err := Preflight(c)
	if err == nil || r.Status != "blocked" || len(r.Cases) != 16 || len(r.Blockers) != 3 {
		t.Fatalf("unavailable HTTP gate passed: %+v %v", r, err)
	}
	if r.Blockers[0] != "platform_http_management_contract_not_published" {
		t.Fatal("wrong upstream prerequisite", r.Blockers)
	}
	data, _ := json.Marshal(r)
	for _, obsolete := range []string{"transit", "native", "journal", "serving"} {
		if strings.Contains(strings.ToLower(string(data)), obsolete) {
			t.Fatalf("report retained obsolete %s prerequisite", obsolete)
		}
	}
	if r.ValidateEvidence(c) == nil || passedReport(c).ValidateEvidence(c) == nil {
		t.Fatal("blocked or fabricated passing mock evidence qualified live services")
	}
	c.ArtifactSHA256 = strings.Repeat("0", 64)
	if _, err = Preflight(c); err == nil || err.Error() != "installed artifact digest mismatch" {
		t.Fatal("different artifact accepted", err)
	}
}

func TestExplicitHTTPConfigurationAndHardCutover(t *testing.T) {
	if _, err := Load(""); err == nil {
		t.Fatal("ambient config accepted")
	}
	for _, change := range []func(*Config){
		func(c *Config) { c.Endpoint = "http://localhost" },
		func(c *Config) { c.Endpoint = "https://secret@localhost" },
		func(c *Config) { c.APIVersion = "spoke.management.native.v2" },
		func(c *Config) { c.APISchemaSHA256 = strings.Repeat("0", 64) },
		func(c *Config) { c.CredentialFile = "/nix/store/credential" },
		func(c *Config) { c.DeadlineSeconds = 0 },
		func(c *Config) { c.Scope.ProjectID = "" },
		func(c *Config) { c.PlatformRevision = "floating" },
		func(c *Config) { c.Gate = "unknown" },
	} {
		c := validConfig(t)
		change(&c)
		if c.Validate() == nil {
			t.Fatal("invalid live config accepted")
		}
	}
	c := validConfig(t)
	data, _ := json.Marshal(c)
	file := filepath.Join(t.TempDir(), "live.json")
	if err := os.WriteFile(file, data, 0600); err != nil {
		t.Fatal(err)
	}
	if got, err := Load(file); err != nil || got != c {
		t.Fatalf("HTTP configuration failed: %+v %v", got, err)
	}
	for _, obsolete := range []string{"profile", "transit_revision", "journal_directory"} {
		var legacy map[string]any
		_ = json.Unmarshal(data, &legacy)
		legacy[obsolete] = "old-native-setting"
		b, _ := json.Marshal(legacy)
		if err := os.WriteFile(file, b, 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(file); err == nil {
			t.Fatalf("accepted superseded configuration field %s", obsolete)
		}
	}
}

func TestHTTPReportRequiresCompleteBoundEvidence(t *testing.T) {
	c := validConfig(t)
	if err := passedReport(c).validateCases(c); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*Report){
		func(r *Report) { r.Transport = "native" },
		func(r *Report) { r.APIVersion = "other" },
		func(r *Report) { r.APISchemaSHA256 = strings.Repeat("b", 64) },
		func(r *Report) { r.PlatformRevision = strings.Repeat("b", 40) },
		func(r *Report) { r.ArtifactSHA256 = strings.Repeat("b", 64) },
		func(r *Report) { r.Blockers = []string{"missing_backend"} },
		func(r *Report) { r.Status = "blocked" },
		func(r *Report) { r.Cases[0].Status = "skipped" },
		func(r *Report) { r.Cases[1] = r.Cases[0] },
		func(r *Report) { r.Cases = r.Cases[:10] },
		func(r *Report) { r.Cases = nil },
	} {
		r := passedReport(c)
		change(&r)
		if r.validateCases(c) == nil {
			t.Fatal("invalid HTTP evidence accepted", r)
		}
	}
	for _, family := range []string{"metadata", "initial-graph", "jobs", "messaging", "volumes", "networking", "recovery", "policies"} {
		partial := c
		partial.Gate = family
		r := passedReport(partial)
		if r.validateCases(partial) != nil || r.validateCases(c) == nil {
			t.Fatalf("family %s substituted for the full advertised graph", family)
		}
	}
	if len(RequiredCases("unknown")) != 0 {
		t.Fatal("unknown gate silently selected metadata cases")
	}
}
