package liveplatform

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

const declarationEvidence = "local-publication-terraform-declarations"

func (c Config) validateDeclarations() error {
	p := platformclient.CurrentPublicationContract()
	if c.EvidenceKind != declarationEvidence || c.Gate != "publication-declarations" || c.APIVersion != p.Version || c.APISchemaSHA256 != p.SchemaSHA256 || c.ManagementProfile != p.Name || c.ProfileSchemaSHA256 != p.SchemaSHA256 {
		return errors.New("explicit normal publication declaration profile and schema required")
	}
	if !sourceRevision.MatchString(c.ProviderRevision) || !sourceRevision.MatchString(c.PlatformRevision) || !sourceRevision.MatchString(c.HarnessRevision) || !strings.HasSuffix(c.ProviderVersion, "-dev.g"+c.ProviderRevision) {
		return errors.New("immutable provider, harness and Platform revisions required")
	}
	if !privatePath(c.HarnessCheckout) || !privatePath(c.FixtureCheckout) || !filepath.IsAbs(c.Mirror) || !filepath.IsAbs(c.Artifact) || !hexDigest.MatchString(c.ArtifactSHA256) {
		return errors.New("explicit clean checkouts, mirror, installed executable and hash required")
	}
	want := filepath.Join(c.Mirror, "registry.terraform.io/spoke-sh/spoke", c.ProviderVersion, "linux_amd64", "terraform-provider-spoke_v"+c.ProviderVersion)
	if c.Artifact != want || !runID.MatchString(c.RunID) || c.DeadlineSeconds < 1 || c.DeadlineSeconds > 1800 || c.MaxObjects != 64 {
		return errors.New("exact mirror layout and bounded declaration budgets required")
	}
	if c.Endpoint != "" || c.CAFile != "" || c.CredentialFile != "" || c.Scope != (platformclient.Scope{}) {
		return errors.New("isolated declaration fixture cannot select hosted credentials, endpoint or scope")
	}
	return nil
}

type declarationReport struct {
	EvidenceKind     string            `json:"evidence_kind"`
	Result           string            `json:"result"`
	ProviderRevision string            `json:"provider_revision"`
	HarnessRevision  string            `json:"harness_revision"`
	PlatformRevision string            `json:"platform_revision"`
	ProviderArtifact installedArtifact `json:"provider_artifact"`
	TerraformVersion string            `json:"terraform_version"`
	System           string            `json:"system"`
	CheckedAt        time.Time         `json:"checked_at"`
	Cases            []string          `json:"cases"`
	Contract         struct {
		Name         string `json:"name"`
		Version      string `json:"version"`
		Status       string `json:"status"`
		SchemaSHA256 string `json:"schema_sha256"`
	} `json:"contract"`
}

func checkDeclarationReport(u declarationReport, c Config, started time.Time) error {
	if u.EvidenceKind != declarationEvidence || u.Result != "passed" || u.ProviderRevision != c.ProviderRevision || u.HarnessRevision != c.HarnessRevision || u.PlatformRevision != c.PlatformRevision || u.ProviderArtifact.Revision != c.ProviderRevision || u.ProviderArtifact.Version != c.ProviderVersion || u.ProviderArtifact.SHA256 != c.ArtifactSHA256 || u.TerraformVersion != "1.15.3" || u.System != "linux_amd64" || u.CheckedAt.Before(started) || u.CheckedAt.After(time.Now().Add(time.Minute)) || u.Contract.Name != c.ManagementProfile || u.Contract.Version != c.APIVersion || u.Contract.Status != "development-declarations" || u.Contract.SchemaSHA256 != c.APISchemaSHA256 {
		return errors.New("declaration report identity, freshness or result mismatch")
	}
	required := map[string]bool{}
	for _, name := range RequiredCases(c.Gate) {
		required[name] = true
	}
	for _, name := range u.Cases {
		if !required[name] {
			return errors.New("unknown or duplicate declaration case")
		}
		delete(required, name)
	}
	if len(required) != 0 {
		return errors.New("missing declaration cases")
	}
	return nil
}

func cleanDeclarationSources(c Config) error {
	for path, revision := range map[string]string{c.HarnessCheckout: c.HarnessRevision, c.FixtureCheckout: c.PlatformRevision} {
		for _, check := range []struct {
			args []string
			want string
		}{
			{[]string{"rev-parse", "HEAD"}, revision}, {[]string{"status", "--porcelain"}, ""},
		} {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			out, err := exec.CommandContext(ctx, "git", append([]string{"-C", path}, check.args...)...).Output()
			cancel()
			if err != nil || strings.TrimSpace(string(out)) != check.want {
				return errors.New("declaration sources must be clean at their selected revisions")
			}
		}
	}
	return nil
}

// RunDeclarations invokes the reviewed provider harness and Platform-owned native
// fixture. Terraform always runs the exact installed artifact; Go compiles only
// the separate test driver. A fresh consumer copy uses committed readonly locks.
func RunDeclarations(c Config, r Report) (Report, error) {
	fail := func(err error) (Report, error) { r.Status = "failed"; return r, err }
	if err := c.Validate(); err != nil {
		return fail(err)
	}
	if c.EvidenceKind != declarationEvidence || runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return fail(errors.New("declaration runner requires Linux amd64"))
	}
	if err := cleanDeclarationSources(c); err != nil {
		return fail(err)
	}
	dir, err := os.MkdirTemp("", "spoke-publication-live-")
	if err != nil {
		return fail(err)
	}
	r.ExecutionReport = filepath.Join(dir, "execution.json")
	log, err := os.OpenFile(filepath.Join(dir, "runner.log"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return fail(err)
	}
	defer log.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.DeadlineSeconds)*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "nix", "develop", "--no-update-lock-file", "--command", "python3", "scripts/qualify-publication-terraform.py", "--platform", c.FixtureCheckout, "--mirror", c.Mirror, "--provider-revision", c.ProviderRevision, "--report", r.ExecutionReport, "--consumer")
	command.Dir = c.HarnessCheckout
	for _, key := range []string{"PATH", "HOME", "USER", "TMPDIR", "SSL_CERT_FILE", "NIX_SSL_CERT_FILE"} {
		if value, ok := os.LookupEnv(key); ok {
			command.Env = append(command.Env, key+"="+value)
		}
	}
	command.Stdout, command.Stderr = log, log
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error { return stopFixtureTree(command.Process.Pid) }
	command.WaitDelay = 5 * time.Second
	started := time.Now()
	if err = command.Run(); err != nil {
		return fail(fmt.Errorf("declaration execution failed; private evidence retained at %s", dir))
	}
	if err = cleanDeclarationSources(c); err != nil {
		return fail(err)
	}
	data, err := os.ReadFile(r.ExecutionReport)
	var u declarationReport
	if err != nil || len(data) > 65536 || json.Unmarshal(data, &u) != nil {
		return fail(errors.New("missing or invalid fresh declaration evidence"))
	}
	if err = checkDeclarationReport(u, c, started); err != nil {
		return fail(err)
	}
	r.ProviderRevision, r.ProviderVersion, r.HarnessRevision = c.ProviderRevision, c.ProviderVersion, c.HarnessRevision
	r.TerraformVersion, r.System, r.CheckedAt = u.TerraformVersion, u.System, u.CheckedAt.Format(time.RFC3339Nano)
	r.Status, r.Blockers = "passed", []string{}
	for i := range r.Cases {
		r.Cases[i].Status = "passed"
	}
	return r, r.ValidateEvidence(c)
}
