package liveplatform

import (
	"crypto/sha256"
	"fmt"
	"github.com/spoke-sh/terraform-provider-spoke/internal/nativecontract"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func validConfig(t *testing.T) Config {
	t.Helper()
	dir := t.TempDir()
	artifact := filepath.Join(dir, "provider")
	if err := os.WriteFile(artifact, []byte("synthetic artifact"), 0700); err != nil {
		t.Fatal(err)
	}
	return Config{Profile: nativecontract.Profile, Gate: "service-graph", Endpoint: "https://isolated.example.invalid", CAFile: filepath.Join(dir, "ca.pem"), CredentialFile: filepath.Join(dir, "seed"), JournalDirectory: filepath.Join(dir, "journal"), Artifact: artifact, ArtifactSHA256: fmt.Sprintf("%x", sha256.Sum256([]byte("synthetic artifact"))), PlatformRevision: nativecontract.PlatformRevision, TransitRevision: nativecontract.TransitRevision, Scope: nativecontract.Scope{Organization: 1, Project: 2, Environment: 3}, RunID: "isolated-test", DeadlineSeconds: 60, MaxObjects: 16}
}
func TestPreflightCannotQualifyMockOrPartialRun(t *testing.T) {
	c := validConfig(t)
	r, err := Preflight(c)
	if err == nil || r.Status != "blocked" || len(r.Cases) != 16 || len(r.Blockers) != 4 {
		t.Fatal("unavailable native gate passed")
	}
	if err = r.ValidateEvidence(c); err == nil {
		t.Fatal("blocked evidence accepted")
	}
	r.Status = "passed"
	r.Blockers = nil
	for i := range r.Cases {
		r.Cases[i].Status = "passed"
	}
	if err = r.ValidateEvidence(c); err != nil {
		t.Fatal(err)
	}
	r.Cases[0].Status = "skipped"
	if r.ValidateEvidence(c) == nil {
		t.Fatal("skipped gate accepted")
	}
	r.Cases[0].Status = "passed"
	r.Cases = r.Cases[1:]
	if r.ValidateEvidence(c) == nil {
		t.Fatal("missing case accepted")
	}
	r.Cases = nil
	if r.ValidateEvidence(c) == nil {
		t.Fatal("empty run accepted")
	}
	c.ArtifactSHA256 = strings.Repeat("0", 64)
	if _, err = Preflight(c); err == nil {
		t.Fatal("different artifact accepted")
	}
}
func TestExplicitConfiguration(t *testing.T) {
	if _, err := Load(""); err == nil {
		t.Fatal("ambient config accepted")
	}
	for _, change := range []func(*Config){func(c *Config) { c.Endpoint = "http://localhost" }, func(c *Config) { c.Endpoint = "https://secret@localhost" }, func(c *Config) { c.Profile = "mock-v5" }, func(c *Config) { c.CredentialFile = "/nix/store/seed" }, func(c *Config) { c.DeadlineSeconds = 0 }, func(c *Config) { c.Scope.Project = 0 }, func(c *Config) { c.TransitRevision = "floating" }} {
		c := validConfig(t)
		change(&c)
		if c.Validate() == nil {
			t.Fatal("invalid live config accepted")
		}
	}
}

func TestFamilyGatesCannotSubstituteForFullGraph(t *testing.T) {
	c := validConfig(t)
	for _, family := range []string{"jobs", "messaging", "volumes", "networking", "recovery", "policies"} {
		c.Gate = family
		r, err := Preflight(c)
		if err == nil || len(r.Cases) != 6 {
			t.Fatal("missing family blockers", family)
		}
		r.Status = "passed"
		r.Blockers = nil
		for i := range r.Cases {
			r.Cases[i].Status = "passed"
		}
		if r.ValidateEvidence(c) != nil {
			t.Fatal("family identity")
		}
		c.Gate = "service-graph"
		if r.ValidateEvidence(c) == nil {
			t.Fatal("partial family qualified full graph")
		}
	}
}
