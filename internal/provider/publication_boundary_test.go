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
	kind := map[string]string{"spoke_static_application": "application", "spoke_static_applications": "application", "spoke_static_target_binding": "binding", "spoke_static_target_bindings": "binding", "spoke_static_target": "target", "spoke_static_targets": "target"}[name]
	if kind == "" {
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
	providerConfig := map[string]any{"endpoint": peer.URL, "organization_id": "example", "project_id": "saga", "token_file": token, "ca_file": ca}
	base := "/v1/publication/organizations/example/projects/saga"
	canonical := "example/saga/1"
	if kind == "target" {
		providerConfig["environment_id"] = "nightly"
		base += "/environments/nightly"
		canonical = "example/saga/nightly/1"
	}
	configured, err := server.ConfigureProvider(ctx, &tfprotov6.ConfigureProviderRequest{TerraformVersion: "1.15.3", Config: httpDynamic(t, schemas.Provider, providerConfig)})
	if err != nil || len(configured.Diagnostics) != 0 {
		t.Fatalf("configure: %v %+v", err, configured)
	}
	var diagnostics []*tfprotov6.Diagnostic
	if resource {
		key, _ := publicationKey(kind, "boundary", 0)
		prior := httpDynamic(t, schemas.ResourceSchemas[name], map[string]any{"id": canonical, "resource_id": "1", "management_key": "boundary", "owner_id": "201", "operation_key": key})
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
		config := map[string]any{}
		if strings.HasSuffix(name, "s") {
			if kind == "binding" {
				config["application_id"] = "1"
			}
		} else {
			config["resource_id"] = "1"
		}
		read, err := server.ReadDataSource(ctx, &tfprotov6.ReadDataSourceRequest{TypeName: name, Config: httpDynamic(t, schemas.DataSourceSchemas[name], config)})
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
	if r.Method != "GET" || r.URL.Path != base+"/capabilities" || r.Header.Get("Authorization") != "Bearer boundary-publication-token" || r.Header.Get("X-Spoke-API-Version") != api.PublicationVersion || r.Header.Get("X-Spoke-Organization") != "example" {
		t.Fatal("publication bypassed scoped HTTPS contract")
	}
}
