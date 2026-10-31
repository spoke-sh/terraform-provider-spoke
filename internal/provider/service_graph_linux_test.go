//go:build linux

package provider

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
	"github.com/spoke-sh/terraform-provider-spoke/internal/testutil/mockplatform"
)

func (p *consumer) resourceIDs() map[string]string {
	p.t.Helper()
	var state struct {
		Resources []struct {
			Mode, Type, Name string
			Instances        []struct {
				Attributes struct {
					ResourceID string `json:"resource_id"`
				} `json:"attributes"`
			}
		}
	}
	if err := json.Unmarshal(p.run("state", "pull"), &state); err != nil {
		p.t.Fatal(err)
	}
	ids := map[string]string{}
	for _, r := range state.Resources {
		if r.Mode == "managed" && len(r.Instances) == 1 {
			ids[r.Type+"."+r.Name] = r.Instances[0].Attributes.ResourceID
		}
	}
	return ids
}
func graphConsumer(t *testing.T, s *mockplatform.Server, binary string) (*consumer, api.Scope, string, map[string]any) {
	t.Helper()
	bootstrap := newConsumer(t, binary, scopeGraphConfig(s, "graph-bootstrap"))
	bootstrap.apply()
	ids := bootstrap.resourceIDs()
	scope := api.Scope{OrganizationID: s.Scope.OrganizationID, ProjectID: ids["spoke_project.test"], EnvironmentID: ids["spoke_environment.test"]}
	config, err := os.ReadFile("../../examples/service-graph/main.tf")
	if err != nil {
		t.Fatal(err)
	}
	p := newConsumer(t, binary, string(config))
	vars := map[string]any{"endpoint": s.HTTP.URL, "organization_id": scope.OrganizationID, "project_id": scope.ProjectID, "environment_id": scope.EnvironmentID, "service_account_id": ids["spoke_service_account.test"], "token_file": s.TokenFile, "ca_file": s.CAFile, "image_digest": mockplatform.TestDigest, "secret_version": "v1"}
	p.variables(vars)
	return p, scope, ids["spoke_service_account.test"], vars
}
func (p *consumer) variables(v map[string]any) {
	p.t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		p.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(p.dir, "terraform.tfvars.json"), b, 0600); err != nil {
		p.t.Fatal(err)
	}
}
func graphObject(t *testing.T, s *mockplatform.Server, scope api.Scope, kind, id string) api.Object {
	t.Helper()
	o, ok := s.Object(scope, kind, id)
	if !ok {
		t.Fatalf("missing %s/%s", kind, id)
	}
	return o
}
func TestAccMockInstalledServiceGraph(t *testing.T) {
	s := mockAcceptance(t)
	p, scope, account, vars := graphConsumer(t, s, buildInstalledProvider(t))
	p.apply()
	ids := p.resourceIDs()
	registry := ids["spoke_artifact_registry.images"]
	app := ids["spoke_container_service.app"]
	endpoint := ids["spoke_endpoint.app"]
	secret := ids["spoke_secret_reference.credential"]
	s.AdvanceControllers()
	if graphObject(t, s, scope, "container_service", app).Status.Reason != "artifact_missing" {
		t.Fatal("missing image not observed")
	}
	if err := s.PublishArtifact(scope, registry, "app/web", mockplatform.TestDigest, "release", 128); err != nil {
		t.Fatal(err)
	}
	s.SetSecretVersion(api.SecretSpec{ExternalOwner: "external", ExternalID: "credential", Version: "v1"}, true)
	s.AdvanceControllers()
	p.apply()
	old := graphObject(t, s, scope, "container_service", app)
	if old.Status.Readiness != "ready" || old.Status.ActiveRevision == 0 || s.Count("service_binding") != 4 {
		t.Fatal("graph did not become ready")
	}
	if graphObject(t, s, scope, "endpoint", endpoint).Status.TLSState != "ready" {
		t.Fatal("TLS observation not ready")
	}
	before := mutations(s)
	p.run("plan", "-detailed-exitcode", "-input=false", "-no-color")
	if mutations(s) != before {
		t.Fatal("empty plan mutated graph")
	}
	// Moving a tag cannot alter the digest of the managed application.
	if err := s.PublishArtifact(scope, registry, "app/web", mockplatform.NextDigest, "release", 256); err != nil {
		t.Fatal(err)
	}
	p.run("plan", "-detailed-exitcode", "-input=false", "-no-color")
	vars["image_digest"], vars["secret_version"] = mockplatform.NextDigest, "v2"
	p.variables(vars)
	p.apply()
	current := graphObject(t, s, scope, "container_service", app)
	if current.Container.Digest != mockplatform.NextDigest || current.Status.DesiredRevision != old.Status.DesiredRevision+1 || current.Status.ActiveRevision != old.Status.ActiveRevision || current.Status.Readiness != "pending" {
		t.Fatalf("rollout observations: %+v", current.Status)
	}
	s.AdvanceControllers()
	if graphObject(t, s, scope, "container_service", app).Status.Reason != "dependency_unavailable" {
		t.Fatal("unresolved rotation not propagated")
	}
	s.SetSecretVersion(api.SecretSpec{ExternalOwner: "external", ExternalID: "credential", Version: "v2"}, true)
	s.SetHealthFailure(app, "health_failed")
	s.AdvanceControllers()
	if got := graphObject(t, s, scope, "container_service", app); got.Status.Readiness != "failed" || got.Status.ActiveRevision != old.Status.ActiveRevision {
		t.Fatal("failed rollout lost active revision")
	}
	s.SetHealthFailure(app, "")
	s.SetHealthFailure(endpoint, "tls_failed")
	s.AdvanceControllers()
	if graphObject(t, s, scope, "endpoint", endpoint).Status.TLSState != "failed" {
		t.Fatal("TLS failure not exposed")
	}
	s.SetHealthFailure(endpoint, "")
	s.AdvanceControllers()
	p.apply()
	if graphObject(t, s, scope, "secret_reference", secret).Status.ResolvedVersion != "v2" {
		t.Fatal("rotation unresolved")
	}
	// Revocation affects runtime observations, never declaration ownership or data.
	vars["grant_database"] = false
	p.variables(vars)
	p.apply()
	s.AdvanceControllers()
	p.apply()
	db := ids["spoke_database.app"]
	if s.RuntimeAllowed(scope, account, "database", db, "database_connect") || graphObject(t, s, scope, "container_service", app).Status.Readiness == "ready" {
		t.Fatal("revoked runtime access remained usable")
	}
	vars["grant_database"] = true
	p.variables(vars)
	p.apply()
	s.AdvanceControllers()
	p.apply()
	if graphObject(t, s, scope, "endpoint", endpoint).Status.Readiness != "ready" {
		t.Fatal("restored graph not ready")
	}
	// Keep nonempty, protected services and independent grants in the configuration.
	s.SetUsage(scope, "database", db, 100)
	s.SetUsage(scope, "bucket", ids["spoke_bucket.assets"], 100)
	vars["deploy"] = false
	p.variables(vars)
	p.apply()
	p.run("plan", "-detailed-exitcode", "-input=false", "-no-color")
	for _, kind := range []string{"container_service", "service_binding", "endpoint"} {
		if s.Count(kind) != 0 {
			t.Fatalf("retained compute: %s", kind)
		}
	}
	for _, kind := range []string{"artifact_registry", "database", "bucket", "secret_reference"} {
		if s.Count(kind) != 1 {
			t.Fatalf("lost independent service: %s", kind)
		}
	}
	if s.Count("grant") != 5 || s.Count("service_account") != 1 || !s.RuntimeAllowed(scope, account, "database", db, "database_connect") {
		t.Fatal("retirement removed independent authority")
	}
	s.Restart(t)
	p.run("plan", "-detailed-exitcode", "-input=false", "-no-color")
}

