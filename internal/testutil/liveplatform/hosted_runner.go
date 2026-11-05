package liveplatform

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

// StopConsumerTree stops only the selected local command and its descendants,
// including provider children that created independent process groups.
func StopConsumerTree(pid int) error { return stopFixtureTree(pid) }

// WritePrivateJSON atomically checkpoints evidence without exposing file contents
// or following an existing output symlink. The enclosing directory is private.
func WritePrivateJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".checkpoint-*")
	if err != nil {
		return errors.New("cannot create private checkpoint")
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(append(data, '\n')); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil || closeErr != nil {
		return errors.New("cannot persist checkpoint")
	}
	if err = os.Rename(f.Name(), path); err != nil {
		return errors.New("cannot replace checkpoint")
	}
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

func privateDirectory(path string) error {
	if err := os.Mkdir(path, 0700); err != nil && !os.IsExist(err) {
		return errors.New("cannot create run directory; its parent must exist")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0700 {
		return errors.New("run directory must be a private real directory with mode 0700")
	}
	return nil
}

func cleanConsumerHarness(c Config) error {
	wd, err := os.Getwd()
	if err != nil || wd != filepath.Join(c.HarnessCheckout, "internal/provider") {
		return errors.New("live test must execute from the selected provider harness checkout")
	}
	for _, check := range []struct {
		args []string
		want string
	}{
		{[]string{"rev-parse", "HEAD"}, c.HarnessRevision},
		{[]string{"status", "--porcelain"}, ""},
	} {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		out, err := exec.CommandContext(ctx, "git", append([]string{"-C", c.HarnessCheckout}, check.args...)...).Output()
		cancel()
		if err != nil || strings.TrimSpace(string(out)) != check.want {
			return errors.New("consumer harness must be clean at its selected revision")
		}
	}
	return nil
}

func checkConsumerArtifact(c Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, c.Artifact, "-version").Output()
	var identity struct{ Version, Revision, Contract, Platform string }
	if err != nil || json.Unmarshal(out, &identity) != nil || identity.Version != c.ProviderVersion || identity.Revision != c.ProviderRevision || identity.Contract != c.APIVersion || identity.Platform != "linux_amd64" {
		return errors.New("installed consumer artifact identity mismatch; no source-build fallback")
	}
	return nil
}

// RunHosted invokes the current clean harness's dedicated test driver. That
// driver runs the exact installed provider and only consumer HTTP operations.
// No Platform checkout, deployment tool, native process or operator credential
// is involved. A single flock excludes concurrent runs using the same journal.
func RunHosted(c Config, r Report) (Report, error) {
	fail := func(err error) (Report, error) {
		r.Status = "failed"
		r.Blockers = []string{"consumer_execution_incomplete_private_state_retained"}
		return r, err
	}
	if err := c.Validate(); err != nil {
		return fail(err)
	}
	if c.EvidenceKind != HostedEvidence || runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return fail(errors.New("hosted consumer runner requires Linux amd64"))
	}
	if err := cleanConsumerHarness(c); err != nil {
		return fail(err)
	}
	if _, err := Preflight(c); err != nil {
		return fail(err)
	}
	if err := checkConsumerArtifact(c); err != nil {
		return fail(err)
	}
	if err := privateDirectory(c.RunDirectory); err != nil {
		return fail(err)
	}
	lock, err := os.OpenFile(filepath.Join(c.RunDirectory, ".lock"), os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return fail(errors.New("cannot lock run directory"))
	}
	defer lock.Close()
	if err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return fail(errors.New("consumer run already active"))
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	configPath := filepath.Join(c.RunDirectory, "inputs.json")
	if data, err := os.ReadFile(configPath); err == nil {
		var previous Config
		if json.Unmarshal(data, &previous) != nil || previous != c {
			return fail(errors.New("run inputs changed; preserve the original keys and state for recovery"))
		}
	} else if os.IsNotExist(err) {
		entries, e := os.ReadDir(c.RunDirectory)
		if e != nil || len(entries) != 1 {
			return fail(errors.New("new run requires an empty private directory"))
		}
		if err := WritePrivateJSON(configPath, c); err != nil {
			return fail(err)
		}
	} else {
		return fail(errors.New("cannot inspect prior run inputs"))
	}
	if data, err := os.ReadFile(filepath.Join(c.RunDirectory, "execution.json")); err == nil {
		var prior Report
		if json.Unmarshal(data, &prior) != nil || prior.Status == "passed" {
			return fail(errors.New("completed or invalid run cannot be reused as fresh evidence"))
		}
	}
	binary, err := os.Executable()
	if err != nil {
		return fail(errors.New("cannot locate consumer test driver"))
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.DeadlineSeconds)*time.Second)
	defer cancel()
	list, err := exec.CommandContext(ctx, binary, "-test.list=^TestHostedMetadataConsumer$").Output()
	if err != nil || strings.TrimSpace(string(list)) != "TestHostedMetadataConsumer" {
		return fail(errors.New("required hosted consumer driver missing"))
	}
	log, err := os.OpenFile(filepath.Join(c.RunDirectory, "driver.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return fail(errors.New("cannot open private consumer log"))
	}
	defer log.Close()
	terraform, err := exec.LookPath("terraform")
	if err != nil {
		return fail(errors.New("locked Terraform executable required"))
	}
	command := exec.CommandContext(ctx, binary, "-test.run=^TestHostedMetadataConsumer$", "-test.v", "-test.timeout="+time.Duration(c.DeadlineSeconds*int(time.Second)).String())
	command.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + c.RunDirectory, "TMPDIR=" + c.RunDirectory,
		"SPOKE_HOSTED_CONSUMER_CONFIG=" + configPath, "SPOKE_TEST_PROVIDER_MIRROR=" + c.Mirror,
		"SPOKE_TEST_PROVIDER_VERSION=" + c.ProviderVersion, "TF_ACC_TERRAFORM_PATH=" + terraform}
	command.Stdout, command.Stderr = log, log
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error { return stopFixtureTree(command.Process.Pid) }
	command.WaitDelay = 5 * time.Second
	started := time.Now()
	err = command.Run()
	data, readErr := os.ReadFile(filepath.Join(c.RunDirectory, "execution.json"))
	if readErr == nil {
		var partial Report
		if json.Unmarshal(data, &partial) == nil && partial.Consumer != nil && partial.Consumer.RunID == c.RunID {
			r = partial
		}
	}
	if err != nil {
		return fail(errors.New("consumer execution failed; original inputs, keys and state retained in the private run directory"))
	}
	if readErr != nil {
		return fail(errors.New("consumer execution did not write evidence"))
	}
	if err := cleanConsumerHarness(c); err != nil {
		return fail(err)
	}
	r, err = CheckHostedExecution(data, c, started)
	if err != nil {
		return fail(err)
	}
	return r, nil
}
