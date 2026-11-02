package liveplatform

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

type installedArtifact struct {
	Repository   string `json:"repository"`
	Revision     string `json:"revision"`
	Version      string `json:"version"`
	System       string `json:"system"`
	SHA256       string `json:"sha256"`
	SchemaSHA256 string `json:"schema_sha256"`
}
type upstreamReport struct {
	Schema               int               `json:"schema"`
	Commit               string            `json:"commit"`
	Artifact             installedArtifact `json:"artifact"`
	Result               string            `json:"result"`
	Profile              string            `json:"profile"`
	PlatformSchemaSHA256 string            `json:"platform_schema_sha256"`
	Schedules            []string          `json:"schedules"`
	Scope                string            `json:"scope"`
	Command              string            `json:"command"`
	CompletedAt          time.Time         `json:"completed_at"`
	TerraformVersion     string            `json:"terraform_version"`
	System               string            `json:"system"`
}

// These are emitted only after the Platform runner executes both raw HTTPS and
// ordinary installed Terraform, including actual process death and server restart.
var metadataSchedules = []string{
	"original HTTP contract", "uncertain cleanup original-operation recovery", "installed external drift convergence",
	"bounded pending discovery at 32 reservations", "same-owner token rotation", "last-slot quota race",
	"two gateways at identical native signer sequence", "provider death before native forwarding",
	"provider death after acceptance", "provider death after commit before reply", "provider recovery after terminal receipt expiry",
	"concurrent Terraform same and different keys",
}

func checkUpstreamReport(u upstreamReport, c Config, a installedArtifact, started time.Time) error {
	if u.Schema != 1 || u.Commit != c.PlatformRevision || u.Artifact != a || u.Result != "passed" || u.Profile != c.ManagementProfile || u.PlatformSchemaSHA256 != c.ProfileSchemaSHA256 || u.Command != "pnpm test:provider:installed" || u.CompletedAt.Before(started) || u.CompletedAt.After(time.Now().Add(time.Minute)) || u.TerraformVersion != "1.15.3" || u.System != "linux_amd64" {
		return errors.New("Platform metadata report identity, freshness or result mismatch")
	}
	required := map[string]bool{}
	for _, s := range metadataSchedules {
		required[s] = true
	}
	for _, s := range u.Schedules {
		if !required[s] {
			return errors.New("unknown or duplicate Platform schedule")
		}
		delete(required, s)
	}
	if len(required) != 0 {
		return errors.New("missing Platform metadata schedules")
	}
	return nil
}

func cleanFixture(c Config) error {
	for _, check := range []struct {
		args []string
		want string
	}{{[]string{"rev-parse", "HEAD"}, c.PlatformRevision}, {[]string{"status", "--porcelain"}, ""}} {
		cmd := exec.Command("git", append([]string{"-C", c.FixtureCheckout}, check.args...)...)
		out, err := cmd.Output()
		if err != nil || strings.TrimSpace(string(out)) != check.want {
			return errors.New("Platform fixture must be clean at the selected immutable revision")
		}
	}
	for path, digest := range map[string]string{"contracts/http/bucket-v1.schema.json": c.ProfileSchemaSHA256, "contracts/http/bucket-v1.vectors.json": platformclient.CurrentProfile().VectorsSHA256, "contracts/http/provider.schema.json": c.APISchemaSHA256} {
		b, err := os.ReadFile(filepath.Join(c.FixtureCheckout, path))
		if err != nil || fmt.Sprintf("%x", sha256.Sum256(b)) != digest {
			return errors.New("Platform fixture contract differs from pinned profile")
		}
	}
	return nil
}

