package liveplatform

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

func localConfig(t *testing.T) Config {
	c := validConfig(t)
	p := platformclient.CurrentProfile()
	c.ProviderRevision = strings.Repeat("b", 40)
	c.ProviderVersion = "0.1.0-dev.g" + c.ProviderRevision
	c.Gate = "metadata"
	c.EvidenceKind = "local-platform-metadata"
	c.FixtureCheckout = t.TempDir()
	c.ManagementProfile = p.Name
	c.ProfileSchemaSHA256 = p.SchemaSHA256
	c.Endpoint = ""
	c.CAFile = ""
	c.CredentialFile = ""
	c.Scope = platformclient.Scope{}
	c.MaxObjects = 32
	return c
}
func TestLocalMetadataRequiresExplicitIsolatedProfile(t *testing.T) {
	c := localConfig(t)
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	if _, err := Preflight(c); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*Config){
		func(c *Config) { c.EvidenceKind = "hosted" }, func(c *Config) { c.Gate = "service-graph" },
		func(c *Config) { c.ManagementProfile = "unknown" }, func(c *Config) { c.ProfileSchemaSHA256 = strings.Repeat("0", 64) },
		func(c *Config) { c.FixtureCheckout = "relative" }, func(c *Config) { c.MaxObjects = 31 },
		func(c *Config) { c.CredentialFile = "/private/token" }, func(c *Config) { c.Endpoint = "https://live.example.invalid" },
		func(c *Config) { c.Scope.OrganizationID = "org" },
	} {
		bad := c
		change(&bad)
		if bad.Validate() == nil {
			t.Fatal("invalid metadata target accepted")
		}
	}
	r, _ := Preflight(c)
	if _, err := RunMetadata(c, r); err == nil {
		t.Fatal("uncommitted/missing fixture accepted")
	}
	if r.ValidateEvidence(c) == nil {
		t.Fatal("preflight alone qualified metadata")
	}
}
func TestMetadataEvidenceBindsSourceArtifactAndExecutedSchedules(t *testing.T) {
	c := localConfig(t)
	started := time.Now().Add(-time.Minute)
	a := installedArtifact{Repository: "spoke-sh/terraform-provider-spoke", Revision: strings.Repeat("b", 40), Version: c.ProviderVersion, System: "x86_64-linux", SHA256: c.ArtifactSHA256, SchemaSHA256: c.APISchemaSHA256}
	u := upstreamReport{Schema: 1, Commit: c.PlatformRevision, Artifact: a, Result: "passed", Profile: c.ManagementProfile, PlatformSchemaSHA256: c.ProfileSchemaSHA256, Command: "pnpm test:provider:installed", CompletedAt: time.Now(), TerraformVersion: "1.15.3", System: "linux_amd64", Schedules: append([]string{}, metadataSchedules...)}
	if err := checkUpstreamReport(u, c, a, started); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*upstreamReport){
		func(u *upstreamReport) { u.Commit = strings.Repeat("c", 40) }, func(u *upstreamReport) { u.Artifact.SHA256 = strings.Repeat("c", 64) },
		func(u *upstreamReport) { u.Result = "skipped" }, func(u *upstreamReport) { u.Profile = "other" },
		func(u *upstreamReport) { u.PlatformSchemaSHA256 = strings.Repeat("c", 64) }, func(u *upstreamReport) { u.CompletedAt = started.Add(-time.Second) },
		func(u *upstreamReport) { u.Schedules = u.Schedules[:len(u.Schedules)-1] }, func(u *upstreamReport) { u.Schedules[1] = u.Schedules[0] },
		func(u *upstreamReport) { u.TerraformVersion = "unknown" }, func(u *upstreamReport) { u.System = "darwin_arm64" },
	} {
		bad := u
		bad.Schedules = append([]string{}, u.Schedules...)
		change(&bad)
		if checkUpstreamReport(bad, c, a, started) == nil {
			t.Fatal("unqualified evidence accepted")
		}
	}
	r := passedReport(c)
	r.EvidenceKind = c.EvidenceKind
	r.ManagementProfile = c.ManagementProfile
	r.ProfileSchemaSHA256 = c.ProfileSchemaSHA256
	r.ProviderRevision = a.Revision
	r.ProviderVersion = a.Version
	r.TerraformVersion = u.TerraformVersion
	r.System = u.System
	if err := r.ValidateEvidence(c); err != nil {
		t.Fatal(err)
	}
	full := validConfig(t)
	if r.ValidateEvidence(full) == nil || r.validateCases(full) == nil {
		t.Fatal("metadata qualified full services")
	}
}
func TestFixtureRejectsDirtyOrWrongSourceBeforeExecution(t *testing.T) {
	c := localConfig(t)
	if err := os.WriteFile(filepath.Join(c.FixtureCheckout, "unreviewed"), []byte("dirty"), 0600); err != nil {
		t.Fatal(err)
	}
	if cleanFixture(c) == nil {
		t.Fatal("unreviewed source accepted")
	}
}
