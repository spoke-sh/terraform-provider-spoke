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
)

// This test installs a real binary through Terraform's filesystem mirror. The
// consumer has no source imports or dev_overrides. Linux /proc lets the parent
// kill only that provider after server acceptance, leaving Terraform and the
// independent fixture ledger alive.
func TestAccMockProviderProcessRecovery(t *testing.T) {
	s := mockAcceptance(t)
	root := t.TempDir()
	mirror := filepath.Join(root, "mirror")
	binary := filepath.Join(mirror, "registry.terraform.io", "spoke-sh", "spoke", "0.0.1", runtime.GOOS+"_"+runtime.GOARCH, "terraform-provider-spoke_v0.0.1")
	if err := os.MkdirAll(filepath.Dir(binary), 0700); err != nil {
		t.Fatal(err)
	}
	buildCtx, buildCancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer buildCancel()
	build := exec.CommandContext(buildCtx, "go", "build", "-ldflags=-X main.version=0.0.1", "-o", binary, "../..")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	consumer := filepath.Join(root, "consumer")
	if err := os.Mkdir(consumer, 0700); err != nil {
		t.Fatal(err)
	}
	cli := filepath.Join(root, "terraform.rc")
	installation := fmt.Sprintf(`provider_installation {
  filesystem_mirror {
    path = %q
    include = ["registry.terraform.io/spoke-sh/spoke"]
  }
}
`, mirror)
	config := `terraform {
  required_providers {
    spoke = { source = "spoke-sh/spoke", version = "0.0.1" }
  }
}
` + bucketConfig(s, "process-test", "deletion_protection = false")
	for path, contents := range map[string]string{cli: installation, filepath.Join(consumer, "main.tf"): config} {
		if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
	}
	// Do not inherit user Terraform CLI arguments, logging, reattach, or cache
	// settings into the isolated consumer.
	env := []string{}
	for _, v := range os.Environ() {
		key, _, _ := strings.Cut(v, "=")
		if !strings.HasPrefix(key, "TF_") && key != "SPOKE_TOKEN" && key != "PATH" {
			env = append(env, v)
		}
	}
	env = append(env, "TF_CLI_CONFIG_FILE="+cli, "TF_IN_AUTOMATION=1", "CHECKPOINT_DISABLE=1", "PATH="+filepath.Dir(os.Getenv("TF_ACC_TERRAFORM_PATH")))
	command := func(ctx context.Context, args ...string) *exec.Cmd {
		cmd := exec.CommandContext(ctx, os.Getenv("TF_ACC_TERRAFORM_PATH"), args...)
		cmd.Dir = consumer
		cmd.Env = env
		return cmd
	}
	run := func(args ...string) []byte {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		out, err := command(ctx, args...).CombinedOutput()
		if err != nil {
			t.Fatalf("terraform %v: %v\n%s", args, err, out)
		}
		return out
	}
	run("init", "-backend=false", "-input=false", "-no-color")
	lock, err := os.ReadFile(filepath.Join(consumer, ".terraform.lock.hcl"))
	if err != nil || !bytes.Contains(lock, []byte("h1:")) {
		t.Fatalf("missing provider checksum lock: %v", err)
	}
	run("init", "-backend=false", "-input=false", "-lockfile=readonly", "-no-color")
	s.HoldOperations(true)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	apply := command(ctx, "apply", "-auto-approve", "-input=false", "-no-color")
	var output bytes.Buffer
	apply.Stdout = &output
	apply.Stderr = &output
	if err := apply.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	finished := make(chan struct{})
	go func() { done <- apply.Wait(); close(finished) }()
	t.Cleanup(func() { cancel(); <-finished })
	var accepted api.Operation
	select {
	case accepted = <-s.Accepted():
	case err := <-done:
		t.Fatalf("apply ended before acceptance: %v\n%s", err, output.String())
	case <-ctx.Done():
		t.Fatal("timed out waiting for acceptance")
	}
	childFiles, err := filepath.Glob(fmt.Sprintf("/proc/%d/task/*/children", apply.Process.Pid))
	if err != nil {
		t.Fatal(err)
	}
	var children []byte
	for _, path := range childFiles {
		b, err := os.ReadFile(path)
		if err == nil {
			children = append(children, append(b, ' ')...)
		}
	}
	killed := false
	for _, child := range strings.Fields(string(children)) {
		path, err := os.Readlink("/proc/" + child + "/exe")
		if err != nil {
			continue
		}
		if path != binary {
			continue
		}
		pid, err := strconv.Atoi(child)
		if err != nil {
			t.Fatal(err)
		}
		process, err := os.FindProcess(pid)
		if err != nil {
			t.Fatal(err)
		}
		if err := process.Kill(); err != nil {
			t.Fatal(err)
		}
		killed = true
	}
	if !killed {
		t.Fatal("could not identify the installed provider process")
	}
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("apply succeeded after provider death")
		}
	case <-ctx.Done():
		t.Fatal("Terraform did not report provider death")
	}
	s.CompleteAll()
	s.HoldOperations(false)
	run("apply", "-auto-approve", "-input=false", "-no-color")
	if s.OperationCount() != 1 || s.BucketCount() != 1 {
		t.Fatal("restart duplicated the accepted create")
	}
	state := run("state", "pull")
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
		t.Fatal(err)
	}
	found := false
	for _, r := range decoded.Resources {
		if r.Type == "spoke_bucket" {
			for _, i := range r.Instances {
				found = i.Attributes.ID == canonicalID(s.Scope, accepted.TargetID)
			}
		}
	}
	if !found {
		t.Fatal("state did not recover the original scoped bucket identity")
	}
	if bytes.Contains(state, []byte(s.Token)) {
		t.Fatal("credential bytes leaked into state")
	}
	run("plan", "-detailed-exitcode", "-input=false", "-no-color")
	run("destroy", "-auto-approve", "-input=false", "-no-color")
	if s.BucketCount() != 0 {
		t.Fatal("destroy left a bucket")
	}
}