// RunMetadata executes the Platform-owned isolated fixture; it never replaces
// native authority with a mock or interprets metadata success as service evidence.
// The fixed runner provisions only synthetic loopback services and consumes an
// exact installed binary whose digest and committed manifest agree.
func RunMetadata(c Config, r Report) (Report, error) {
	fail := func(err error) (Report, error) { r.Status = "failed"; return r, err }
	if err := c.Validate(); err != nil {
		return fail(err)
	}
	if c.EvidenceKind != "local-platform-metadata" || runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return fail(errors.New("metadata runner requires explicit local Platform selection on Linux amd64"))
	}
	if err := cleanFixture(c); err != nil {
		return fail(err)
	}
	var a installedArtifact
	data, err := os.ReadFile(filepath.Join(c.FixtureCheckout, "contracts/http/provider.json"))
	if err != nil || json.Unmarshal(data, &a) != nil || a.Repository != "spoke-sh/terraform-provider-spoke" || a.Revision != c.ProviderRevision || a.Version != c.ProviderVersion || a.SHA256 != c.ArtifactSHA256 || a.SchemaSHA256 != c.APISchemaSHA256 || a.System != "x86_64-linux" {
		return fail(errors.New("selected artifact differs from committed Platform consumer pin"))
	}
	identityContext, identityCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer identityCancel()
	identityCmd := exec.CommandContext(identityContext, c.Artifact, "-version")
	identityCmd.WaitDelay = 5 * time.Second
	identity, err := identityCmd.Output()
	var fields map[string]string
	if err != nil || json.Unmarshal(identity, &fields) != nil || fields["revision"] != a.Revision || fields["version"] != a.Version || fields["platform"] != "linux_amd64" || fields["contract"] != c.APIVersion || fields["contract_schema_sha256"] != c.APISchemaSHA256 || fields["management_profile"] != c.ManagementProfile || fields["profile_schema_sha256"] != c.ProfileSchemaSHA256 {
		return fail(errors.New("installed executable identity differs from selected profile and consumer pin"))
	}
	dir, err := os.MkdirTemp("/tmp", "spoke-platform-metadata-")
	if err != nil {
		return fail(err)
	}
	reportPath := filepath.Join(dir, "upstream.json")
	logPath := filepath.Join(dir, "runner.log")
	log, err := os.OpenFile(logPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return fail(err)
	}
	defer log.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.DeadlineSeconds)*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "nix", "develop", "--no-update-lock-file", "--command", "pnpm", "test:provider:installed")
	command.Dir = c.FixtureCheckout
	// Do not inherit credentials, Terraform overrides or another artifact/report.
	for _, key := range []string{"PATH", "HOME", "USER", "TMPDIR", "SSL_CERT_FILE", "NIX_SSL_CERT_FILE"} {
		if value, ok := os.LookupEnv(key); ok {
			command.Env = append(command.Env, key+"="+value)
		}
	}
	command.Env = append(command.Env, "SPOKE_HTTP_PROVIDER_BINARY="+c.Artifact, "SPOKE_HTTP_PROVIDER_REPORT="+reportPath)
	command.Stdout, command.Stderr = log, log
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error { return stopFixtureTree(command.Process.Pid) }
	command.WaitDelay = 5 * time.Second
	started := time.Now()
	if err = command.Run(); err != nil {
		return fail(fmt.Errorf("Platform metadata execution failed; private evidence retained at %s", dir))
	}
	if err = cleanFixture(c); err != nil {
		return fail(err)
	}
	data, err = os.ReadFile(reportPath)
	var upstream upstreamReport
	if err != nil || len(data) > 65536 || json.Unmarshal(data, &upstream) != nil {
		return fail(errors.New("missing or invalid fresh Platform evidence"))
	}
	if err = checkUpstreamReport(upstream, c, a, started); err != nil {
		return fail(err)
	}
	r.ProviderRevision, r.ProviderVersion, r.TerraformVersion, r.System = a.Revision, a.Version, upstream.TerraformVersion, upstream.System
	r.Status, r.Blockers = "passed", []string{}
	for i := range r.Cases {
		r.Cases[i].Status = "passed"
	}
	return r, r.ValidateEvidence(c)
}

// The fixture intentionally creates independent Terraform process groups for
// crash tests. Killing only the launcher group would leave those consumers alive.
// Freeze descendants before collecting their children, then kill the frozen tree.
// This runner is Linux-only; the external fixture retains interrupted state.
func stopFixtureTree(pid int) error {
	seen := map[int]bool{}
	var stop func(int)
	stop = func(current int) {
		if seen[current] {
			return
		}
		seen[current] = true
		_ = syscall.Kill(current, syscall.SIGSTOP)
		paths, _ := filepath.Glob(fmt.Sprintf("/proc/%d/task/*/children", current))
		for _, path := range paths {
			data, _ := os.ReadFile(path)
			for _, child := range strings.Fields(string(data)) {
				if id, err := strconv.Atoi(child); err == nil {
					stop(id)
				}
			}
		}
		_ = syscall.Kill(current, syscall.SIGKILL)
	}
	stop(pid)
	return nil
}
