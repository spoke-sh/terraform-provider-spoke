//go:build linux

package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
	"github.com/spoke-sh/terraform-provider-spoke/internal/testutil/mockplatform"
)

type consumer struct {
	t           *testing.T
	dir, binary string
	env         []string
}

func buildInstalledProvider(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	binary := filepath.Join(root, "registry.terraform.io", "spoke-sh", "spoke", "0.0.1", runtime.GOOS+"_"+runtime.GOARCH, "terraform-provider-spoke_v0.0.1")
	if err := os.MkdirAll(filepath.Dir(binary), 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if out, err := exec.CommandContext(ctx, "go", "build", "-ldflags=-X main.version=0.0.1", "-o", binary, "../..").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	return binary
}
func newConsumer(t *testing.T, binary, config string) *consumer {
	t.Helper()
	p := &consumer{t: t, dir: t.TempDir(), binary: binary}
	mirror := binary
	for range 6 {
		mirror = filepath.Dir(mirror)
	}
	cli := filepath.Join(p.dir, "terraform.rc")
	installation := fmt.Sprintf("provider_installation {\n filesystem_mirror {\n path = %q\n include = [\"registry.terraform.io/spoke-sh/spoke\"]\n }\n}\n", mirror)
	if err := os.WriteFile(cli, []byte(installation), 0600); err != nil {
		t.Fatal(err)
	}
	for _, v := range os.Environ() {
		key, _, _ := strings.Cut(v, "=")
		if !strings.HasPrefix(key, "TF_") && key != "SPOKE_TOKEN" && key != "PATH" {
			p.env = append(p.env, v)
		}
	}
	p.env = append(p.env, "TF_CLI_CONFIG_FILE="+cli, "TF_IN_AUTOMATION=1", "CHECKPOINT_DISABLE=1", "PATH="+filepath.Dir(os.Getenv("TF_ACC_TERRAFORM_PATH")))
	p.config(config)
	p.run("init", "-backend=false", "-input=false", "-no-color")
	lock, err := os.ReadFile(filepath.Join(p.dir, ".terraform.lock.hcl"))
	if err != nil || !bytes.Contains(lock, []byte("h1:")) {
		t.Fatalf("missing checksum lock: %v", err)
	}
	p.run("init", "-backend=false", "-input=false", "-lockfile=readonly", "-no-color")
	return p
}
func (p *consumer) config(config string) {
	p.t.Helper()
	config = "terraform {\n required_providers {\n spoke = { source = \"spoke-sh/spoke\", version = \"0.0.1\" }\n }\n}\n" + config
	if err := os.WriteFile(filepath.Join(p.dir, "main.tf"), []byte(config), 0600); err != nil {
		p.t.Fatal(err)
	}
}
func (p *consumer) command(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, os.Getenv("TF_ACC_TERRAFORM_PATH"), args...)
	cmd.Dir = p.dir
	cmd.Env = p.env
	return cmd
}
func (p *consumer) attempt(args ...string) ([]byte, error) {
	p.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	return p.command(ctx, args...).CombinedOutput()
}
func (p *consumer) run(args ...string) []byte {
	p.t.Helper()
	out, err := p.attempt(args...)
	if err != nil {
		p.t.Fatalf("terraform %v: %v\n%s", args, err, out)
	}
	return out
}
func (p *consumer) apply()   { p.run("apply", "-auto-approve", "-input=false", "-no-color") }
func (p *consumer) destroy() { p.run("destroy", "-auto-approve", "-input=false", "-no-color") }
func (p *consumer) bucketID() string {
	p.t.Helper()
	state := p.run("state", "pull")
	var decoded struct {
		Resources []struct {
			Type      string `json:"type"`
			Instances []struct {
				Attributes struct {
					ID string `json:"id"`
				} `json:"attributes"`
			} `json:"instances"`
		} `json:"resources"`
	}
	if err := json.Unmarshal(state, &decoded); err != nil {
		p.t.Fatal(err)
	}
	for _, r := range decoded.Resources {
		if r.Type == "spoke_bucket" && len(r.Instances) > 0 {
			return r.Instances[0].Attributes.ID
		}
	}
	return ""
}
func (p *consumer) killAt(b *mockplatform.Barrier, args ...string) api.Operation {
	p.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmd := p.command(ctx, args...)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Start(); err != nil {
		p.t.Fatal(err)
	}
	done := make(chan error, 1)
	finished := make(chan struct{})
	go func() { done <- cmd.Wait(); close(finished) }()
	defer func() { cancel(); <-finished }()
	var op api.Operation
	select {
	case op = <-b.Reached:
	case err := <-done:
		p.t.Fatalf("Terraform ended before barrier: %v\n%s", err, output.String())
	case <-ctx.Done():
		p.t.Fatal("barrier deadline")
	}
	childrenFiles, err := filepath.Glob(fmt.Sprintf("/proc/%d/task/*/children", cmd.Process.Pid))
	if err != nil {
		p.t.Fatal(err)
	}
	killed := false
	seen := map[string]bool{}
	for _, path := range childrenFiles {
		children, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, child := range strings.Fields(string(children)) {
			if seen[child] {
				continue
			}
			seen[child] = true
			executable, err := os.Readlink("/proc/" + child + "/exe")
			if err != nil || executable != p.binary {
				continue
			}
			pid, err := strconv.Atoi(child)
			if err != nil {
				p.t.Fatal(err)
			}
			process, err := os.FindProcess(pid)
			if err != nil {
				p.t.Fatal(err)
			}
			if err = process.Kill(); err != nil {
				p.t.Fatal(err)
			}
			killed = true
		}
	}
	if !killed {
		p.t.Fatal("installed provider process not found")
	}
	select {
	case err := <-done:
		if err == nil {
			p.t.Fatal("Terraform succeeded after provider death")
		}
	case <-ctx.Done():
		p.t.Fatal("Terraform did not report provider death")
	}
	return op
}

