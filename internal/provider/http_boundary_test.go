package provider

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

func httpDynamic(t *testing.T, schema *tfprotov6.Schema, values map[string]any) *tfprotov6.DynamicValue {
	t.Helper()
	fields := map[string]any{}
	for _, a := range schema.Block.Attributes {
		fields[a.Name] = nil
	}
	for k, v := range values {
		fields[k] = v
	}
	data, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	return &tfprotov6.DynamicValue{JSON: data}
}

func httpScope(kind string) api.Scope {
	for _, s := range []api.Scope{{OrganizationID: "org"}, {OrganizationID: "org", ProjectID: "project"}, {OrganizationID: "org", ProjectID: "project", EnvironmentID: "env"}} {
		if s.AllowsKind(kind) {
			return s
		}
	}
	return api.Scope{}
}

// This enumerates the actual protocol registry, so a newly registered surface
// cannot silently escape coverage. The peer is a raw HTTPS server, independent
// of the fixture and client serializer. Every refresh must reach that endpoint,
// carry the common authentication/contract, and preserve its explicit denial.
func TestEveryRegisteredSurfaceUsesPlatformHTTP(t *testing.T) {
	ctx := context.Background()
	registry := providerserver.NewProtocol6(New("test")())()
	schemas, err := registry.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(schemas.ResourceSchemas) < 23 || len(schemas.DataSourceSchemas) < 47 {
		t.Fatal("registered service surface unexpectedly narrowed")
	}
	collections := map[string]string{}
	for name := range schemas.ResourceSchemas {
		kind := strings.TrimPrefix(name, "spoke_")
		if api.Collection(kind) == "" {
			t.Fatalf("resource %s has no HTTP collection", name)
		}
		collections[api.Collection(kind)] = kind
	}
	check := func(t *testing.T, name string, resource bool) {
		kind := strings.TrimPrefix(name, "spoke_")
		scope := httpScope(kind)
		input := map[string]any{}
		wantPath := scope.Path() + "/" + api.Collection(kind) + "/object"
		if !resource {
			switch {
			case collections[kind] != "":
				scope = httpScope(collections[kind])
				wantPath = scope.Path() + "/" + kind
			case kind == "project" || kind == "environment":
				scope = httpScope("bucket")
				wantPath = "/v5/organizations/org/projects/project"
				if kind == "environment" {
					wantPath += "/environments/env"
				}
			case kind == "artifact":
				scope = httpScope("bucket")
				input = map[string]any{"registry_id": "registry", "repository": "app", "tag": "latest"}
				wantPath = scope.Path() + "/artifact_registries/registry/artifact"
			case kind == "job_executions" || kind == "recovery_points":
				scope = httpScope("bucket")
				input["parent_id"] = "parent"
				wantPath = scope.Path() + "/jobs/parent/executions"
				if kind == "recovery_points" {
					wantPath = scope.Path() + "/backup_policies/parent/recovery_points"
				}
			case api.Collection(kind) != "":
				input["resource_id"] = "object"
			default:
				t.Fatalf("data source %s needs an HTTP request expectation", name)
			}
		}
		if !scope.Valid() {
			t.Fatalf("invalid HTTP scope for %s", name)
		}
		requests := make(chan *http.Request, 8)
		peer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests <- r.Clone(context.Background())
			w.Header().Set("X-Spoke-API-Version", api.APIVersion)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"code":"forbidden"}`))
		}))
		defer peer.Close()
		dir := t.TempDir()
		caFile, tokenFile := filepath.Join(dir, "ca.pem"), filepath.Join(dir, "token")
		if err := os.WriteFile(caFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: peer.Certificate().Raw}), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(tokenFile, []byte("http-boundary-credential"), 0600); err != nil {
			t.Fatal(err)
		}
		config := map[string]any{"endpoint": peer.URL, "organization_id": scope.OrganizationID, "ca_file": caFile, "token_file": tokenFile}
		if scope.ProjectID != "" {
			config["project_id"] = scope.ProjectID
		}
		if scope.EnvironmentID != "" {
			config["environment_id"] = scope.EnvironmentID
		}
		server := providerserver.NewProtocol6(New("test")())()
		configured, err := server.ConfigureProvider(ctx, &tfprotov6.ConfigureProviderRequest{TerraformVersion: "1.15.3", Config: httpDynamic(t, schemas.Provider, config)})
		if err != nil || len(configured.Diagnostics) != 0 {
			t.Fatalf("configure: %v %+v", err, configured)
		}
		var diagnostics []*tfprotov6.Diagnostic
		if resource {
			id := scope.OrganizationID + "/object"
			if scope.ProjectID != "" {
				id = scope.OrganizationID + "/" + scope.ProjectID + "/object"
			}
			if scope.EnvironmentID != "" {
				id = canonicalID(scope, "object")
			}
			imported, err := server.ImportResourceState(ctx, &tfprotov6.ImportResourceStateRequest{TypeName: name, ID: id})
			if err != nil || len(imported.Diagnostics) != 0 || len(imported.ImportedResources) != 1 {
				t.Fatalf("import setup: %v %+v", err, imported)
			}
			prior := imported.ImportedResources[0].State
			read, err := server.ReadResource(ctx, &tfprotov6.ReadResourceRequest{TypeName: name, CurrentState: prior})
			if err != nil {
				t.Fatal(err)
			}
			diagnostics = read.Diagnostics
			// Denial cannot erase the imported identity.
			before, err := prior.Unmarshal(schemas.ResourceSchemas[name].ValueType())
			if err != nil {
				t.Fatal(err)
			}
			after, err := read.NewState.Unmarshal(schemas.ResourceSchemas[name].ValueType())
			if err != nil || !before.Equal(after) {
				t.Fatal("HTTP denial changed resource state", err)
			}
		} else {
			read, err := server.ReadDataSource(ctx, &tfprotov6.ReadDataSourceRequest{TypeName: name, Config: httpDynamic(t, schemas.DataSourceSchemas[name], input)})
			if err != nil {
				t.Fatal(err)
			}
			diagnostics = read.Diagnostics
		}
		if len(diagnostics) != 1 || diagnostics[0].Severity != tfprotov6.DiagnosticSeverityError || !strings.Contains(diagnostics[0].Detail, "forbidden (HTTP 403)") {
			t.Fatalf("HTTP denial was bypassed or lost: %+v", diagnostics)
		}
		if len(requests) != 1 {
			t.Fatalf("expected one HTTP observation, got %d", len(requests))
		}
		req := <-requests
		if req.Method != http.MethodGet || req.URL.Path != wantPath || req.Header.Get("Authorization") != "Bearer http-boundary-credential" || req.Header.Get("X-Spoke-Organization") != "org" || req.Header.Get("X-Spoke-API-Version") != api.APIVersion || req.Header.Get("Accept") != "application/json" {
			t.Fatalf("%s bypassed shared HTTP semantics: %s %s", name, req.Method, req.URL.Path)
		}
	}
	for name := range schemas.ResourceSchemas {
		t.Run("resource/"+name, func(t *testing.T) { check(t, name, true) })
	}
	for name := range schemas.DataSourceSchemas {
		t.Run("data/"+name, func(t *testing.T) { check(t, name, false) })
	}
}

// Terraform owns its RPC transport; service network and SDK code belongs only in
// platformclient. Enforce that boundary for future production source files too.
func TestProviderServiceBoundaryImports(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	clients, err := filepath.Glob("../platformclient/*.go")
	if err != nil {
		t.Fatal(err)
	}
	files = append(files, clients...)
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		ast, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range ast.Imports {
			name, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				t.Fatal(err)
			}
			client := strings.HasPrefix(file, "../platformclient/")
			network := name == "net" || (strings.HasPrefix(name, "net/") && name != "net/netip" && name != "net/url") || name == "crypto/tls"
			if name == "C" || (!client && network) || name == "os/exec" || name == "syscall" || name == "unsafe" || name == "plugin" || name == "database/sql" {
				t.Errorf("%s bypasses Platform HTTP client with %s", file, name)
			}
			if strings.Contains(strings.Split(name, "/")[0], ".") && !strings.HasPrefix(name, "github.com/hashicorp/") && name != "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient" && !(client && name == "github.com/robfig/cron/v3") {
				t.Errorf("%s adds service dependency outside the Platform HTTP client: %s", file, name)
			}
		}
	}
}