func TestAccMockPartialServiceGraphRecovery(t *testing.T) {
	_ = mockAcceptance(t)
	binary := buildInstalledProvider(t)
	for _, tc := range []struct{ kind, key, phase string }{{"container_service", "graph-app", "after_accept"}, {"service_binding", "graph-binding-database", "after_complete"}} {
		t.Run(tc.kind, func(t *testing.T) {
			s := mockAcceptance(t)
			p, scope, _, _ := graphConsumer(t, s, binary)
			barrier := s.BlockNextFor(tc.phase, tc.kind, tc.key)
			op := p.killAt(barrier, "apply", "-auto-approve", "-input=false", "-no-color", "-parallelism=1")
			s.Restart(t)
			s.CompleteAll()
			p.apply()
			p.run("plan", "-detailed-exitcode", "-input=false", "-no-color")
			if op.TargetID == "" {
				t.Fatal("barrier missing accepted identity")
			}
			graphObject(t, s, scope, tc.kind, op.TargetID)
			for kind, want := range map[string]int{"project": 2, "environment": 2, "service_account": 1, "grant": 5, "artifact_registry": 1, "database": 1, "bucket": 1, "secret_reference": 1, "container_service": 1, "service_binding": 4, "endpoint": 1} {
				if got := s.Count(kind); got != want {
					t.Fatalf("%s: %d != %d", kind, got, want)
				}
			}
			// Four bootstrap declarations plus fourteen service declarations, each accepted once.
			if s.OperationCount() != 18 {
				t.Fatalf("duplicated effects: %d", s.OperationCount())
			}
		})
	}
}

func TestAccMockServiceProcessRecovery(t *testing.T) {
	_ = mockAcceptance(t)
	binary := buildInstalledProvider(t)
	for _, kind := range []string{"artifact_registry", "database", "secret_reference", "service_binding", "container_service", "endpoint"} {
		for _, action := range []string{"create", "update", "delete"} {
			for _, phase := range []string{"after_accept", "after_complete"} {
				t.Run(kind+"/"+action+"/"+phase, func(t *testing.T) {
					s := mockAcceptance(t)
					d := s.ServiceDependencies(t)
					counts := map[string]int{}
					for _, k := range api.Kinds() {
						counts[k] = s.Count(k)
					}
					p := newConsumer(t, binary, serviceConfig(s, d, kind, "initial"))
					id := ""
					if action != "create" {
						p.apply()
						id = p.resourceIDs()["spoke_"+kind+".test"]
					}
					args := []string{"apply", "-auto-approve", "-input=false", "-no-color"}
					if action == "update" {
						p.config(serviceConfig(s, d, kind, "renamed"))
					}
					if action == "delete" {
						args[0] = "destroy"
					}
					before := s.OperationCount()
					accepted := p.killAt(s.BlockNext(phase), args...)
					s.Restart(t)
					s.CompleteAll()
					p.run(args...)
					if s.OperationCount() != before+1 {
						t.Fatal("duplicated recovered effect")
					}
					if action != "delete" {
						got := p.resourceIDs()["spoke_"+kind+".test"]
						if got != accepted.TargetID || (id != "" && id != got) {
							t.Fatal("recovery lost accepted identity")
						}
						p.run("plan", "-detailed-exitcode", "-input=false", "-no-color")
						p.destroy()
					}
					for k, want := range counts {
						if s.Count(k) != want {
							t.Fatalf("recovery/cleanup changed independent %s", k)
						}
					}
				})
			}
		}
	}
}
