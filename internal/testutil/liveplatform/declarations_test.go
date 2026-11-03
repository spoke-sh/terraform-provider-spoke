package liveplatform

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

func declarationConfig(t *testing.T) Config {
	c := localConfig(t)
	p := platformclient.CurrentPublicationContract()
	c.EvidenceKind, c.Gate = declarationEvidence, "publication-declarations"
	c.APIVersion, c.APISchemaSHA256, c.ManagementProfile, c.ProfileSchemaSHA256 = p.Version, p.SchemaSHA256, p.Name, p.SchemaSHA256
	c.MaxObjects = 64
	c.HarnessCheckout, c.HarnessRevision, c.Mirror = t.TempDir(), strings.Repeat("c", 40), t.TempDir()
	data, err := os.ReadFile(c.Artifact)
	if err != nil {
		t.Fatal(err)
	}
	c.Artifact = filepath.Join(c.Mirror, "registry.terraform.io/spoke-sh/spoke", c.ProviderVersion, "linux_amd64", "terraform-provider-spoke_v"+c.ProviderVersion)
	if err = os.MkdirAll(filepath.Dir(c.Artifact), 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(c.Artifact, data, 0700); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestDeclarationSelectionIsExplicitAndCannotQualifyServices(t *testing.T) {
	c := declarationConfig(t)
	r, err := Preflight(c)
	if err != nil {
		t.Fatal(err)
	}
	if r.ValidateEvidence(c) == nil {
		t.Fatal("preflight qualified declarations")
	}
	for _, change := range []func(*Config){
		func(c *Config) { c.Gate = "service-graph" },
		func(c *Config) { c.EvidenceKind = "local-platform-metadata" },
		func(c *Config) { c.ManagementProfile = "spoke.platform.http.publication.peer.v1" },
		func(c *Config) { c.APISchemaSHA256 = strings.Repeat("0", 64) },
		func(c *Config) { c.APIVersion = "mock-v5" },
		func(c *Config) { c.HarnessRevision = "main" },
		func(c *Config) { c.HarnessCheckout = "relative" },
		func(c *Config) { c.Artifact += "-other" },
		func(c *Config) { c.CredentialFile = "/private/token" },
		func(c *Config) { c.Endpoint = "https://live.example.invalid" },
		func(c *Config) { c.MaxObjects = 65 },
		func(c *Config) { c.DeadlineSeconds = 1801 },
	} {
		bad := c
		change(&bad)
		if bad.Validate() == nil {
			t.Fatal("unsupported declaration configuration accepted")
		}
	}
	if _, err := RunDeclarations(c, r); err == nil {
		t.Fatal("uncommitted fixture accepted")
	}
	r = passedReport(c)
	r.EvidenceKind, r.ManagementProfile, r.ProfileSchemaSHA256 = c.EvidenceKind, c.ManagementProfile, c.ProfileSchemaSHA256
	r.ProviderRevision, r.ProviderVersion, r.HarnessRevision = c.ProviderRevision, c.ProviderVersion, c.HarnessRevision
	r.TerraformVersion, r.System, r.CheckedAt, r.ExecutionReport = "1.15.3", "linux_amd64", time.Now().Format(time.RFC3339Nano), filepath.Join(t.TempDir(), "execution.json")
	if err := r.ValidateEvidence(c); err != nil {
		t.Fatal(err)
	}
	for _, gate := range []string{"metadata", "static-publication", "service-graph"} {
		full := validConfig(t)
		full.Gate = gate
		if r.ValidateEvidence(full) == nil || r.validateCases(full) == nil {
			t.Fatal("local declarations qualified services")
		}
	}
	r.Cases[0].Status = "skipped"
	if r.ValidateEvidence(c) == nil {
		t.Fatal("skipped case qualified")
	}
}

func TestDeclarationReportBindsArtifactHarnessProfileAndAllCases(t *testing.T) {
	c := declarationConfig(t)
	start := time.Now().Add(-time.Minute)
	u := declarationReport{EvidenceKind: c.EvidenceKind, Result: "passed", ProviderRevision: c.ProviderRevision, HarnessRevision: c.HarnessRevision, PlatformRevision: c.PlatformRevision, ProviderArtifact: installedArtifact{Revision: c.ProviderRevision, Version: c.ProviderVersion, SHA256: c.ArtifactSHA256}, TerraformVersion: "1.15.3", System: "linux_amd64", CheckedAt: time.Now(), Cases: RequiredCases(c.Gate)}
	u.Contract.Name, u.Contract.Version, u.Contract.Status, u.Contract.SchemaSHA256 = c.ManagementProfile, c.APIVersion, "development-declarations", c.APISchemaSHA256
	if err := checkDeclarationReport(u, c, start); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*declarationReport){
		func(u *declarationReport) { u.ProviderRevision = u.HarnessRevision },
		func(u *declarationReport) { u.HarnessRevision = "unknown" },
		func(u *declarationReport) { u.PlatformRevision = "unknown" },
		func(u *declarationReport) { u.ProviderArtifact.SHA256 = strings.Repeat("a", 64) },
		func(u *declarationReport) { u.Result = "skipped" },
		func(u *declarationReport) { u.Contract.Name += ".peer" },
		func(u *declarationReport) { u.CheckedAt = start.Add(-time.Second) },
		func(u *declarationReport) { u.Cases = u.Cases[:len(u.Cases)-1] },
		func(u *declarationReport) { u.Cases[1] = u.Cases[0] },
	} {
		bad := u
		bad.Cases = append([]string{}, u.Cases...)
		change(&bad)
		if checkDeclarationReport(bad, c, start) == nil {
			t.Fatal("incomplete/mismatched declaration evidence accepted")
		}
	}
}
