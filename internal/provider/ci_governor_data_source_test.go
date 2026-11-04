package provider

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

func TestGovernorDiscovery(t *testing.T) {
	f := newGovernorFixture(t)
	p := newGovernorRPC(t, f)
	ctx := context.Background()
	schemas, err := p.server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	read := func(name string, values map[string]any, wantError bool) *tfprotov6.ReadDataSourceResponse {
		t.Helper()
		before := f.writes.Load()
		r, e := p.server.ReadDataSource(ctx, &tfprotov6.ReadDataSourceRequest{TypeName: name, Config: httpDynamic(t, schemas.DataSourceSchemas[name], values)})
		if e != nil {
			t.Fatal(e)
		}
		if wantError {
			if len(r.Diagnostics) == 0 {
				t.Fatal("invalid discovery succeeded")
			}
		} else {
			governorNoDiagnostics(t, r.Diagnostics)
		}
		if f.writes.Load() != before {
			t.Fatal("discovery submitted work")
		}
		return r
	}
	read("spoke_ci_governors", nil, false)
	read("spoke_ci_governor", map[string]any{"id": "foreign/operators/internal/governor-one"}, true)
	read("spoke_ci_governor", map[string]any{"id": "example/operators/internal/missing"}, true)
	client := f.client(t)
	var first string
	for i := range 12 {
		policy := governorTestPolicy(fmt.Sprintf("policy-%d", i))
		policy.AdmissionPolicyID = fmt.Sprintf("admission-%d", i)
		if err := f.current.Load().AllowReference("admission_policy_id", policy.AdmissionPolicyID, "operator"); err != nil {
			t.Fatal(err)
		}
		op, err := client.Apply(ctx, api.GovernorRequest{ManagementKey: fmt.Sprintf("key-%d", i), Intent: api.GovernorIntent{Kind: "create", Policy: policy}})
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			first = *op.ResourceID
		}
	}
	read("spoke_ci_governor", map[string]any{"id": governorCanonical(governorScopeValue, first)}, false)
	r := read("spoke_ci_governors", nil, false)
	v, err := r.State.Unmarshal(schemas.DataSourceSchemas["spoke_ci_governors"].ValueType())
	if err != nil {
		t.Fatal(err)
	}
	// Decode the wire value without reusing the provider's mapping.
	var attrs map[string]tftypes.Value
	if err := v.As(&attrs); err != nil {
		t.Fatal(err)
	}
	var items []tftypes.Value
	if err := attrs["governors"].As(&items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 12 {
		t.Fatal("incomplete collection")
	}
	last := ""
	for _, g := range items {
		var fields map[string]tftypes.Value
		if err := g.As(&fields); err != nil {
			t.Fatal(err)
		}
		var id, readiness string
		var jobs big.Float
		if err := fields["id"].As(&id); err != nil {
			t.Fatal(err)
		}
		if err := fields["readiness"].As(&readiness); err != nil {
			t.Fatal(err)
		}
		if err := fields["active_jobs"].As(&jobs); err != nil {
			t.Fatal(err)
		}
		if readiness != "disabled" || jobs.Sign() != 0 || id <= last {
			t.Fatal("unstable/runtime projection")
		}
		last = id
	}
	f.hook.Store(&governorHTTPHook{serve: func(w http.ResponseWriter, r *http.Request, next http.Handler) {
		if r.URL.Path != governorTestScope+"/governors" {
			next.ServeHTTP(w, r)
			return
		}
		rec := httptest.NewRecorder()
		next.ServeHTTP(rec, r)
		for k, vs := range rec.Header() {
			w.Header()[k] = vs
		}
		w.WriteHeader(503)
		_, _ = w.Write([]byte(`{"code":"incomplete_result"}`))
	}})
	read("spoke_ci_governors", nil, true)
	f.hook.Store(nil)
	if err := f.current.Load().Revoke("operator", true); err != nil {
		t.Fatal(err)
	}
	read("spoke_ci_governor", map[string]any{"id": governorCanonical(governorScopeValue, first)}, true)
	read("spoke_ci_governors", nil, true)
}
