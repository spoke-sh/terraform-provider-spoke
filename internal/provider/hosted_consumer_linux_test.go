//go:build linux

package provider

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
	"github.com/spoke-sh/terraform-provider-spoke/internal/testutil/liveplatform"
	"github.com/spoke-sh/terraform-provider-spoke/internal/testutil/mockplatform"
)

func metadataClient(t *testing.T, c liveplatform.Config) *api.Client {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	client, err := liveplatform.ConsumerClient(ctx, c)
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func consumerProviderConfig(c liveplatform.Config) string {
	return fmt.Sprintf(`provider "spoke" {
 endpoint = %q
 organization_id = %q
 project_id = %q
 environment_id = %q
 token_file = %q
 ca_file = %q
 timeout_seconds = 30
}
`, c.Endpoint, c.Scope.OrganizationID, c.Scope.ProjectID, c.Scope.EnvironmentID, c.CredentialFile, c.CAFile)
}

func consumerBucketConfig(c liveplatform.Config, key, name string, protected bool) string {
	return consumerProviderConfig(c) + fmt.Sprintf(`resource "spoke_bucket" "test" {
 management_key = %q
 name = %q
 deletion_protection = %t
}
`, key, name, protected)
}

func installedConsumer(t *testing.T, c liveplatform.Config, binary, dir, config string) *consumer {
	t.Helper()
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal("cannot create consumer root")
	}
	p := &consumer{t: t, dir: dir, binary: binary, version: filepath.Base(filepath.Dir(filepath.Dir(binary))), stopTree: true}
	mirror := binary
	for range 6 {
		mirror = filepath.Dir(mirror)
	}
	cli := filepath.Join(dir, "terraform.rc")
	installation := fmt.Sprintf("provider_installation {\n filesystem_mirror {\n path = %q\n include = [\"registry.terraform.io/spoke-sh/spoke\"]\n }\n}\n", mirror)
	if err := os.WriteFile(cli, []byte(installation), 0600); err != nil {
		t.Fatal("cannot write consumer CLI configuration")
	}
	// No ambient token, Terraform override, plugin cache, credential helper,
	// compiler or cloud CLI reaches the installed Terraform process.
	// Plugin Unix sockets need a short pathname even when the retained run root
	// has a long name. Only temporary process files live here, never the journal.
	sockets, err := os.MkdirTemp("/tmp", "spoke-tf-")
	if err != nil {
		t.Fatal("cannot create private plugin socket directory")
	}
	t.Cleanup(func() { _ = os.RemoveAll(sockets) })
	p.env = []string{"HOME=" + dir, "TMPDIR=" + sockets, "LANG=C", "TF_CLI_CONFIG_FILE=" + cli, "TF_IN_AUTOMATION=1", "CHECKPOINT_DISABLE=1", "PATH=" + filepath.Dir(os.Getenv("TF_ACC_TERRAFORM_PATH"))}
	p.config(config)
	args := []string{"init", "-backend=false", "-input=false", "-no-color"}
	if _, err := os.Stat(filepath.Join(dir, ".terraform.lock.hcl")); err == nil {
		args = append(args, "-lockfile=readonly")
	}
	p.run(args...)
	lock, err := os.ReadFile(filepath.Join(dir, ".terraform.lock.hcl"))
	if err != nil || !bytes.Contains(lock, []byte("h1:")) {
		t.Fatal("missing installed consumer checksum lock")
	}
	p.run("init", "-backend=false", "-input=false", "-lockfile=readonly", "-no-color")
	return p
}

type metadataConsumer struct {
	t           *testing.T
	c           liveplatform.Config
	binary, key string
	client      *api.Client
	j           *consumerJournal
	p           *consumer
}

