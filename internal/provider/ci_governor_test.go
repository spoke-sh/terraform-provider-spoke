package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	fwprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
	peer "github.com/spoke-sh/terraform-provider-spoke/internal/testutil/mockplatform/governor"
)

var governorScopeValue = api.Scope{OrganizationID: "example", ProjectID: "operators", EnvironmentID: "internal"}
var governorTokens = map[string]peer.Identity{"governor-secret-canary": {Principal: "operator", Operator: true}, "foreign-secret-canary": {Principal: "foreign", Operator: true}, "reader-secret-canary": {Principal: "operator"}}

type governorHTTPHook struct {
	serve func(http.ResponseWriter, *http.Request, http.Handler)
}
type governorFixture struct {
	server            *httptest.Server
	current           atomic.Pointer[peer.Server]
	hook              atomic.Pointer[governorHTTPHook]
	writes            atomic.Int64
	ledger, ca, token string
	schema            []byte
}

func newGovernorFixture(t *testing.T) *governorFixture {
	t.Helper()
	f := &governorFixture{}
	var err error
	f.schema, err = os.ReadFile("../platformclient/governor-contract/schema.json")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	f.ledger, f.ca, f.token = filepath.Join(dir, "ledger.json"), filepath.Join(dir, "ca.pem"), filepath.Join(dir, "token")
	f.reload(t)
	for kind, id := range map[string]string{"admission_policy_id": "admission", "budget_policy_id": "budget", "cache_policy_id": "cache"} {
		if err := f.current.Load().AllowReference(kind, id, "operator", "foreign"); err != nil {
			t.Fatal(err)
		}
	}
	f.server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			f.writes.Add(1)
		}
		next := f.current.Load()
		if h := f.hook.Load(); h != nil {
			h.serve(w, r, next)
		} else {
			next.ServeHTTP(w, r)
		}
	}))
	t.Cleanup(f.server.Close)
	for path, b := range map[string][]byte{f.ca: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: f.server.Certificate().Raw}), f.token: []byte("governor-secret-canary")} {
		if err := os.WriteFile(path, b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	return f
}
func (f *governorFixture) reload(t *testing.T) {
	t.Helper()
	s, err := peer.New(f.ledger, governorTestScope, f.schema, governorTokens)
	if err != nil {
		t.Fatal(err)
	}
	f.current.Store(s)
}
func (f *governorFixture) client(t *testing.T) *api.GovernorClient {
	t.Helper()
	c, err := api.NewGovernor(api.Config{Endpoint: f.server.URL, Scope: governorScopeValue, Token: "governor-secret-canary", HTTPClient: f.server.Client(), PollInterval: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// Exercise the adapter through Framework RPC before exposing the registry entry.
type governorTestProvider struct{ fwprovider.Provider }

func (p governorTestProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{newGovernorResource}
}

type governorRPC struct {
	t       *testing.T
	server  tfprotov6.ProviderServer
	schema  *tfprotov6.Schema
	state   *tfprotov6.DynamicValue
	private []byte
}

func newGovernorRPC(t *testing.T, f *governorFixture) *governorRPC {
	t.Helper()
	s := providerserver.NewProtocol6(governorTestProvider{New("test")()})()
	schemas, err := s.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	governorNoDiagnostics(t, schemas.Diagnostics)
	res, err := s.ConfigureProvider(context.Background(), &tfprotov6.ConfigureProviderRequest{Config: httpDynamic(t, schemas.Provider, map[string]any{"endpoint": f.server.URL, "organization_id": "example", "project_id": "operators", "environment_id": "internal", "ca_file": f.ca, "token_file": f.token, "timeout_seconds": 1})})
	if err != nil {
		t.Fatal(err)
	}
	governorNoDiagnostics(t, res.Diagnostics)
	return &governorRPC{t: t, server: s, schema: schemas.ResourceSchemas["spoke_ci_governor"], state: &tfprotov6.DynamicValue{JSON: []byte("null")}}
}
func governorNoDiagnostics(t *testing.T, ds []*tfprotov6.Diagnostic) {
	t.Helper()
	for _, d := range ds {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("%s: %s", d.Summary, d.Detail)
		}
	}
}
func governorConfig(name string, protection bool) map[string]any {
	return map[string]any{"management_key": "stable", "name": name, "admission_policy_id": "admission", "budget_policy_id": "budget", "cache_policy_id": "cache", "execution_profile_ids": []string{"linux-x86_64-v1"}, "max_active_jobs": 2, "max_allocated_vcpus": 16, "max_warm_runners": 0, "max_job_seconds": 3600, "max_queue_seconds": 60, "retirement_policy": "drain", "deletion_protection": protection}
}
func (p *governorRPC) plan(values map[string]any) (*tfprotov6.PlanResourceChangeResponse, *tfprotov6.DynamicValue) {
	p.t.Helper()
	config := &tfprotov6.DynamicValue{JSON: []byte("null")}
	if values != nil {
		config = httpDynamic(p.t, p.schema, values)
	}
	res, err := p.server.PlanResourceChange(context.Background(), &tfprotov6.PlanResourceChangeRequest{TypeName: "spoke_ci_governor", PriorState: p.state, ProposedNewState: config, Config: config, PriorPrivate: p.private})
	if err != nil {
		p.t.Fatal(err)
	}
	return res, config
}
func (p *governorRPC) apply(plan *tfprotov6.PlanResourceChangeResponse, config *tfprotov6.DynamicValue) *tfprotov6.ApplyResourceChangeResponse {
	p.t.Helper()
	res, err := p.server.ApplyResourceChange(context.Background(), &tfprotov6.ApplyResourceChangeRequest{TypeName: "spoke_ci_governor", PriorState: p.state, PlannedState: plan.PlannedState, PlannedPrivate: plan.PlannedPrivate, Config: config})
	if err != nil {
		p.t.Fatal(err)
	}
	return res
}
func (p *governorRPC) change(values map[string]any) {
	p.t.Helper()
	plan, cfg := p.plan(values)
	governorNoDiagnostics(p.t, plan.Diagnostics)
	res := p.apply(plan, cfg)
	governorNoDiagnostics(p.t, res.Diagnostics)
	p.state, p.private = res.NewState, res.Private
}
func (p *governorRPC) values() map[string]tftypes.Value {
	p.t.Helper()
	v, err := p.state.Unmarshal(p.schema.ValueType())
	if err != nil {
		p.t.Fatal(err)
	}
	var values map[string]tftypes.Value
	if err = v.As(&values); err != nil {
		p.t.Fatal(err)
	}
	return values
}

func TestGovernorFrameworkLifecycle(t *testing.T) {
	f := newGovernorFixture(t)
	p := newGovernorRPC(t, f)
	p.change(governorConfig("a", false))
	var id string
	if err := p.values()["resource_id"].As(&id); err != nil {
		t.Fatal(err)
	}
	for i, name := range []string{"b", "a"} {
		p.change(governorConfig(name, false))
		op, err := f.client(t).Management(context.Background(), "stable")
		if err != nil || op.Sequence != uint64(i+1) || op.ResourceID == nil || *op.ResourceID != id {
			t.Fatalf("distinct intent: %+v %v", op, err)
		}
	}
	before := f.writes.Load()
	p.change(governorConfig("a", false))
	imported, err := p.server.ImportResourceState(context.Background(), &tfprotov6.ImportResourceStateRequest{TypeName: "spoke_ci_governor", ID: "example/operators/internal/" + id})
	if err != nil {
		t.Fatal(err)
	}
	governorNoDiagnostics(t, imported.Diagnostics)
	if len(imported.ImportedResources) != 1 || f.writes.Load() != before {
		t.Fatal("import/no-op wrote intent")
	}
	p.change(nil)
	if v, err := p.state.Unmarshal(p.schema.ValueType()); err != nil || !v.IsNull() {
		t.Fatal("retirement retained state", err)
	}
	op, err := f.client(t).Management(context.Background(), "stable")
	if err != nil || op.Sequence != 3 || op.Status != "applied" {
		t.Fatal("retirement lost receipt", err)
	}
	plan, _ := p.plan(governorConfig("a", false))
	if len(plan.Diagnostics) == 0 {
		t.Fatal("retired key reused")
	}
}

func TestGovernorFrameworkFences(t *testing.T) {
	for _, scenario := range []string{"saved-update-revision", "saved-destroy-revision", "saved-create-principal", "protected", "pending-changed", "receipt-expired", "denied-refresh", "unknown", "null", "private-version", "lost-create-completion"} {
		t.Run(scenario, func(t *testing.T) {
			f := newGovernorFixture(t)
			p := newGovernorRPC(t, f)
			if scenario == "unknown" || scenario == "null" {
				cfg := httpDynamic(t, p.schema, governorConfig("a", false))
				v, _ := cfg.Unmarshal(p.schema.ValueType())
				var attrs map[string]tftypes.Value
				_ = v.As(&attrs)
				if scenario == "unknown" {
					attrs["name"] = tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
				} else {
					attrs["name"] = tftypes.NewValue(tftypes.String, nil)
				}
				dynamic, _ := tfprotov6.NewDynamicValue(p.schema.ValueType(), tftypes.NewValue(p.schema.ValueType(), attrs))
				res, err := p.server.ValidateResourceConfig(context.Background(), &tfprotov6.ValidateResourceConfigRequest{TypeName: "spoke_ci_governor", Config: &dynamic})
				if err != nil {
					t.Fatal(err)
				}
				if scenario == "unknown" {
					governorNoDiagnostics(t, res.Diagnostics)
				} else if len(res.Diagnostics) == 0 {
					t.Fatal("null accepted")
				}
				if f.writes.Load() != 0 {
					t.Fatal("validation wrote")
				}
				return
			}
			if scenario == "saved-create-principal" || scenario == "lost-create-completion" {
				plan, cfg := p.plan(governorConfig("a", false))
				governorNoDiagnostics(t, plan.Diagnostics)
				if scenario == "saved-create-principal" {
					if err := os.WriteFile(f.token, []byte("foreign-secret-canary"), 0600); err != nil {
						t.Fatal(err)
					}
					p = newGovernorRPC(t, f)
					res := p.apply(plan, cfg)
					if len(res.Diagnostics) == 0 || f.writes.Load() != 0 {
						t.Fatal("create rebased owner")
					}
				} else {
					res := p.apply(plan, cfg)
					governorNoDiagnostics(t, res.Diagnostics)
					f.reload(t)
					p = newGovernorRPC(t, f)
					p.change(governorConfig("a", false))
					op, _ := f.client(t).Management(context.Background(), "stable")
					if op.Sequence != 0 {
						t.Fatal("lost creation allocated successor")
					}
				}
				return
			}
			p.change(governorConfig("a", scenario == "protected"))
			client := f.client(t)
			op, _ := client.Management(context.Background(), "stable")
			id := *op.ResourceID
			if scenario == "denied-refresh" {
				f.current.Load().Revoke("operator", true)
				before := f.writes.Load()
				res, err := p.server.ReadResource(context.Background(), &tfprotov6.ReadResourceRequest{TypeName: "spoke_ci_governor", CurrentState: p.state, Private: p.private})
				if err != nil {
					t.Fatal(err)
				}
				if len(res.Diagnostics) == 0 || f.writes.Load() != before {
					t.Fatal("denial lost")
				}
				old, _ := p.state.Unmarshal(p.schema.ValueType())
				next, _ := res.NewState.Unmarshal(p.schema.ValueType())
				if !old.Equal(next) {
					t.Fatal("denial erased state")
				}
				return
			}
			if scenario == "protected" {
				plan, _ := p.plan(nil)
				if len(plan.Diagnostics) == 0 {
					t.Fatal("protected destroy planned")
				}
				return
			}
			if scenario == "pending-changed" {
				f.current.Load().SetHold(true)
				policy := governorTestPolicy("b")
				_, err := client.Prepare(context.Background(), api.GovernorRequest{ManagementKey: "stable", Sequence: 1, Intent: api.GovernorIntent{Kind: "update", TargetID: &id, ExpectedRevision: 1, Policy: policy}})
				if err != nil {
					t.Fatal(err)
				}
				before := f.writes.Load()
				plan, _ := p.plan(governorConfig("c", false))
				if len(plan.Diagnostics) == 0 || f.writes.Load() != before {
					t.Fatal("conflicting intent resumed")
				}
				return
			}
			if scenario == "receipt-expired" {
				if err := f.current.Load().Expire("stable", 0); err != nil {
					t.Fatal(err)
				}
				plan, _ := p.plan(governorConfig("b", false))
				if len(plan.Diagnostics) == 0 {
					t.Fatal("expired identity reused")
				}
				return
			}
			cfgValues := governorConfig("b", false)
			if scenario == "saved-destroy-revision" {
				cfgValues = nil
			}
			plan, cfg := p.plan(cfgValues)
			governorNoDiagnostics(t, plan.Diagnostics)
			before := f.writes.Load()
			if scenario == "private-version" {
				var data map[string][]byte
				if err := json.Unmarshal(plan.PlannedPrivate, &data); err != nil {
					t.Fatal(err)
				}
				data[governorPrivateKey] = bytes.ReplaceAll(data[governorPrivateKey], []byte(`"version":1`), []byte(`"version":99`))
				plan.PlannedPrivate, _ = json.Marshal(data)
			} else {
				if err := f.current.Load().Drift(id); err != nil {
					t.Fatal(err)
				}
			}
			res := p.apply(plan, cfg)
			if len(res.Diagnostics) == 0 || f.writes.Load() != before {
				b, _ := json.Marshal(res.Diagnostics)
				t.Fatalf("stale fence mutated: %s", b)
			}
			for _, d := range res.Diagnostics {
				if strings.Contains(d.Detail, "secret-canary") {
					t.Fatal("credential leaked")
				}
			}
		})
	}
}
