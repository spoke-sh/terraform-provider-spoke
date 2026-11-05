package provider

import (
	"context"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

func checkGovernorHTTPBoundary(t *testing.T, name string, isResource bool) {
	t.Helper()
	ctx := context.Background()
	requests := make(chan *http.Request, 8)
	peer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.Clone(ctx)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Spoke-API-Version", api.GovernorVersion)
		w.Header().Set("X-Spoke-Management-Profile", api.GovernorProfile)
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(403)
		_, _ = w.Write([]byte(`{"code":"forbidden"}`))
	}))
	defer peer.Close()
	dir := t.TempDir()
	ca, token := filepath.Join(dir, "ca"), filepath.Join(dir, "token")
	for path, b := range map[string][]byte{ca: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: peer.Certificate().Raw}), token: []byte("boundary-governor-token")} {
		if err := os.WriteFile(path, b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	s := providerserver.NewProtocol6(New("test")())()
	schemas, err := s.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	c, err := s.ConfigureProvider(ctx, &tfprotov6.ConfigureProviderRequest{Config: httpDynamic(t, schemas.Provider, map[string]any{"endpoint": peer.URL, "organization_id": "example", "project_id": "operators", "environment_id": "internal", "ca_file": ca, "token_file": token})})
	if err != nil {
		t.Fatal(err)
	}
	governorNoDiagnostics(t, c.Diagnostics)
	var ds []*tfprotov6.Diagnostic
	if isResource {
		prior := httpDynamic(t, schemas.ResourceSchemas[name], map[string]any{"id": "example/operators/internal/governor-one", "resource_id": "governor-one", "owner_id": "operator", "management_key": "stable", "revision": 1, "operation_sequence": 0})
		res, err := s.ReadResource(ctx, &tfprotov6.ReadResourceRequest{TypeName: name, CurrentState: prior})
		if err != nil {
			t.Fatal(err)
		}
		ds = res.Diagnostics
		before, _ := prior.Unmarshal(schemas.ResourceSchemas[name].ValueType())
		after, e := res.NewState.Unmarshal(schemas.ResourceSchemas[name].ValueType())
		if e != nil || !before.Equal(after) {
			t.Fatal("governor denial changed state", e)
		}
	} else {
		cfg := map[string]any{}
		if name == "spoke_ci_governor" {
			cfg["id"] = "example/operators/internal/governor-one"
		}
		res, err := s.ReadDataSource(ctx, &tfprotov6.ReadDataSourceRequest{TypeName: name, Config: httpDynamic(t, schemas.DataSourceSchemas[name], cfg)})
		if err != nil {
			t.Fatal(err)
		}
		ds = res.Diagnostics
	}
	if len(ds) != 1 || !strings.Contains(ds[0].Detail, "forbidden") || len(requests) != 1 {
		t.Fatalf("governor denial bypassed: %+v requests=%d", ds, len(requests))
	}
	r := <-requests
	if r.Method != "GET" || r.URL.Path != governorTestScope+"/capabilities" || r.Header.Get("Authorization") != "Bearer boundary-governor-token" || r.Header.Get("X-Spoke-API-Version") != api.GovernorVersion || r.Header.Get("X-Spoke-Organization") != "example" || r.Header.Get("Accept") != "application/json" {
		t.Fatal("governor bypassed scoped HTTPS")
	}
}