func TestAccMockProviderProcessRecovery(t *testing.T) {
	// Qualify the installed binary, rather than in-process protocol factories.
	_ = mockAcceptance(t)
	binary := buildInstalledProvider(t)
	for _, kind := range []string{"create", "update", "delete"} {
		for _, phase := range []string{"before_dispatch", "before_accept", "after_accept", "after_complete"} {
			t.Run(kind+"/"+phase, func(t *testing.T) {
				s := mockAcceptance(t)
				config := bucketConfig(s, "process-test", "deletion_protection = false")
				p := newConsumer(t, binary, config)
				priorOps := 0
				originalID := ""
				if kind != "create" {
					p.apply()
					originalID = p.bucketID()
					priorOps = 1
				}
				args := []string{"apply", "-auto-approve", "-input=false", "-no-color"}
				if kind == "update" {
					p.config(bucketConfig(s, "renamed", "deletion_protection = false"))
				}
				if kind == "delete" {
					args[0] = "destroy"
				}
				barrier := s.BlockNext(phase)
				accepted := p.killAt(barrier, args...)
				// No heap or live server ledger is reused as recovery evidence.
				s.Restart(t)
				s.CompleteAll()
				p.run(args...)
				expected := 1
				if kind == "delete" {
					expected = 0
				}
				if s.OperationCount() != priorOps+1 || s.BucketCount() != expected {
					t.Fatalf("duplicated or lost work: operations=%d buckets=%d", s.OperationCount(), s.BucketCount())
				}
				if kind != "delete" {
					id := p.bucketID()
					if originalID != "" && id != originalID {
						t.Fatal("update changed identity")
					}
					if accepted.TargetID != "" && id != canonicalID(s.Scope, accepted.TargetID) {
						t.Fatal("did not recover accepted identity")
					}
					state := p.run("state", "pull")
					if bytes.Contains(state, []byte(s.Token)) {
						t.Fatal("credential leaked into state")
					}
					p.run("plan", "-detailed-exitcode", "-input=false", "-no-color")
					p.destroy()
				}
				if s.BucketCount() != 0 {
					t.Fatal("cleanup left managed bucket")
				}
			})
		}
	}
}

