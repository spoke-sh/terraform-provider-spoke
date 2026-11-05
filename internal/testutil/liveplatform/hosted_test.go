package liveplatform

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

func hostedConfig(t *testing.T) Config {
	c := validConfig(t)
	p := api.CurrentProfile()
	c.Gate, c.EvidenceKind = "hosted-metadata", HostedEvidence
	c.ManagementProfile, c.ProfileSchemaSHA256 = p.Name, p.SchemaSHA256
	c.ProviderRevision, c.HarnessRevision = strings.Repeat("b", 40), strings.Repeat("c", 40)
	c.ProviderVersion = "0.1.0-dev.g" + c.ProviderRevision
	c.Mirror = filepath.Join(t.TempDir(), "mirror")
	data, err := os.ReadFile(c.Artifact)
	if err != nil {
		t.Fatal(err)
	}
	c.Artifact = filepath.Join(c.Mirror, "registry.terraform.io/spoke-sh/spoke", c.ProviderVersion, "linux_amd64", "terraform-provider-spoke_v"+c.ProviderVersion)
	if err := os.MkdirAll(filepath.Dir(c.Artifact), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(c.Artifact, data, 0700); err != nil {
		t.Fatal(err)
	}
	c.ExpectedPrincipal = "consumer-principal"
	c.HarnessCheckout = filepath.Join(t.TempDir(), "harness")
	c.RunDirectory = filepath.Join(t.TempDir(), "run")
	c.MaxObjects = 32
	return c
}

func hostedReport(c Config) Report {
	r := passedReport(c)
	r.EvidenceKind, r.ManagementProfile, r.ProfileSchemaSHA256 = c.EvidenceKind, c.ManagementProfile, c.ProfileSchemaSHA256
	r.ProviderRevision, r.ProviderVersion, r.HarnessRevision = c.ProviderRevision, c.ProviderVersion, c.HarnessRevision
	r.TerraformVersion, r.System = "1.15.3", "linux_amd64"
	r.CheckedAt = time.Now().UTC().Format(time.RFC3339Nano)
	r.ExecutionReport = filepath.Join(c.RunDirectory, "execution.json")
	r.Consumer = &ConsumerEvidence{Endpoint: c.Endpoint, Scope: c.Scope, Principal: c.ExpectedPrincipal, RunID: c.RunID, StartedAt: time.Now().Add(-time.Minute).UTC().Format(time.RFC3339Nano), Limitations: append([]string{}, ConsumerLimits...)}
	r.Consumer.AttemptStartedAt = r.Consumer.StartedAt
	return r
}

func TestHostedSelectionRequiresConsumerIdentity(t *testing.T) {
	c := hostedConfig(t)
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	r, err := Preflight(c)
	if err != nil || len(r.Cases) != 15 || r.Status != "blocked" {
		t.Fatal("consumer preflight must not claim execution", err)
	}
	// The CA/token do not exist: preflight cannot read credentials or use HTTP.
	for _, change := range []func(*Config){
		func(c *Config) { c.EvidenceKind = "local-platform-metadata" },
		func(c *Config) { c.Gate = "service-graph" },
		func(c *Config) { c.Endpoint = "http://localhost" },
		func(c *Config) { c.Endpoint += "/" },
		func(c *Config) { c.Endpoint += "?token=secret" },
		func(c *Config) { c.ExpectedPrincipal = "" },
		func(c *Config) { c.Scope.EnvironmentID = "" },
		func(c *Config) { c.ManagementProfile = "" },
		func(c *Config) { c.ProfileSchemaSHA256 = strings.Repeat("0", 64) },
		func(c *Config) { c.ProviderVersion = "latest" },
		func(c *Config) { c.ProviderRevision = "unknown" },
		func(c *Config) { c.HarnessRevision = "main" },
		func(c *Config) { c.Mirror = "relative" },
		func(c *Config) { c.Artifact += "-other" },
		func(c *Config) { c.FixtureCheckout = "/platform" },
		func(c *Config) { c.CredentialFile = "/nix/store/secret" },
		func(c *Config) { c.RunDirectory = c.HarnessCheckout + "/run" },
		func(c *Config) { c.RunDirectory = "/" },
		func(c *Config) { c.MaxObjects = 64 },
		func(c *Config) { c.DeadlineSeconds = 1801 },
	} {
		bad := c
		change(&bad)
		if bad.Validate() == nil {
			t.Fatal("invalid consumer target accepted", bad.Gate)
		}
	}
	local := validConfig(t)
	local.RunDirectory = c.RunDirectory
	if local.Validate() == nil {
		t.Fatal("consumer fields silently changed another gate")
	}
}

func TestHostedReportCannotReuseOrBroadenEvidence(t *testing.T) {
	c := hostedConfig(t)
	r := hostedReport(c)
	if err := r.ValidateEvidence(c); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(r)
	if _, err := CheckHostedExecution(data, c, time.Now().Add(-2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := CheckHostedExecution(data, c, time.Now().Add(time.Second)); err == nil {
		t.Fatal("stale execution accepted")
	}
	for _, change := range []func(*Report){
		func(r *Report) { r.Consumer = nil },
		func(r *Report) { r.Consumer.Principal = "foreign" },
		func(r *Report) { r.Consumer.Endpoint = "https://other.invalid" },
		func(r *Report) { r.Consumer.Scope.EnvironmentID = "other" },
		func(r *Report) { r.Consumer.RunID = "other-run" },
		func(r *Report) { r.Consumer.Limitations = nil },
		func(r *Report) { r.Consumer.StartedAt = time.Now().Add(time.Hour).Format(time.RFC3339Nano) },
		func(r *Report) { r.CheckedAt = time.Now().Add(time.Hour).Format(time.RFC3339Nano) },
		func(r *Report) { r.ExecutionReport = "/other/report.json" },
		func(r *Report) { r.HarnessRevision = c.ProviderRevision },
		func(r *Report) { r.ProviderRevision = c.HarnessRevision },
		func(r *Report) { r.TerraformVersion = "latest" },
		func(r *Report) { r.EvidenceKind = "synthetic-consumer" },
		func(r *Report) { r.Cases[0].Status = "skipped" },
		func(r *Report) { r.Cases[0] = r.Cases[1] },
		func(r *Report) { r.Cases = r.Cases[1:] },
	} {
		bad := hostedReport(c)
		change(&bad)
		if bad.ValidateEvidence(c) == nil {
			t.Fatal("invalid consumer report accepted")
		}
	}
	service := validConfig(t)
	if r.ValidateEvidence(service) == nil {
		t.Fatal("consumer metadata qualified services")
	}
	if len(RequiredCases("service-graph")) != 20 {
		t.Fatal("consumer cases expanded the release manifest")
	}
}

func TestPrivateCheckpointDoesNotFollowOutputSymlink(t *testing.T) {
	dir := t.TempDir()
	victim := filepath.Join(dir, "victim")
	output := filepath.Join(dir, "report.json")
	if err := os.WriteFile(victim, []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(victim, output); err != nil {
		t.Fatal(err)
	}
	if err := WritePrivateJSON(output, map[string]string{"status": "blocked"}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(victim)
	if string(data) != "preserve" {
		t.Fatal("checkpoint overwrote linked file")
	}
	info, _ := os.Lstat(output)
	if !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
		t.Fatal("checkpoint is not private regular file")
	}
}