func (h *metadataConsumer) config(name string, protected bool) {
	h.p.config(consumerBucketConfig(h.c, h.key, name, protected))
}
func (h *metadataConsumer) receipt(key string) api.Management {
	h.t.Helper()
	m, err := h.client.InspectManagement(h.t.Context(), "bucket", key)
	if err != nil {
		h.t.Fatal(err)
	}
	return m
}
func (h *metadataConsumer) object(key string) api.Object {
	h.t.Helper()
	m := h.receipt(key)
	if m.ResourceID == "" {
		h.t.Fatal("expected retained resource")
	}
	o, err := h.client.GetObject(h.t.Context(), "bucket", m.ResourceID)
	if err != nil {
		h.t.Fatal(err)
	}
	return o
}
func (h *metadataConsumer) checkMetadata(o api.Object) {
	h.t.Helper()
	if o.OwnerID != h.c.ExpectedPrincipal || o.Status == nil || o.Status.Readiness != "pending" || o.Status.Reason != "dependency_unavailable" || o.Status.Endpoint != "" || o.Status.ActiveRevision != 0 {
		h.t.Fatal("bucket declaration unexpectedly claims usable storage or a different owner")
	}
}
func emptyConsumerPlan(p *consumer) { p.run("plan", "-detailed-exitcode", "-input=false", "-no-color") }

func (h *metadataConsumer) lifecycle() {
	j, p := h.j, h.p
	h.config("consumer-a", true)
	j.step("create", p.apply)
	o := captureConsumer(j, "created", func() api.Object { return h.object(h.key) })
	j.step("created-observations", func() { h.checkMetadata(o); emptyConsumerPlan(p) })
	j.step("import", func() {
		q := installedConsumer(h.t, h.c, h.binary, filepath.Join(h.c.RunDirectory, "import"), consumerBucketConfig(h.c, h.key, "consumer-a", true))
		if _, err := os.Stat(filepath.Join(q.dir, "terraform.tfstate")); os.IsNotExist(err) {
			q.run("import", "-input=false", "-no-color", "spoke_bucket.test", canonicalID(h.c.Scope, o.ID))
		} else if err != nil {
			h.t.Fatal("cannot inspect import recovery state")
		}
		emptyConsumerPlan(q)
	})
	j.step("discovery", func() {
		config := consumerProviderConfig(h.c) + fmt.Sprintf("data \"spoke_bucket\" \"exact\" { resource_id = %q }\ndata \"spoke_buckets\" \"all\" {}\n", o.ID)
		q := installedConsumer(h.t, h.c, h.binary, filepath.Join(h.c.RunDirectory, "discovery"), config)
		q.apply()
		emptyConsumerPlan(q)
		state := q.run("show", "-json")
		if !bytes.Contains(state, []byte(o.ID)) || !bytes.Contains(state, []byte("dependency_unavailable")) {
			h.t.Fatal("missing Terraform discovery evidence")
		}
		objects, err := h.client.List(h.t.Context(), "bucket")
		if err != nil || len(objects) != 1 || objects[0].ID != o.ID {
			h.t.Fatal("incomplete isolated bucket snapshot")
		}
	})
}

