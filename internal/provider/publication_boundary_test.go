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

func checkPublicationHTTPBoundary(t *testing.T, name string, resource bool) {
	t.Helper()
	if name != "spoke_static_application" {
		t.Fatalf("publication surface %s needs explicit HTTP scope expectations", name)
	}
	ctx := context.Background()
	requests := make(chan *http.Request, 8)
	peer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.Clone(ctx)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Spoke-API-Version", api.PublicationVersion)
		w.Header().Set("X-Spoke-Management-Profile", api.PublicationProfile)
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(403)
		_, _ = w.Write([]byte(`{"code":"forbidden"}`))
	}))
	defer peer.Close()
	dir := t.TempDir()
	ca, token := filepath.Join(dir, "ca"), filepath.Join(dir, "token")
	if err := os.WriteFile(ca, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: peer.Certificate().Raw}), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(token, []byte("boundary-publication-token"), 0600); err != nil {
		t.Fatal(err)
	}
	server := providerserver.NewProtocol6(New("test")())()
	schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	configured, err := server.ConfigureProvider(ctx, &tfprotov6.ConfigureProviderRequest{TerraformVersion: "1.15.3", Config: httpDynamic(t, schemas.Provider, map[string]any{"endpoint": peer.URL, "organization_id": "example", "project_id": "saga", "token_file": token, "ca_file": ca})})
	if err != nil || len(configured.Diagnostics) != 0 {
		t.Fatalf("configure: %v %+v", err, configured)
	}
	var diagnostics []*tfprotov6.Diagnostic
	if resource {
		prior := httpDynamic(t, schemas.ResourceSchemas[name], map[string]any{"id": "example/saga/1", "resource_id": "1", "management_key": "boundary", "owner_id": "201", "operation_key": "tf1a_boundary_00"})
		read, err := server.ReadResource(ctx, &tfprotov6.ReadResourceRequest{TypeName: name, CurrentState: prior})
		if err != nil {
			t.Fatal(err)
		}
		diagnostics = read.Diagnostics
		before, err := prior.Unmarshal(schemas.ResourceSchemas[name].ValueType())
		if err != nil {
			t.Fatal(err)
		}
		after, err := read.NewState.Unmarshal(schemas.ResourceSchemas[name].ValueType())
		if err != nil || !before.Equal(after) {
			t.Fatal("publication denial changed state", err)
		}
	} else {
		read, err := server.ReadDataSource(ctx, &tfprotov6.ReadDataSourceRequest{TypeName: name, Config: httpDynamic(t, schemas.DataSourceSchemas[name], map[string]any{"resource_id": "1"})})
		if err != nil {
			t.Fatal(err)
		}
		diagnostics = read.Diagnostics
	}
	if len(diagnostics) != 1 || !strings.Contains(diagnostics[0].Detail, "forbidden") {
		t.Fatalf("denial lost: %+v", diagnostics)
	}
	if len(requests) != 1 {
		t.Fatalf("expected one observational request: %d", len(requests))
	}
	r := <-requests
	if r.Method != "GET" || r.URL.Path != "/v1/publication/organizations/example/projects/saga/capabilities" || r.Header.Get("Authorization") != "Bearer boundary-publication-token" || r.Header.Get("X-Spoke-API-Version") != api.PublicationVersion || r.Header.Get("X-Spoke-Organization") != "example" {
		t.Fatal("publication bypassed scoped HTTPS contract")
	}
}
