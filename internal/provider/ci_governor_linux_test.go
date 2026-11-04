//go:build linux

package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

func governorTerraformConfig(f *governorFixture, key, name string, protection bool) string {
	return fmt.Sprintf(`
provider "spoke" {
 endpoint = %q
 organization_id = "example"
 project_id = "operators"
 environment_id = "internal"
 ca_file = %q
 token_file = %q
 timeout_seconds = 3
}
resource "spoke_ci_governor" "test" {
 management_key = %q
 name = %q
 admission_policy_id = "admission"
 budget_policy_id = "budget"
 cache_policy_id = "cache"
 execution_profile_ids = ["linux-x86_64-v1"]
 max_active_jobs = 2
 max_allocated_vcpus = 16
 max_warm_runners = 0
 max_job_seconds = 3600
 max_queue_seconds = 60
 retirement_policy = "drain"
 deletion_protection = %t
}
`, f.server.URL, f.ca, f.token, key, name, protection)
}
func governorTerraformState(t *testing.T, p *consumer) map[string]any {
	t.Helper()
	var state struct {
		Resources []struct {
			Mode, Type string
			Instances  []struct{ Attributes map[string]any }
		}
	}
	if err := json.Unmarshal(p.run("state", "pull"), &state); err != nil {
		t.Fatal(err)
	}
	for _, r := range state.Resources {
		if r.Mode == "managed" && r.Type == "spoke_ci_governor" && len(r.Instances) == 1 {
			return r.Instances[0].Attributes
		}
	}
	return nil
}
func governorEmptyPlan(t *testing.T, p *consumer, f *governorFixture) {
	t.Helper()
	before := f.writes.Load()
	p.run("plan", "-detailed-exitcode", "-input=false", "-no-color")
	if f.writes.Load() != before {
		t.Fatal("planning submitted intent")
	}
}
func governorSecretAudit(t *testing.T, p *consumer, extra ...[]byte) {
	t.Helper()
	data := append(extra, p.run("state", "pull"), p.run("show", "-json", "saved.tfplan"))
	for _, b := range data {
		for token := range governorTokens {
			if bytes.Contains(b, []byte(token)) {
				t.Fatal("credential canary leaked")
			}
		}
	}
	if strings.Contains(strings.Join(p.env, "\n"), "PATH="+filepath.Dir(os.Getenv("TF_ACC_TERRAFORM_PATH"))+":") {
		t.Fatal("compiler path added")
	}
}