func (h *metadataConsumer) drift() {
	j, p := h.j, h.p
	h.config("consumer-b", true)
	j.step("rename-b", p.apply)
	b := captureConsumer(j, "revision-b", func() api.Object { return h.object(h.key) })
	h.config("consumer-a", true)
	j.step("rename-a", p.apply)
	a := captureConsumer(j, "revision-a", func() api.Object { return h.object(h.key) })
	j.step("a-b-a", func() {
		if a.ID != b.ID || a.Revision <= b.Revision || a.Name != "consumer-a" {
			h.t.Fatal("A to B to A lost identity or revision")
		}
		emptyConsumerPlan(p)
	})
	h.config("consumer-b", true)
	j.step("saved-plan", func() { p.run("plan", "-input=false", "-no-color", "-out=stale.plan") })
	intent := captureConsumer(j, "drift-intent", func() api.Intent {
		spec := a.Spec
		spec.Name = "consumer-external"
		return api.Intent{Kind: "update", ResourceKind: "bucket", ManagementKey: h.key, TargetID: a.ID, ExpectedRevision: a.Revision, Spec: &spec}
	})
	j.step("external-drift", func() {
		if _, err := h.client.Mutate(h.t.Context(), intent); err != nil {
			h.t.Fatal(err)
		}
	})
	j.step("stale-plan-conflict", func() {
		out, err := p.attempt("apply", "-input=false", "-no-color", "stale.plan")
		if err == nil || !bytes.Contains(out, []byte("revision_conflict")) {
			h.t.Fatal("stale Terraform plan did not preserve the revision fence")
		}
	})
	j.step("stale-http-conflict", func() {
		stale := intent
		spec := *intent.Spec
		spec.Name = "consumer-stale"
		stale.Spec = &spec
		if _, err := h.client.Mutate(h.t.Context(), stale); !api.IsCode(err, "revision_conflict") {
			h.t.Fatal("stale HTTP mutation was not rejected")
		}
	})
	h.config("consumer-a", true)
	j.step("repair-drift", func() { p.apply(); emptyConsumerPlan(p); h.checkMetadata(h.object(h.key)) })
}

func (h *metadataConsumer) denial() {
	h.j.step("invalid-credential", func() {
		p := h.p
		h.config("consumer-a", true)
		before := p.run("state", "pull")
		invalid := filepath.Join(p.dir, "invalid-token")
		if err := os.WriteFile(invalid, []byte("deliberately-invalid-consumer-credential"), 0600); err != nil {
			h.t.Fatal(err)
		}
		c := h.c
		c.CredentialFile = invalid
		p.config(consumerBucketConfig(c, h.key, "consumer-a", true))
		out, err := p.attempt("apply", "-refresh-only", "-auto-approve", "-input=false", "-no-color")
		if err == nil || !bytes.Contains(out, []byte("unauthenticated")) {
			h.t.Fatal("invalid credential was not refused")
		}
		if !bytes.Equal(before, p.run("state", "pull")) {
			h.t.Fatal("denied refresh changed Terraform state")
		}
		h.config("consumer-a", true)
	})
}

func (h *metadataConsumer) protection() {
	j, p := h.j, h.p
	h.config("consumer-a", true)
	j.step("protected-delete", func() {
		if _, err := p.attempt("destroy", "-auto-approve", "-input=false", "-no-color"); err == nil {
			h.t.Fatal("protected Terraform deletion succeeded")
		}
		o := h.object(h.key)
		_, err := h.client.Mutate(h.t.Context(), api.Intent{Kind: "delete", ResourceKind: "bucket", ManagementKey: h.key, TargetID: o.ID, ExpectedRevision: o.Revision})
		if !api.IsCode(err, "deletion_protected") {
			h.t.Fatal("protected API deletion succeeded")
		}
	})
	h.config("consumer-a", false)
	j.step("disable-protection", p.apply)
	intent := captureConsumer(j, "external-delete-intent", func() api.Intent {
		o := h.object(h.key)
		return api.Intent{Kind: "delete", ResourceKind: "bucket", ManagementKey: h.key, TargetID: o.ID, ExpectedRevision: o.Revision}
	})
	j.step("external-delete", func() {
		if _, err := h.client.Mutate(h.t.Context(), intent); err != nil {
			h.t.Fatal(err)
		}
	})
	j.step("authoritative-absence", func() {
		if _, err := h.client.GetObject(h.t.Context(), "bucket", intent.TargetID); !api.IsCode(err, "resource_not_found") {
			h.t.Fatal("missing authoritative absence")
		}
		p.run("apply", "-refresh-only", "-auto-approve", "-input=false", "-no-color")
		if p.bucketID() != "" {
			h.t.Fatal("authoritative absence retained Terraform object")
		}
		m := h.receipt(h.key)
		if m.ResourceID != "" || m.Operation == nil || m.Operation.Kind != "delete" || m.Operation.Status != "succeeded" {
			h.t.Fatal("missing retained deletion receipt")
		}
	})
}

