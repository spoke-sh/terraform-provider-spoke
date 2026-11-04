package platformclient_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	pc "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
	peer "github.com/spoke-sh/terraform-provider-spoke/internal/testutil/mockplatform/governor"
)

const govBase = "/v1/ci/organizations/example/projects/operators/environments/internal"

var govScope = pc.Scope{OrganizationID: "example", ProjectID: "operators", EnvironmentID: "internal"}
var govTokens = map[string]peer.Identity{
	"governor-secret-canary": {Principal: "operator-one", Operator: true},
	"other-secret-canary":    {Principal: "operator-two", Operator: true},
	"job-secret-canary":      {Principal: "operator-one", Operator: false},
}

type govFixture struct {
	peer   *peer.Server
	http   *httptest.Server
	ledger string
	schema []byte
}

func govPolicy() pc.GovernorPolicy {
	return pc.GovernorPolicy{Name: "internal-ci", AdmissionPolicyID: "internal-entitlements", BudgetPolicyID: "pilot-budget", CachePolicyID: "tenant-cache", ExecutionProfileIDs: []string{"linux-x86_64-v1"}, MaxActiveJobs: 2, MaxAllocatedVCPUs: 16, MaxJobSeconds: 7200, MaxQueueSeconds: 900, RetirementPolicy: "drain"}
}
func govCreate(key string) pc.GovernorRequest {
	p := govPolicy()
	return pc.GovernorRequest{ManagementKey: key, Intent: pc.GovernorIntent{Kind: "create", Policy: &p}}
}
func govChange(key, id, kind string, seq, rev uint64, p *pc.GovernorPolicy) pc.GovernorRequest {
	return pc.GovernorRequest{ManagementKey: key, Sequence: seq, Intent: pc.GovernorIntent{Kind: kind, TargetID: &id, ExpectedRevision: rev, Policy: p}}
}
func newGovFixture(t *testing.T) *govFixture {
	t.Helper()
	schema, err := os.ReadFile("governor-contract/schema.json")
	if err != nil {
		t.Fatal(err)
	}
	f := &govFixture{ledger: filepath.Join(t.TempDir(), "ledger.json"), schema: schema}
	f.restart(t)
	for kind, id := range map[string]string{"admission_policy_id": "internal-entitlements", "budget_policy_id": "pilot-budget", "cache_policy_id": "tenant-cache"} {
		if err := f.peer.AllowReference(kind, id, "operator-one", "operator-two"); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { f.http.Close() })
	return f
}
func (f *govFixture) restart(t *testing.T) {
	t.Helper()
	if f.http != nil {
		f.http.Close()
	}
	var err error
	f.peer, err = peer.New(f.ledger, govBase, f.schema, govTokens)
	if err != nil {
		t.Fatal(err)
	}
	f.http = httptest.NewTLSServer(f.peer)
}
func (f *govFixture) client(t *testing.T, token string) *pc.GovernorClient {
	t.Helper()
	c, err := pc.NewGovernor(pc.Config{Endpoint: f.http.URL, Scope: govScope, Token: token, HTTPClient: f.http.Client(), PollInterval: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// Once the accepted operation reaches its first observational wait, keep it
// pending until the caller's deadline. This avoids scheduler-sensitive polling
// counts or requiring admission to finish inside a tiny timeout.
func (f *govFixture) deadlineClient(t *testing.T) *pc.GovernorClient {
	t.Helper()
	c, err := pc.NewGovernor(pc.Config{Endpoint: f.http.URL, Scope: govScope, Token: "governor-secret-canary", HTTPClient: f.http.Client(), Wait: func(ctx context.Context, _ time.Duration) error { <-ctx.Done(); return ctx.Err() }})
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func govApply(t *testing.T, c *pc.GovernorClient, r pc.GovernorRequest) pc.GovernorOperation {
	t.Helper()
	op, err := c.Apply(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	if op.Status != "applied" || op.ResourceID == nil {
		t.Fatal("incomplete policy application")
	}
	return op
}
func govError(t *testing.T, err error, code string) {
	t.Helper()
	var e *pc.GovernorError
	if !errors.As(err, &e) || e.Code != code {
		t.Fatalf("expected %s, got %v", code, err)
	}
}

func TestGovernorConformance(t *testing.T) {
	cases := map[string]func(*testing.T){
		"lifecycle-distinct-intent":                  govLifecycle,
		"lost-replies-original-identity":             govLostReplies,
		"client-death-and-service-restart":           govProcessRecovery,
		"accepted-deadline-and-explicit-resume":      govDeadline,
		"observational-bounded-discovery":            govDiscovery,
		"authorization-ownership-and-revision":       govAuthority,
		"protected-drain-and-retained-tombstone":     govDrain,
		"abort-and-receipt-expiry":                   govAbort,
		"malformed-and-incompatible-peer":            govMalformed,
		"collection-integrity-and-no-partial-result": govCollections,
		"receipt-identity-and-absence":               govReceiptIdentity,
		"tls-redirect-and-secret-boundaries":         govTransport,
	}
	raw, err := os.ReadFile("../../testdata/contracts/governor-client-cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Cases []string `json:"required_cases"`
	}
	if json.Unmarshal(raw, &manifest) != nil {
		t.Fatal("invalid manifest")
	}
	names := make([]string, 0, len(cases))
	for name := range cases {
		names = append(names, name)
	}
	slices.Sort(names)
	want := slices.Clone(manifest.Cases)
	slices.Sort(want)
	if !reflect.DeepEqual(names, want) {
		t.Fatal("conformance inventory drift")
	}
	for _, name := range names {
		t.Run(name, cases[name])
	}
}

func govLifecycle(t *testing.T) {
	f := newGovFixture(t)
	c := f.client(t, "governor-secret-canary")
	r := govCreate("stable")
	first := govApply(t, c, r)
	id := *first.ResourceID
	if err := f.peer.SeedUsage(id, 2, 16, 0); err != nil {
		t.Fatal(err)
	}
	p := govPolicy()
	p.MaxActiveJobs = 1
	second := govApply(t, c, govChange("stable", id, "update", 1, 1, &p))
	g, err := c.Governor(context.Background(), id)
	if err != nil || g.Policy.MaxActiveJobs != 1 || g.ActiveJobs != 2 || g.AllocatedVCPUs != 16 || g.Revision != 2 || g.ActiveRevision != 0 {
		t.Fatal("policy overwrite or revision failure", g, err)
	}
	p = govPolicy()
	third := govApply(t, c, govChange("stable", id, "update", 2, 2, &p))
	if first.ID == second.ID || first.ID == third.ID || first.Fingerprint == third.Fingerprint {
		t.Fatal("return to A replayed an earlier intent")
	}
	replay := govApply(t, c, r)
	if replay.ID != first.ID {
		t.Fatal("exact replay changed original identity")
	}
	g, err = c.Governor(context.Background(), id)
	if err != nil || g.Revision != 3 {
		t.Fatal("replay reapplied old policy")
	}
}

func govRecoveryRequest(t *testing.T, f *govFixture, kind string) (pc.GovernorRequest, int) {
	t.Helper()
	if kind == "create" {
		return govCreate("stable"), 1
	}
	first := govApply(t, f.client(t, "governor-secret-canary"), govCreate("stable"))
	if kind == "retire" {
		return govChange("stable", *first.ResourceID, kind, 1, 1, nil), 0
	}
	p := govPolicy()
	p.MaxActiveJobs = 4
	return govChange("stable", *first.ResourceID, kind, 1, 1, &p), 1
}

func govLostReplies(t *testing.T) {
	for _, kind := range []string{"create", "update", "retire"} {
		for _, stage := range []string{"prepare", "commit"} {
			for _, when := range []string{"before", "after"} {
				t.Run(kind+"/"+stage+"-"+when, func(t *testing.T) {
					f := newGovFixture(t)
					request, count := govRecoveryRequest(t, f, kind)
					f.peer.Drop(stage, when, 1)
					c := f.client(t, "governor-secret-canary")
					op := govApply(t, c, request)
					f.restart(t)
					c = f.client(t, "governor-secret-canary")
					replayed := govApply(t, c, request)
					all, err := c.Governors(context.Background())
					if err != nil || len(all) != count || replayed.ID != op.ID || *replayed.ResourceID != *op.ResourceID {
						t.Fatal("uncertain retry allocated a new identity", err)
					}
					if kind == "update" && (all[0].Revision != 2 || all[0].Policy.MaxActiveJobs != 4) {
						t.Fatal("uncertain update lost or repeated policy")
					}
				})
			}
		}
	}
}

type govChildConfig struct {
	Endpoint string
	Phase    string
	Request  pc.GovernorRequest
}

func TestGovernorClientHelper(t *testing.T) {
	path := os.Getenv("SPOKE_GOVERNOR_CHILD")
	if path == "" {
		t.Skip("subprocess helper")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var cfg govChildConfig
	if json.Unmarshal(raw, &cfg) != nil {
		t.Fatal("config")
	}
	// The loopback peer uses the same standard httptest certificate as the
	// temporary trust source. No network request is made to that trust source.
	trust := httptest.NewTLSServer(http.NotFoundHandler())
	defer trust.Close()
	c, err := pc.NewGovernor(pc.Config{Endpoint: cfg.Endpoint, Scope: govScope, Token: "governor-secret-canary", HTTPClient: trust.Client(), PollInterval: 5 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Phase == "after-prepare" {
		_, err = c.Prepare(context.Background(), cfg.Request)
	} else if cfg.Phase == "after-completion" || cfg.Phase == "after-acceptance" {
		_, err = c.Apply(context.Background(), cfg.Request)
	}
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("GOVERNOR-READY")
	select {}
}
func govProcessRecovery(t *testing.T) {
	for _, kind := range []string{"create", "update", "retire"} {
		for _, phase := range []string{"before-dispatch", "after-prepare", "after-acceptance", "after-completion"} {
			t.Run(kind+"/"+phase, func(t *testing.T) {
				f := newGovFixture(t)
				request, count := govRecoveryRequest(t, f, kind)
				if phase == "after-acceptance" {
					f.peer.SetHold(true)
				}
				path := filepath.Join(t.TempDir(), "child.json")
				raw, _ := json.Marshal(govChildConfig{Endpoint: f.http.URL, Phase: phase, Request: request})
				if err := os.WriteFile(path, raw, 0600); err != nil {
					t.Fatal(err)
				}
				cmd := exec.Command(os.Args[0], "-test.run=^TestGovernorClientHelper$", "-test.timeout=20s")
				cmd.Env = append(os.Environ(), "SPOKE_GOVERNOR_CHILD="+path)
				stdout, err := cmd.StdoutPipe()
				if err != nil {
					t.Fatal(err)
				}
				var stderr bytes.Buffer
				cmd.Stderr = &stderr
				if err = cmd.Start(); err != nil {
					t.Fatal(err)
				}
				defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
				ready := make(chan struct{})
				go func() {
					scanner := bufio.NewScanner(stdout)
					for scanner.Scan() {
						if scanner.Text() == "GOVERNOR-READY" {
							close(ready)
							return
						}
					}
				}()
				var prior pc.GovernorOperation
				if phase == "after-acceptance" {
					c := f.client(t, "governor-secret-canary")
					deadline := time.Now().Add(5 * time.Second)
					for time.Now().Before(deadline) {
						prior, err = c.Request(context.Background(), "stable", request.Sequence)
						if err == nil && prior.Status == "accepted" {
							break
						}
						time.Sleep(5 * time.Millisecond)
					}
					if prior.Status != "accepted" {
						t.Fatal("child did not reach durable acceptance")
					}
				} else {
					select {
					case <-ready:
					case <-time.After(5 * time.Second):
						t.Fatal("child did not reach barrier")
					}
					if phase != "before-dispatch" {
						prior, err = f.client(t, "governor-secret-canary").Request(context.Background(), "stable", request.Sequence)
						if err != nil {
							t.Fatal(err)
						}
					}
				}
				if err = cmd.Process.Kill(); err != nil {
					t.Fatal(err)
				}
				if err = cmd.Wait(); err == nil {
					t.Fatal("client was not killed")
				}
				f.restart(t)
				if err = f.peer.Settle(); err != nil {
					t.Fatal(err)
				}
				c := f.client(t, "governor-secret-canary")
				var op pc.GovernorOperation
				if phase == "before-dispatch" {
					op = govApply(t, c, request)
				} else {
					op, err = c.Resume(context.Background(), "stable")
					if err != nil {
						t.Fatal(err)
					}
				}
				if prior.ID != "" && op.ID != prior.ID {
					t.Fatal("restart changed retained acceptance")
				}
				all, err := c.Governors(context.Background())
				if err != nil || len(all) != count {
					t.Fatal("duplicate/missing governor after process loss")
				}
				if kind == "update" && (all[0].Revision != 2 || all[0].Policy.MaxActiveJobs != 4) {
					t.Fatal("process loss changed policy or applied it twice")
				}
			})
		}
	}
}

func govDeadline(t *testing.T) {
	f := newGovFixture(t)
	f.peer.SetHold(true)
	c := f.client(t, "governor-secret-canary")
	if _, err := c.Prepare(context.Background(), govCreate("stable")); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	op, err := f.deadlineClient(t).Apply(ctx, govCreate("stable"))
	if !errors.Is(err, context.DeadlineExceeded) || op.Status != "accepted" {
		t.Fatal("timeout lost accepted identity", op.Status, err)
	}
	latest, err := c.Management(context.Background(), "stable")
	if err != nil || latest.ID != op.ID || latest.Status != "accepted" {
		t.Fatal("timeout aborted work")
	}
	if err = f.peer.Settle(); err != nil {
		t.Fatal(err)
	}
	recovered, err := c.Resume(context.Background(), "stable")
	if err != nil || recovered.ID != op.ID {
		t.Fatal("resume changed request", err)
	}
}

func govDiscovery(t *testing.T) {
	f := newGovFixture(t)
	c := f.client(t, "governor-secret-canary")
	for i := 0; i < 10; i++ {
		r := govCreate(fmt.Sprintf("key-%02d", i))
		r.Intent.Policy.AdmissionPolicyID = fmt.Sprintf("entitlement-%02d", i)
		if err := f.peer.AllowReference("admission_policy_id", r.Intent.Policy.AdmissionPolicyID, "operator-one"); err != nil {
			t.Fatal(err)
		}
		govApply(t, c, r)
	}
	for i := 0; i < 10; i++ {
		r := govCreate(fmt.Sprintf("pending-%02d", i))
		r.Intent.Policy.AdmissionPolicyID = fmt.Sprintf("pending-entitlement-%02d", i)
		if err := f.peer.AllowReference("admission_policy_id", r.Intent.Policy.AdmissionPolicyID, "operator-one"); err != nil {
			t.Fatal(err)
		}
		if _, err := c.Prepare(context.Background(), r); err != nil {
			t.Fatal(err)
		}
	}
	before := f.peer.Snapshot()
	all, err := c.Governors(context.Background())
	if err != nil || len(all) != 10 {
		t.Fatal("governor collection", len(all), err)
	}
	pending, err := c.Pending(context.Background())
	if err != nil || len(pending) != 10 {
		t.Fatal("pending collection", len(pending), err)
	}
	for _, g := range all {
		if _, err = c.Governor(context.Background(), g.ID); err != nil {
			t.Fatal(err)
		}
	}
	for _, op := range pending {
		if _, err = c.Request(context.Background(), op.ManagementKey, op.Sequence); err != nil {
			t.Fatal(err)
		}
	}
	other := f.client(t, "other-secret-canary")
	foreign, err := other.Governors(context.Background())
	if err != nil || len(foreign) != 0 {
		t.Fatal("foreign listing disclosure")
	}
	if !bytes.Equal(before, f.peer.Snapshot()) {
		t.Fatal("reads advanced accepted work")
	}
}

func govAuthority(t *testing.T) {
	f := newGovFixture(t)
	c := f.client(t, "governor-secret-canary")
	first := govApply(t, c, govCreate("stable"))
	id := *first.ResourceID
	other := f.client(t, "other-secret-canary")
	_, err := other.Management(context.Background(), "stable")
	govError(t, err, "forbidden")
	_, err = other.Governor(context.Background(), id)
	govError(t, err, "forbidden")
	job := f.client(t, "job-secret-canary")
	_, err = job.Prepare(context.Background(), govCreate("job"))
	govError(t, err, "forbidden")
	_, err = c.Prepare(context.Background(), govCreate("overlap"))
	govError(t, err, "ownership_conflict")
	foreign := govCreate("foreign-policy")
	foreign.Intent.Policy.AdmissionPolicyID = "foreign-admission"
	if err := f.peer.AllowReference("admission_policy_id", "foreign-admission", "operator-two"); err != nil {
		t.Fatal(err)
	}
	_, err = c.Prepare(context.Background(), foreign)
	govError(t, err, "forbidden")
	refChange := govCreate("revoked-reference")
	refChange.Intent.Policy.AdmissionPolicyID = "reference-entitlement"
	if err = f.peer.AllowReference("admission_policy_id", "reference-entitlement", "operator-one"); err != nil {
		t.Fatal(err)
	}
	prepared, err := c.Prepare(context.Background(), refChange)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.peer.AllowReference("budget_policy_id", "pilot-budget", "operator-two"); err != nil {
		t.Fatal(err)
	}
	refusedReference, err := c.Apply(context.Background(), refChange)
	if err == nil || refusedReference.ID != prepared.ID || refusedReference.Reason != "authority_revoked" {
		t.Fatal("changed reference authority was not fenced")
	}
	if err = f.peer.AllowReference("budget_policy_id", "pilot-budget", "operator-one", "operator-two"); err != nil {
		t.Fatal(err)
	}
	if err = f.peer.Drift(id); err != nil {
		t.Fatal(err)
	}
	p := govPolicy()
	op, err := c.Apply(context.Background(), govChange("stable", id, "update", 1, 1, &p))
	var outcome *pc.GovernorOutcomeError
	if !errors.As(err, &outcome) || op.Status != "refused" || op.Reason != "revision_conflict" {
		t.Fatal("stale revision did not remain refused", err)
	}
	_, err = c.Prepare(context.Background(), govChange("stable", id, "update", 3, 2, &p))
	govError(t, err, "sequence_conflict")
	p.AdmissionPolicyID = "different-entitlement"
	_, err = c.Prepare(context.Background(), govChange("stable", id, "update", 2, 2, &p))
	govError(t, err, "unsupported_capability")
	// The fixture's explicit settlement control models an independent cleanup
	// owner. This is synthetic evidence, not native revocation qualification.
	blocked := govCreate("revoked-create")
	blocked.Intent.Policy.AdmissionPolicyID = "revoked-entitlement"
	if err = f.peer.AllowReference("admission_policy_id", "revoked-entitlement", "operator-one"); err != nil {
		t.Fatal(err)
	}
	f.peer.SetHold(true)
	if _, err = c.Prepare(context.Background(), blocked); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	accepted, err := f.deadlineClient(t).Apply(ctx, blocked)
	if !errors.Is(err, context.DeadlineExceeded) || accepted.Status != "accepted" {
		t.Fatal("missing accepted revocation fixture", err)
	}
	if err = f.peer.Revoke("operator-one", true); err != nil {
		t.Fatal(err)
	}
	_, err = c.Management(context.Background(), "revoked-create")
	govError(t, err, "forbidden")
	if err = f.peer.Settle(); err != nil {
		t.Fatal(err)
	}
	if err = f.peer.Revoke("operator-one", false); err != nil {
		t.Fatal(err)
	}
	refused, err := c.Management(context.Background(), "revoked-create")
	if err != nil || refused.ID != accepted.ID || refused.Status != "refused" || refused.Reason != "authority_revoked" {
		t.Fatal("revocation lost original receipt", err)
	}
}

func govDrain(t *testing.T) {
	f := newGovFixture(t)
	c := f.client(t, "governor-secret-canary")
	r := govCreate("stable")
	r.Intent.Policy.DeletionProtection = true
	first := govApply(t, c, r)
	id := *first.ResourceID
	op, err := c.Apply(context.Background(), govChange("stable", id, "retire", 1, 1, nil))
	var outcome *pc.GovernorOutcomeError
	if !errors.As(err, &outcome) || op.Reason != "deletion_protected" {
		t.Fatal("protected retire", err)
	}
	p := govPolicy()
	govApply(t, c, govChange("stable", id, "update", 2, 1, &p))
	if err = f.peer.SeedUsage(id, 2, 16, 1); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	retire := govChange("stable", id, "retire", 3, 2, nil)
	if _, err = c.Prepare(context.Background(), retire); err != nil {
		t.Fatal(err)
	}
	op, err = f.deadlineClient(t).Apply(ctx, retire)
	if !errors.Is(err, context.DeadlineExceeded) || op.Reason != "draining" {
		t.Fatal("unknown usage retired governor", err)
	}
	g, err := c.Governor(context.Background(), id)
	if err != nil || g.State != "draining" || g.Policy.CachePolicyID != "tenant-cache" || g.UnsettledUsage != 1 {
		t.Fatal("drain lost policy or usage")
	}
	if err = f.peer.SeedUsage(id, 0, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err = f.peer.Settle(); err != nil {
		t.Fatal(err)
	}
	done := govApply(t, c, retire)
	if done.ID != op.ID {
		t.Fatal("retirement identity changed")
	}
	_, err = c.Governor(context.Background(), id)
	govError(t, err, "resource_not_found")
	_, err = f.client(t, "other-secret-canary").Governor(context.Background(), id)
	govError(t, err, "forbidden")
	policyReceipt, err := c.Request(context.Background(), "stable", 2)
	if err != nil || policyReceipt.Intent.Policy.CachePolicyID != "tenant-cache" {
		t.Fatal("retirement discarded independently owned cache reference", err)
	}
	_, err = c.Prepare(context.Background(), pc.GovernorRequest{ManagementKey: "stable", Sequence: 4, Intent: r.Intent})
	govError(t, err, "sequence_conflict")
	newOp := govApply(t, c, govCreate("new-incarnation"))
	if *newOp.ResourceID == id {
		t.Fatal("recreated identity recycled")
	}
	old, err := c.Request(context.Background(), "stable", 3)
	if err != nil || old.Status != "applied" {
		t.Fatal("tombstone lost receipt")
	}
}

func govAbort(t *testing.T) {
	f := newGovFixture(t)
	c := f.client(t, "governor-secret-canary")
	r := govCreate("stable")
	op, err := c.Prepare(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.peer.Expire("stable", 0); err == nil {
		t.Fatal("pending expired")
	}
	f.peer.Drop("abort", "after", 1)
	done, err := c.Abort(context.Background(), r)
	if err != nil || done.Status != "aborted" || done.ID != op.ID {
		t.Fatal("abort recovery", err)
	}
	if err = f.peer.Expire("stable", 0); err != nil {
		t.Fatal(err)
	}
	f.restart(t)
	c = f.client(t, "governor-secret-canary")
	_, err = c.Apply(context.Background(), r)
	govError(t, err, "receipt_expired")
	_, err = c.Abort(context.Background(), govCreate("absent"))
	govError(t, err, "request_not_found")
	if _, err = c.Management(context.Background(), "absent"); err == nil {
		t.Fatal("abort created request")
	}
}

// transform uses a real TLS peer and alters only selected response messages.
func govTransform(t *testing.T, f *govFixture, change func(*http.Request, *httptest.ResponseRecorder)) *pc.GovernorClient {
	t.Helper()
	h := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := httptest.NewRecorder()
		f.peer.ServeHTTP(rec, r)
		change(r, rec)
		for key, values := range rec.Header() {
			w.Header()[key] = values
		}
		w.WriteHeader(rec.Code)
		_, _ = w.Write(rec.Body.Bytes())
	}))
	t.Cleanup(h.Close)
	c, err := pc.NewGovernor(pc.Config{Endpoint: h.URL, Scope: govScope, Token: "governor-secret-canary", HTTPClient: h.Client(), PollInterval: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func govObject(rec *httptest.ResponseRecorder, change func(map[string]any)) {
	var v map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &v)
	change(v)
	rec.Body.Reset()
	_ = json.NewEncoder(rec.Body).Encode(v)
}

func govMalformed(t *testing.T) {
	changes := map[string]func(*httptest.ResponseRecorder){
		"profile":           func(r *httptest.ResponseRecorder) { r.Header().Set("X-Spoke-Management-Profile", "mock-v5") },
		"duplicate-version": func(r *httptest.ResponseRecorder) { r.Header().Add("X-Spoke-API-Version", pc.GovernorVersion) },
		"oversized": func(r *httptest.ResponseRecorder) {
			r.Body.Reset()
			_, _ = r.Body.WriteString(strings.Repeat(" ", 131073))
		},
		"trailing-json": func(r *httptest.ResponseRecorder) { _, _ = r.Body.WriteString(" {}") },
		"unknown-field": func(r *httptest.ResponseRecorder) {
			govObject(r, func(v map[string]any) { v["secret"] = "governor-secret-canary" })
		},
		"missing-required-capability": func(r *httptest.ResponseRecorder) {
			govObject(r, func(v map[string]any) { delete(v["capabilities"].(map[string]any), "prepare_commit") })
		},
		"unqualified-execution": func(r *httptest.ResponseRecorder) {
			govObject(r, func(v map[string]any) { v["capabilities"].(map[string]any)["execution"] = true })
		},
		"scope": func(r *httptest.ResponseRecorder) {
			govObject(r, func(v map[string]any) { v["scope"] = strings.Replace(govBase, "internal", "foreign", 1) })
		},
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			f := newGovFixture(t)
			c := govTransform(t, f, func(r *http.Request, rec *httptest.ResponseRecorder) {
				if strings.HasSuffix(r.URL.Path, "/capabilities") {
					change(rec)
				}
			})
			before := f.peer.Snapshot()
			_, err := c.Apply(context.Background(), govCreate("stable"))
			if err == nil || strings.Contains(err.Error(), "governor-secret-canary") || !bytes.Equal(before, f.peer.Snapshot()) {
				t.Fatal("unsafe capability caused mutation or disclosure", err)
			}
		})
	}
	for _, cfg := range []pc.Config{{Endpoint: "http://example.invalid", Scope: govScope, Token: "synthetic"}, {Endpoint: "https://example.invalid", Scope: govScope, Token: "synthetic", ManagementProfile: pc.PublicationProfile}, {Endpoint: "https://example.invalid", Scope: pc.Scope{OrganizationID: "example"}, Token: "synthetic"}} {
		if _, err := pc.NewGovernor(cfg); err == nil {
			t.Fatal("invalid configuration")
		}
	}
}

func govCollections(t *testing.T) {
	changes := map[string]func(*http.Request, *httptest.ResponseRecorder){
		"duplicate-item": func(_ *http.Request, rec *httptest.ResponseRecorder) {
			govObject(rec, func(v map[string]any) { items := v["items"].([]any); items[1] = items[0] })
		},
		"foreign-item": func(_ *http.Request, rec *httptest.ResponseRecorder) {
			govObject(rec, func(v map[string]any) { v["items"].([]any)[0].(map[string]any)["principal_id"] = "foreign" })
		},
		"changed-cut": func(r *http.Request, rec *httptest.ResponseRecorder) {
			if r.URL.Query().Get("cursor") != "" {
				govObject(rec, func(v map[string]any) { v["snapshot"] = 99999 })
			}
		},
		"lost-second-page": func(r *http.Request, rec *httptest.ResponseRecorder) {
			if r.URL.Query().Get("cursor") != "" {
				rec.Code = 503
				rec.Body.Reset()
				_, _ = rec.Body.WriteString(`{"code":"incomplete_result"}`)
			}
		},
		"repeated-cursor": func(r *http.Request, rec *httptest.ResponseRecorder) {
			if r.URL.Query().Get("cursor") != "" {
				govObject(rec, func(v map[string]any) { v["complete"] = false; v["next_cursor"] = r.URL.Query().Get("cursor") })
			}
		},
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			f := newGovFixture(t)
			c := f.client(t, "governor-secret-canary")
			for i := 0; i < 10; i++ {
				r := govCreate(fmt.Sprintf("key-%02d", i))
				r.Intent.Policy.AdmissionPolicyID = fmt.Sprintf("entitlement-%02d", i)
				if err := f.peer.AllowReference("admission_policy_id", r.Intent.Policy.AdmissionPolicyID, "operator-one"); err != nil {
					t.Fatal(err)
				}
				govApply(t, c, r)
			}
			c = govTransform(t, f, func(r *http.Request, rec *httptest.ResponseRecorder) {
				if r.URL.Path == govBase+"/governors" {
					change(r, rec)
				}
			})
			got, err := c.Governors(context.Background())
			if err == nil || got != nil {
				t.Fatal("partial or inconsistent collection returned")
			}
		})
	}
}

func govReceiptIdentity(t *testing.T) {
	for _, field := range []string{"id", "principal_id", "fingerprint", "resource_id", "location", "disappeared", "status-regressed"} {
		t.Run(field, func(t *testing.T) {
			f := newGovFixture(t)
			first := govApply(t, f.client(t, "governor-secret-canary"), govCreate("stable"))
			policy := govPolicy()
			request := govChange("stable", *first.ResourceID, "update", 1, 1, &policy)
			f.peer.SetHold(true)
			var commits atomic.Int32
			c := govTransform(t, f, func(r *http.Request, rec *httptest.ResponseRecorder) {
				if r.Method == "POST" {
					commits.Add(1)
					if field == "location" {
						rec.Header().Set("Location", "https://foreign.invalid/request")
					}
				}
				if r.Method == "GET" && strings.Contains(r.URL.Path, "/requests/") && commits.Load() > 0 {
					switch field {
					case "disappeared":
						rec.Code = 404
						rec.Body.Reset()
						_, _ = rec.Body.WriteString(`{"code":"request_not_found"}`)
					case "status-regressed":
						govObject(rec, func(v map[string]any) { v["status"] = "prepared" })
					case "location":
					default:
						govObject(rec, func(v map[string]any) {
							if field == "fingerprint" {
								v[field] = strings.Repeat("0", 64)
							} else {
								v[field] = "foreign"
							}
						})
					}
				}
			})
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			op, err := c.Apply(ctx, request)
			if err == nil || errors.Is(err, context.DeadlineExceeded) || commits.Load() != 1 || op.ID == "" {
				t.Fatal("receipt mismatch was retried or lost", field, commits.Load(), err)
			}
		})
	}
}

func govTransport(t *testing.T) {
	var leaked atomic.Int32
	destination := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { leaked.Add(1) }))
	defer destination.Close()
	redirect := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, destination.URL, http.StatusTemporaryRedirect)
	}))
	defer redirect.Close()
	c, err := pc.NewGovernor(pc.Config{Endpoint: redirect.URL, Scope: govScope, Token: "governor-secret-canary", HTTPClient: redirect.Client()})
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Capabilities(context.Background())
	if err == nil || leaked.Load() != 0 || strings.Contains(err.Error(), "governor-secret-canary") {
		t.Fatal("credential redirect")
	}
	c, err = pc.NewGovernor(pc.Config{Endpoint: redirect.URL, Scope: govScope, Token: "governor-secret-canary", PollInterval: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Capabilities(context.Background())
	if err == nil {
		t.Fatal("untrusted TLS accepted")
	}
	f := newGovFixture(t)
	trust := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: f.http.Certificate().Raw})
	verified, err := pc.NewGovernor(pc.Config{Endpoint: f.http.URL, Scope: govScope, Token: "governor-secret-canary", CAPEM: trust})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = verified.Capabilities(context.Background()); err != nil {
		t.Fatal("explicit CA rejected", err)
	}
	c = f.client(t, "governor-secret-canary")
	r := govCreate("stable")
	r.Intent.Policy.MaxWarmRunners = 1
	before := f.peer.Snapshot()
	_, err = c.Apply(context.Background(), r)
	if err == nil || !bytes.Equal(before, f.peer.Snapshot()) {
		t.Fatal("invalid policy dispatched")
	}
	denied := govTransform(t, f, func(r *http.Request, rec *httptest.ResponseRecorder) {
		if r.URL.Path == govBase+"/governors/absent" {
			rec.Code = 403
			rec.Body.Reset()
			_, _ = io.WriteString(rec.Body, `{"code":"resource_not_found"}`)
		}
	})
	_, err = denied.Governor(context.Background(), "absent")
	govError(t, err, "invalid_response")
}