func TestAccMockGovernorLifecycle(t *testing.T) {
	_ = mockAcceptance(t)
	binary := buildInstalledProvider(t)
	cases := map[string]func(*testing.T, string){"lifecycle-import-drift-discovery": governorInstalledLifecycle, "identity-fences-and-concurrency": governorInstalledFences, "authority-and-aliases": governorInstalledAuthority, "protected-drain-and-observation": governorInstalledDrain, "schema-bounds-and-confidentiality": governorInstalledSchema}
	raw, err := os.ReadFile("../../testdata/contracts/governor-terraform-cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Lifecycle []string `json:"lifecycle_cases"`
		Crashes   []string `json:"provider_death_cases"`
	}
	if json.Unmarshal(raw, &manifest) != nil {
		t.Fatal("invalid inventory")
	}
	names := make([]string, 0, len(cases))
	for name := range cases {
		names = append(names, name)
	}
	slices.Sort(names)
	slices.Sort(manifest.Lifecycle)
	if !reflect.DeepEqual(names, manifest.Lifecycle) || !reflect.DeepEqual(governorCrashCases(), manifest.Crashes) {
		t.Fatal("missing required governor cases")
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) { cases[name](t, binary) })
	}
}
func governorInstalledLifecycle(t *testing.T, binary string) {
	f := newGovernorFixture(t)
	cfg := func(name string) string { return governorTerraformConfig(f, "stable", name, false) }
	p := newConsumer(t, binary, cfg("a"))
	p.run("validate")
	p.apply()
	state := governorTerraformState(t, p)
	id, rid := state["id"].(string), state["resource_id"].(string)
	if state["readiness"] != "disabled" || state["active_jobs"] != float64(0) {
		t.Fatal("declaration claimed execution", state)
	}
	governorEmptyPlan(t, p, f)
	before := f.writes.Load()
	p.run("refresh", "-input=false", "-no-color")
	p.run("state", "rm", "spoke_ci_governor.test")
	p.run("import", "-input=false", "-no-color", "spoke_ci_governor.test", id)
	if f.writes.Load() != before {
		t.Fatal("read/import mutated")
	}
	governorEmptyPlan(t, p, f)
	for _, name := range []string{"b", "a"} {
		p.config(cfg(name))
		p.apply()
	}
	op, err := f.client(t).Management(context.Background(), "stable")
	if err != nil || op.Sequence != 2 {
		t.Fatal("A B A reused identity", err)
	}
	if err := f.current.Load().Drift(rid); err != nil {
		t.Fatal(err)
	}
	p.run("refresh", "-input=false", "-no-color")
	governorEmptyPlan(t, p, f)
	p.config(cfg("a") + `data "spoke_ci_governor" "exact" { id = spoke_ci_governor.test.id }
data "spoke_ci_governors" "all" {}`)
	before = f.writes.Load()
	p.apply()
	if f.writes.Load() != before {
		t.Fatal("data source submitted work")
	}
	p.config(cfg("a"))
	p.apply()
	p.destroy()
	if governorTerraformState(t, p) != nil {
		t.Fatal("destroy retained governor")
	}
	if _, err := p.attempt("apply", "-auto-approve", "-input=false", "-no-color"); err == nil {
		t.Fatal("retired key reused")
	}
	p.config(governorTerraformConfig(f, "successor", "a", false))
	p.apply()
	if governorTerraformState(t, p)["id"] == id {
		t.Fatal("recreation reused incarnation")
	}
	p.destroy()
}
func governorInstalledFences(t *testing.T, binary string) {
	f := newGovernorFixture(t)
	cfg := func(name string) string { return governorTerraformConfig(f, "stable", name, false) }
	p := newConsumer(t, binary, cfg("a"))
	p.apply()
	id := governorTerraformState(t, p)["resource_id"].(string)
	p.config(cfg("b"))
	p.run("plan", "-out=saved.tfplan", "-input=false", "-no-color")
	before := f.writes.Load()
	if err := f.current.Load().Drift(id); err != nil {
		t.Fatal(err)
	}
	if _, err := p.attempt("apply", "-input=false", "-no-color", "saved.tfplan"); err == nil || f.writes.Load() != before {
		t.Fatal("saved plan rebased revision")
	}
	p.apply()
	g, err := f.client(t).Governor(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	policy := governorTestPolicy("c")
	r := api.GovernorRequest{ManagementKey: "stable", Sequence: 2, Intent: api.GovernorIntent{Kind: "update", TargetID: &id, ExpectedRevision: g.Revision, Policy: policy}}
	if _, err := f.client(t).Prepare(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	before = f.writes.Load()
	p.config(cfg("d"))
	if _, err := p.attempt("plan", "-input=false", "-no-color"); err == nil || f.writes.Load() != before {
		t.Fatal("pending different intent committed")
	}
	p.config(cfg("c"))
	p.apply()
	op, _ := f.client(t).Management(context.Background(), "stable")
	if op.Sequence != 2 {
		t.Fatal("pending request advanced")
	}
	// Two independent Terraform roots plan the same next request. The second is
	// exact recovery; changing its config cannot reinterpret that saved plan.
	clone := newConsumer(t, binary, cfg("c"))
	clone.run("import", "-input=false", "-no-color", "spoke_ci_governor.test", governorCanonical(governorScopeValue, id))
	p.config(cfg("d"))
	clone.config(cfg("d"))
	p.run("plan", "-out=saved.tfplan", "-input=false", "-no-color")
	clone.run("plan", "-out=saved.tfplan", "-input=false", "-no-color")
	p.run("apply", "-input=false", "-no-color", "saved.tfplan")
	clone.run("apply", "-input=false", "-no-color", "saved.tfplan")
	op, _ = f.client(t).Management(context.Background(), "stable")
	if op.Sequence != 3 {
		t.Fatal("identical writer duplicated intent")
	}
	p.destroy()
	for _, conflict := range []bool{false, true} {
		t.Run(fmt.Sprintf("concurrent-conflict-%t", conflict), func(t *testing.T) {
			f := newGovernorFixture(t)
			first := newConsumer(t, binary, governorTerraformConfig(f, "concurrent", "a", false))
			other := "a"
			if conflict {
				other = "b"
			}
			second := newConsumer(t, binary, governorTerraformConfig(f, "concurrent", other, false))
			for _, p := range []*consumer{first, second} {
				p.run("plan", "-out=saved.tfplan", "-input=false", "-no-color")
			}
			type result struct {
				p   *consumer
				err error
				out []byte
			}
			results := make(chan result, 2)
			for _, p := range []*consumer{first, second} {
				go func() {
					out, err := p.attempt("apply", "-input=false", "-no-color", "saved.tfplan")
					results <- result{p, err, out}
				}()
			}
			var winner *consumer
			successes := 0
			for range 2 {
				r := <-results
				if r.err == nil {
					winner = r.p
					successes++
				} else if !conflict {
					t.Fatalf("identical concurrent request failed: %s", r.out)
				}
			}
			want := 2
			if conflict {
				want = 1
			}
			if successes != want {
				t.Fatalf("concurrent successes=%d want=%d", successes, want)
			}
			op, err := f.client(t).Management(context.Background(), "concurrent")
			if err != nil || op.Sequence != 0 || op.Status != "applied" {
				t.Fatal("concurrent request duplicated", err)
			}
			winner.destroy()
		})
	}
}
func governorInstalledAuthority(t *testing.T, binary string) {
	f := newGovernorFixture(t)
	cfg := governorTerraformConfig(f, "stable", "a", false)
	p := newConsumer(t, binary, cfg)
	p.apply()
	id := governorTerraformState(t, p)["id"].(string)
	if err := f.current.Load().Revoke("operator", true); err != nil {
		t.Fatal(err)
	}
	before := f.writes.Load()
	out, err := p.attempt("refresh", "-input=false", "-no-color")
	if err == nil || !strings.Contains(string(out), "forbidden") || governorTerraformState(t, p)["id"] != id || f.writes.Load() != before {
		t.Fatal("denial erased state")
	}
	if err := f.current.Load().Revoke("operator", false); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f.token, []byte("reader-secret-canary"), 0600); err != nil {
		t.Fatal(err)
	}
	p.run("refresh", "-input=false", "-no-color")
	p.config(governorTerraformConfig(f, "stable", "b", false))
	if _, err := p.attempt("apply", "-auto-approve", "-input=false", "-no-color"); err == nil {
		t.Fatal("reader mutated")
	}
	if err := os.WriteFile(f.token, []byte("foreign-secret-canary"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := p.attempt("refresh", "-input=false", "-no-color"); err == nil {
		t.Fatal("foreign owner adopted")
	}
	if err := os.WriteFile(f.token, []byte("governor-secret-canary"), 0600); err != nil {
		t.Fatal(err)
	}
	p.config(cfg)
	p.destroy()
	// Aliases use separate installed provider instances and separate TLS peers.
	a, b := newGovernorFixture(t), newGovernorFixture(t)
	left := strings.Replace(governorTerraformConfig(a, "left", "a", false), `resource "spoke_ci_governor" "test"`, `resource "spoke_ci_governor" "left"`, 1)
	right := strings.Replace(governorTerraformConfig(b, "right", "b", false), "provider \"spoke\" {", "provider \"spoke\" {\n alias = \"other\"", 1)
	right = strings.Replace(right, "resource \"spoke_ci_governor\" \"test\" {", "resource \"spoke_ci_governor\" \"right\" {\n provider = spoke.other", 1)
	aliases := newConsumer(t, binary, left+right)
	aliases.apply()
	for _, pair := range []struct {
		f   *governorFixture
		key string
	}{{a, "left"}, {b, "right"}} {
		op, err := pair.f.client(t).Management(context.Background(), pair.key)
		if err != nil || op.Status != "applied" {
			t.Fatal("alias shared configuration", err)
		}
	}
	aliases.destroy()
}
func governorInstalledDrain(t *testing.T, binary string) {
	f := newGovernorFixture(t)
	p := newConsumer(t, binary, governorTerraformConfig(f, "stable", "a", true))
	p.apply()
	id := governorTerraformState(t, p)["resource_id"].(string)
	if _, err := p.attempt("destroy", "-auto-approve", "-input=false", "-no-color"); err == nil {
		t.Fatal("protected governor deleted")
	}
	p.config(governorTerraformConfig(f, "stable", "a", false))
	p.apply()
	raised := strings.Replace(governorTerraformConfig(f, "stable", "a", false), "max_active_jobs = 2", "max_active_jobs = 3", 1)
	p.config(raised)
	p.apply()
	g, err := f.client(t).Governor(context.Background(), id)
	if err != nil || g.ActiveJobs != 0 || g.AllocatedVCPUs != 0 {
		t.Fatal("raising policy launched work", err)
	}
	if err := f.current.Load().SeedUsage(id, 1, 4, 1); err != nil {
		t.Fatal(err)
	}
	lowered := strings.Replace(strings.Replace(raised, "max_active_jobs = 3", "max_active_jobs = 1", 1), "max_allocated_vcpus = 16", "max_allocated_vcpus = 1", 1)
	p.config(lowered)
	p.apply()
	g, err = f.client(t).Governor(context.Background(), id)
	if err != nil || g.ActiveJobs != 1 || g.AllocatedVCPUs != 4 || g.UnsettledUsage != 1 {
		t.Fatal("lowering ceilings erased outstanding work", err)
	}
	before := f.current.Load().Snapshot()
	out, err := p.attempt("destroy", "-auto-approve", "-input=false", "-no-color")
	if err == nil || governorTerraformState(t, p) == nil {
		t.Fatal("blocked drain erased state")
	}
	op, err := f.client(t).Management(context.Background(), "stable")
	if err != nil || op.Status != "accepted" {
		t.Fatal("drain lost original request", err)
	}
	writes := f.writes.Load()
	p.run("refresh", "-input=false", "-no-color")
	if governorTerraformState(t, p) == nil {
		t.Fatal("pending retirement erased state")
	}
	if f.writes.Load() != writes {
		t.Fatal("refresh settled drain")
	}
	pending, _ := f.client(t).Management(context.Background(), "stable")
	if pending.ID != op.ID || pending.Status != "accepted" {
		t.Fatal("read settled pending work")
	}
	if err := f.current.Load().SeedUsage(id, 0, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := f.current.Load().Settle(); err != nil {
		t.Fatal(err)
	}
	p.destroy()
	terminal, _ := f.client(t).Management(context.Background(), "stable")
	if terminal.ID != op.ID || terminal.Sequence != op.Sequence || terminal.Status != "applied" {
		t.Fatal("drain allocated successor")
	}
	var pre, post struct{ References any }
	if json.Unmarshal(before, &pre) != nil || json.Unmarshal(f.current.Load().Snapshot(), &post) != nil || !reflect.DeepEqual(pre.References, post.References) {
		t.Fatal("retirement erased independent references")
	}
	if bytes.Contains(out, []byte("secret-canary")) {
		t.Fatal("drain diagnostic leaked credential")
	}
}
func governorInstalledSchema(t *testing.T, binary string) {
	f := newGovernorFixture(t)
	cfg := governorTerraformConfig(f, "stable", "a", false)
	p := newConsumer(t, binary, cfg)
	for _, bad := range []string{strings.Replace(cfg, "max_warm_runners = 0", "max_warm_runners = 1", 1), strings.Replace(cfg, "max_active_jobs = 2", "max_active_jobs = null", 1), strings.Replace(cfg, "max_active_jobs = 2", "max_active_jobs = 257", 1), strings.Replace(cfg, "linux-x86_64-v1", "unsupported", 1)} {
		p.config(bad)
		if _, err := p.attempt("validate", "-no-color"); err == nil {
			t.Fatal("unsupported policy validated")
		}
	}
	if f.writes.Load() != 0 {
		t.Fatal("validation dispatched")
	}
	// Terraform's built-in data resource resolves this initially unknown input
	// without installing a second provider or changing the configured key.
	unknown := strings.Replace(cfg, `name = "a"`, `name = terraform_data.policy.output`, 1) + `resource "terraform_data" "policy" { input = "a" }`
	p.config(unknown)
	p.run("plan", "-out=saved.tfplan", "-input=false", "-no-color")
	out := p.run("apply", "-input=false", "-no-color", "saved.tfplan")
	governorSecretAudit(t, p, out)
	governorEmptyPlan(t, p, f)
	p.destroy()
}

type governorCrashEvent struct {
	Request   api.GovernorRequest
	Operation api.GovernorOperation
	Ledger    []byte
}
type governorBarrier struct {
	reached           chan governorCrashEvent
	release, finished chan struct{}
}

func (f *governorFixture) armGovernor(stage string) *governorBarrier {
	b := &governorBarrier{make(chan governorCrashEvent, 1), make(chan struct{}), make(chan struct{})}
	var armed atomic.Bool
	if stage == "after_accept" {
		f.current.Load().SetHold(true)
	}
	f.hook.Store(&governorHTTPHook{serve: func(w http.ResponseWriter, r *http.Request, next http.Handler) {
		match := r.Method == "PUT" && (stage == "before_prepare" || stage == "after_prepare") || r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/commit") && (stage == "after_accept" || stage == "after_complete")
		if !match || !armed.CompareAndSwap(false, true) {
			next.ServeHTTP(w, r)
			return
		}
		defer close(b.finished)
		var event governorCrashEvent
		if r.Method == "PUT" {
			raw, _ := io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewReader(raw))
			_ = json.Unmarshal(raw, &event.Request.Intent)
			parts := strings.Split(strings.TrimPrefix(r.URL.Path, governorTestScope+"/requests/"), "/")
			if len(parts) == 2 {
				event.Request.ManagementKey = parts[0]
				event.Request.Sequence, _ = strconv.ParseUint(parts[1], 10, 64)
			}
		}
		rec := httptest.NewRecorder()
		if stage != "before_prepare" {
			next.ServeHTTP(rec, r)
			_ = json.Unmarshal(rec.Body.Bytes(), &event.Operation)
			event.Request = event.Operation.Request()
		}
		event.Ledger = f.current.Load().Snapshot()
		b.reached <- event
		select {
		case <-b.release:
		case <-r.Context().Done():
			return
		}
		// A before-prepare death must not forward the cancelled mutation.
		if stage == "before_prepare" {
			return
		}
		for k, vs := range rec.Header() {
			w.Header()[k] = vs
		}
		w.WriteHeader(rec.Code)
		_, _ = w.Write(rec.Body.Bytes())
	}})
	return b
}
func governorCrashCases() []string {
	var out []string
	for _, kind := range []string{"create", "update", "retire"} {
		for _, stage := range []string{"before_prepare", "after_prepare", "after_accept", "after_complete"} {
			out = append(out, kind+"/"+stage)
		}
	}
	return out
}
func TestAccMockGovernorProcessRecovery(t *testing.T) {
	_ = mockAcceptance(t)
	binary := buildInstalledProvider(t)
	for index, name := range governorCrashCases() {
		t.Run(name, func(t *testing.T) {
			kind, stage, _ := strings.Cut(name, "/")
			f := newGovernorFixture(t)
			cfg := func(name string) string { return governorTerraformConfig(f, "stable", name, false) }
			p := newConsumer(t, binary, cfg("a"))
			wantSeq := uint64(0)
			wantRevision := uint64(1)
			if kind != "create" {
				p.apply()
				wantSeq = 1
			}
			if kind == "update" {
				p.config(cfg("b"))
				wantRevision = 2
			}
			planArgs := []string{"plan", "-out=saved.tfplan", "-input=false", "-no-color"}
			if kind == "retire" {
				planArgs = append(planArgs, "-destroy")
			}
			p.run(planArgs...)
			priorState, stateErr := os.ReadFile(filepath.Join(p.dir, "terraform.tfstate"))
			if stateErr != nil && !os.IsNotExist(stateErr) {
				t.Fatal(stateErr)
			}
			savedPlan, err := os.ReadFile(filepath.Join(p.dir, "saved.tfplan"))
			if err != nil {
				t.Fatal(err)
			}
			b := f.armGovernor(stage)
			event := killProviderAt(p, b.reached, "apply", "-input=false", "-no-color", "saved.tfplan")
			close(b.release)
			select {
			case <-b.finished:
			case <-time.After(5 * time.Second):
				t.Fatal("barrier handler did not exit")
			}
			f.hook.Store(nil)
			if event.Request.ManagementKey != "stable" || event.Request.Sequence != wantSeq || event.Request.Intent.Kind != kind {
				t.Fatalf("barrier lost selected tuple: %+v", event.Request)
			}
			if stage != "before_prepare" {
				wantStatus := map[string]string{"after_prepare": "prepared", "after_accept": "accepted", "after_complete": "applied"}[stage]
				if event.Operation.ID == "" || event.Operation.Status != wantStatus {
					t.Fatalf("barrier was not durable: %+v", event.Operation)
				}
			}
			// Reload the persisted peer independently, keeping TLS origin and trust.
			f.reload(t)
			if stage == "after_accept" {
				if err := f.current.Load().Settle(); err != nil {
					t.Fatal(err)
				}
			}
			if index%2 == 0 {
				// Terraform itself can save a new state serial while reporting a
				// provider death. Replay the original plan with its original state
				// in a separate consumer, never overwrite the failed caller's state.
				name := "a"
				if kind == "update" {
					name = "b"
				}
				p = newConsumer(t, binary, cfg(name))
				if stateErr == nil {
					if err := os.WriteFile(filepath.Join(p.dir, "terraform.tfstate"), priorState, 0600); err != nil {
						t.Fatal(err)
					}
				}
				if err := os.WriteFile(filepath.Join(p.dir, "saved.tfplan"), savedPlan, 0600); err != nil {
					t.Fatal(err)
				}
				p.run("apply", "-input=false", "-no-color", "saved.tfplan")
			} else if kind == "retire" {
				p.destroy()
			} else {
				p.apply()
			}
			op, err := f.client(t).Management(context.Background(), "stable")
			if err != nil || op.Sequence != wantSeq || op.Status != "applied" || stage != "before_prepare" && (op.ID != event.Operation.ID || op.Fingerprint != event.Operation.Fingerprint) {
				t.Fatalf("recovery changed original receipt: %+v %v", op, err)
			}
			if !reflect.DeepEqual(op.Request(), event.Request) {
				t.Fatal("recovery changed selected intent")
			}
			var ledger struct {
				Keys      map[string]struct{ Requests map[string]json.RawMessage }
				Governors map[string]json.RawMessage
			}
			if json.Unmarshal(f.current.Load().Snapshot(), &ledger) != nil || len(ledger.Keys["stable"].Requests) != int(wantSeq)+1 {
				t.Fatal("duplicate retained mutation")
			}
			if kind != "retire" {
				g, err := f.client(t).Governor(context.Background(), *op.ResourceID)
				if err != nil || g.Revision != wantRevision {
					t.Fatal("duplicate policy effect", err)
				}
				governorEmptyPlan(t, p, f)
				p.destroy()
			} else if len(ledger.Governors) != 0 || governorTerraformState(t, p) != nil {
				t.Fatal("retirement not authoritative")
			}
			t.Logf("retained %s sequence=%d fingerprint=%s; peer ledger independently reloaded; replay=%s", op.ID, op.Sequence, op.Fingerprint, map[bool]string{true: "saved plan", false: "fresh plan"}[index%2 == 0])
		})
	}
}