func (h *metadataConsumer) observations() {
	h.j.step("observational-relay", func() {
		relay := newMetadataRelay(h.t, h.c, h.client)
		c := h.c
		c.Endpoint = relay.server.URL
		c.CAFile = relay.ca
		q := installedConsumer(h.t, c, h.binary, filepath.Join(h.c.RunDirectory, "observations"), consumerProviderConfig(c)+"data \"spoke_buckets\" \"all\" {}\n")
		q.apply()
		emptyConsumerPlan(q)
		q.run("apply", "-refresh-only", "-auto-approve", "-input=false", "-no-color")
		if relay.count() != 0 {
			h.t.Fatal("observation dispatched a mutation")
		}
	})
}

func (h *metadataConsumer) recovery(kind, phase string) {
	t := h.t
	name := kind + "_" + phase
	key := h.key + "-" + strings.ReplaceAll(name, "_", "-")
	j := openConsumerJournal(t, filepath.Join(h.c.RunDirectory, name))
	if j.Done["complete"] {
		return
	}
	relay := newMetadataRelay(t, h.c, h.client)
	c := h.c
	c.Endpoint = relay.server.URL
	c.CAFile = relay.ca
	p := installedConsumer(t, c, h.binary, filepath.Join(h.c.RunDirectory, name), consumerBucketConfig(c, key, "recovery-a", false))
	j.step("unused-key", func() {
		if m := h.receipt(key); m.Operation != nil {
			t.Fatal("recovery key is already owned; preserve original work")
		}
	})
	if kind != "create" {
		j.step("setup", p.apply)
	}
	prior := captureConsumer(j, "prior-receipt", func() api.Management { return h.receipt(key) })
	if kind == "update" {
		p.config(consumerBucketConfig(c, key, "recovery-b", false))
	}
	args := []string{"apply", "-auto-approve", "-input=false", "-no-color"}
	if kind == "delete" {
		args[0] = "destroy"
	}
	if !j.Done["killed"] {
		if j.Done["kill-started"] {
			t.Fatal("interrupted crash proof is incomplete; inspect the original receipt and retained Terraform root before a separate qualification run")
		}
		j.Done["kill-started"] = true
		j.save()
		barrier := relay.arm(phase)
		defer func() {
			relay.disarm()
			select {
			case <-barrier.Release:
			default:
				close(barrier.Release)
			}
		}()
		op := p.killAt(barrier, args...)
		captureConsumer(j, "intercepted", func() api.Operation { return op })
		if phase == "before_forward" && relay.count() != 0 && kind == "create" {
			t.Fatal("intercepted create reached the API")
		}
		j.Done["killed"] = true
		j.save()
		relay.disarm()
		close(barrier.Release)
	}
	original := captureConsumer(j, "intercepted", func() api.Operation { t.Fatal("lost original operation checkpoint"); return api.Operation{} })
	j.step("pre-retry-proof", func() {
		m := h.receipt(key)
		if phase == "before_forward" {
			if m.Generation != prior.Generation {
				t.Fatal("before-forward barrier dispatched work")
			}
		} else if original.ID == "" || m.Operation == nil || m.Operation.ID != original.ID || m.Generation != prior.Generation+1 {
			t.Fatal("accepted operation identity lost before retry")
		}
	})
	j.step("retry", func() { p.run(args...) })
	j.step("recovery-proof", func() {
		m := h.receipt(key)
		if m.Operation == nil || m.Generation != prior.Generation+1 || m.Operation.Status != "succeeded" || m.Operation.Kind != kind || m.Operation.OwnerID != h.c.ExpectedPrincipal {
			t.Fatal("retry duplicated or lost logical mutation")
		}
		if phase != "before_forward" && (m.Operation.ID != original.ID || m.Operation.Fingerprint != original.Fingerprint || m.Operation.TargetID != original.TargetID) {
			t.Fatal("retry replaced original operation")
		}
		if kind == "delete" {
			if m.ResourceID != "" || p.bucketID() != "" {
				t.Fatal("delete recovery retained object")
			}
		} else {
			o := h.object(key)
			h.checkMetadata(o)
			emptyConsumerPlan(p)
			if prior.ResourceID != "" && o.ID != prior.ResourceID {
				t.Fatal("update recovery replaced resource")
			}
		}
	})
	if kind != "delete" {
		j.step("cleanup", p.destroy)
	}
	j.step("complete", func() {
		m := h.receipt(key)
		if m.ResourceID != "" {
			t.Fatal("run resource retained after bounded cleanup")
		}
	})
}