func TestAccMockPendingTimeoutAndRedaction(t *testing.T) {
	s := mockAcceptance(t)
	binary := buildInstalledProvider(t)
	config := strings.Replace(bucketConfig(s, "timeout-test", "deletion_protection = false"), "provider \"spoke\" {", "provider \"spoke\" {\n timeout_seconds = 2", 1)
	p := newConsumer(t, binary, config)
	s.HoldOperations(true)
	out, err := p.attempt("apply", "-auto-approve", "-input=false", "-no-color")
	if err == nil || !bytes.Contains(out, []byte("remains recoverable")) {
		t.Fatalf("pending became success or lost identity: %v\n%s", err, out)
	}
	if bytes.Contains(out, []byte(s.Token)) || s.OperationCount() != 1 || s.BucketCount() != 0 {
		t.Fatal("unsafe pending result")
	}
	s.Restart(t)
	s.CompleteAll()
	s.HoldOperations(false)
	p.apply()
	if s.OperationCount() != 1 {
		t.Fatal("timeout caused duplicate create")
	}
	id := p.bucketID()
	before := p.run("state", "pull")
	// A remote error can carry arbitrary secrets; the client prints only safe codes.
	s.SetErrorCanary(s.Token)
	p.env = append(p.env, "TF_LOG=TRACE", "TF_LOG_PATH="+filepath.Join(p.dir, "trace.log"))
	s.FailBucketReads("unavailable")
	out, err = p.attempt("plan", "-input=false", "-no-color")
	if err == nil || bytes.Contains(out, []byte(s.Token)) {
		t.Fatalf("unsafe outage: %v", err)
	}
	after := p.run("state", "pull")
	if !bytes.Equal(before, after) {
		t.Fatal("failed refresh modified state")
	}
	s.FailBucketReads("")
	p.env = append(p.env, "TF_LOG=TRACE", "TF_LOG_PATH="+filepath.Join(p.dir, "trace.log"))
	p.run("plan", "-out=saved.tfplan", "-input=false", "-no-color")
	plan := p.run("show", "-json", "saved.tfplan")
	log, err := os.ReadFile(filepath.Join(p.dir, "trace.log"))
	if err != nil {
		t.Fatal(err)
	}
	rawPlan, err := os.ReadFile(filepath.Join(p.dir, "saved.tfplan"))
	if err != nil {
		t.Fatal(err)
	}
	for _, data := range [][]byte{plan, rawPlan, log, p.run("state", "pull")} {
		if bytes.Contains(data, []byte(s.Token)) {
			t.Fatal("secret canary in plan, log or state")
		}
	}
	if p.bucketID() != id {
		t.Fatal("outage replaced resource")
	}
	p.destroy()
}

func TestAccMockIndependentStateOwnership(t *testing.T) {
	s := mockAcceptance(t)
	binary := buildInstalledProvider(t)
	config := bucketConfig(s, "same-name", "deletion_protection = false")
	first := newConsumer(t, binary, config)
	first.apply()
	firstID := first.bucketID()
	token := "synthetic-independent-owner"
	s.AddCredential(token, mockplatform.Principal{ID: "independent-owner", OrganizationID: s.Scope.OrganizationID, Admin: true})
	tokenFile := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(tokenFile, []byte(token), 0600); err != nil {
		t.Fatal(err)
	}
	otherConfig := strings.ReplaceAll(config, s.TokenFile, tokenFile)
	second := newConsumer(t, binary, otherConfig)
	out, err := second.attempt("apply", "-auto-approve", "-input=false", "-no-color")
	if err == nil || !bytes.Contains(out, []byte("ownership_conflict")) {
		t.Fatalf("second state took ownership: %v\n%s", err, out)
	}
	out, err = second.attempt("import", "-input=false", "-no-color", "spoke_bucket.test", firstID)
	if err == nil || !bytes.Contains(out, []byte("ownership_conflict")) {
		t.Fatalf("import transferred ownership: %v\n%s", err, out)
	}
	if s.OperationCount() != 1 || s.BucketCount() != 1 {
		t.Fatal("failed takeover mutated service")
	}
	second.config(strings.ReplaceAll(otherConfig, "acceptance-bucket", "independent-bucket"))
	second.apply()
	if second.bucketID() == firstID || s.BucketCount() != 2 {
		t.Fatal("same-name create adopted unrelated object")
	}
	second.destroy()
	if s.BucketCount() != 1 || first.bucketID() != firstID {
		t.Fatal("second owner cleanup damaged first state")
	}
	first.run("plan", "-detailed-exitcode", "-input=false", "-no-color")
	first.destroy()
}