func (h *metadataConsumer) confidentiality() {
	token, err := readFile(h.c.CredentialFile)
	if err != nil {
		h.t.Fatal("cannot inspect credential confinement")
	}
	token = bytes.TrimSpace(token)
	if len(token) == 0 {
		h.t.Fatal("empty credential")
	}
	err = filepath.WalkDir(h.c.RunDirectory, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".terraform" {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("unexpected private evidence symlink")
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Size() > 32<<20 {
			return fmt.Errorf("oversized private evidence")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if bytes.Contains(data, token) {
			return fmt.Errorf("credential found in consumer evidence")
		}
		return nil
	})
	if err != nil {
		h.t.Fatal("credential confinement check failed; raw evidence must remain private")
	}
}

// Both lanes execute these exact consumer operations. Synthetic callers receive
// no hosted Report and cannot select the live dispatcher through this function.
func exerciseMetadataConsumer(t *testing.T, c liveplatform.Config, binary string, record func(string)) {
	t.Helper()
	client := metadataClient(t, c)
	j := openConsumerJournal(t, filepath.Join(c.RunDirectory, "direct"))
	j.step("isolated-target", func() {
		objects, err := client.List(t.Context(), "bucket")
		if err != nil || len(objects) != 0 {
			t.Fatal("qualification requires an explicitly isolated empty bucket scope; no reset performed")
		}
	})
	key := fmt.Sprintf("consumer-%x", sha256.Sum256([]byte(c.RunID)))[:33]
	captureConsumer(j, "management-key", func() string { return key })
	j.step("unused-direct-key", func() {
		m, err := client.InspectManagement(t.Context(), "bucket", key)
		if err != nil || m.Operation != nil {
			t.Fatal("run ID already has retained work; use its original run directory for recovery")
		}
	})
	p := installedConsumer(t, c, binary, filepath.Join(c.RunDirectory, "direct"), consumerBucketConfig(c, key, "consumer-a", true))
	h := &metadataConsumer{t: t, c: c, binary: binary, key: key, client: client, j: j, p: p}
	steps := []func(){h.lifecycle, h.drift, h.denial, h.protection, h.observations}
	for _, kind := range []string{"create", "update", "delete"} {
		for _, phase := range []string{"before_forward", "after_acceptance", "after_success"} {
			steps = append(steps, func() { h.recovery(kind, phase) })
		}
	}
	steps = append(steps, h.confidentiality)
	for i, step := range steps {
		name := liveplatform.HostedCases[i]
		ok := t.Run(name, func(t *testing.T) { h.t = t; h.j.t = t; h.p.t = t; step() })
		if !ok {
			t.FailNow()
		}
		record(name)
	}
}

func TestHostedMetadataConsumer(t *testing.T) {
	path := os.Getenv("SPOKE_HOSTED_CONSUMER_CONFIG")
	if path == "" || os.Getenv("TF_ACC") != "1" || os.Getenv("SPOKE_TEST_TARGET") != "hosted-consumer-child" {
		t.Skip("explicit hosted consumer driver only")
	}
	c, err := liveplatform.Load(path)
	if err != nil || c.Gate != "hosted-metadata" {
		t.Fatal("invalid explicit hosted consumer configuration")
	}
	if os.Getenv("SPOKE_TEST_PROVIDER_MIRROR") != c.Mirror || os.Getenv("SPOKE_TEST_PROVIDER_VERSION") != c.ProviderVersion {
		t.Fatal("exact installed artifact required")
	}
	r, err := liveplatform.Preflight(c)
	if err != nil {
		t.Fatal(err)
	}
	r.HarnessRevision, r.ProviderRevision, r.ProviderVersion = c.HarnessRevision, c.ProviderRevision, c.ProviderVersion
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	version, err := exec.CommandContext(ctx, os.Getenv("TF_ACC_TERRAFORM_PATH"), "version", "-json").Output()
	var cli struct {
		Version  string `json:"terraform_version"`
		Platform string `json:"platform"`
	}
	if err != nil || json.Unmarshal(version, &cli) != nil || cli.Version != "1.15.3" || cli.Platform != "linux_amd64" {
		t.Fatal("locked Linux amd64 Terraform 1.15.3 required")
	}
	r.TerraformVersion, r.System = cli.Version, cli.Platform
	r.ExecutionReport = filepath.Join(c.RunDirectory, "execution.json")
	started := time.Now().UTC().Format(time.RFC3339Nano)
	r.Consumer = &liveplatform.ConsumerEvidence{Endpoint: c.Endpoint, Scope: c.Scope, Principal: c.ExpectedPrincipal, RunID: c.RunID, StartedAt: started, AttemptStartedAt: started, Limitations: liveplatform.ConsumerLimits}
	if data, err := os.ReadFile(r.ExecutionReport); err == nil {
		var prior liveplatform.Report
		if json.Unmarshal(data, &prior) != nil || prior.Consumer == nil || prior.Consumer.RunID != c.RunID {
			t.Fatal("invalid prior consumer checkpoint")
		}
		r.Consumer.StartedAt = prior.Consumer.StartedAt
		r.Consumer.Resumed = true
	}
	r.Status = "running"
	checkpoint := func() {
		r.CheckedAt = time.Now().UTC().Format(time.RFC3339Nano)
		if err := liveplatform.WritePrivateJSON(r.ExecutionReport, r); err != nil {
			t.Fatal(err)
		}
	}
	checkpoint()
	t.Cleanup(func() {
		if t.Failed() {
			r.Status = "failed"
			_ = liveplatform.WritePrivateJSON(r.ExecutionReport, r)
		}
	})
	binary := buildInstalledProvider(t)
	exerciseMetadataConsumer(t, c, binary, func(name string) {
		for i := range r.Cases {
			if r.Cases[i].Name == name {
				r.Cases[i].Status = "passed"
			}
		}
		checkpoint()
	})
	r.Status = "passed"
	r.Blockers = []string{}
	checkpoint()
	if err := r.ValidateEvidence(c); err != nil {
		t.Fatal(err)
	}
}

func TestAccMockHostedMetadataConsumer(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("acceptance tests require TF_ACC")
	}
	if os.Getenv("SPOKE_TEST_TARGET") != "mock" {
		t.Fatal("explicit synthetic lane required")
	}
	s := mockplatform.NewBucketProfile(t)
	c := liveplatform.Config{Endpoint: s.HTTP.URL, CAFile: s.CAFile, CredentialFile: s.TokenFile, Scope: s.Scope, ExpectedPrincipal: "principal-admin", RunID: "synthetic-consumer", RunDirectory: t.TempDir()}
	count := 0
	exerciseMetadataConsumer(t, c, buildInstalledProvider(t), func(string) { count++ })
	if count != len(liveplatform.HostedCases) || s.BucketCount() != 0 {
		t.Fatal("incomplete synthetic consumer execution")
	}
}
