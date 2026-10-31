package platformclient_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
	"github.com/spoke-sh/terraform-provider-spoke/internal/testutil/mockplatform"
)

var serviceKinds = []string{"artifact_registry", "database", "bucket", "secret_reference", "service_binding", "container_service", "endpoint"}

func TestServiceInterruptedCRUD(t *testing.T) {
	for _, kind := range serviceKinds {
		for _, phase := range []string{"before_dispatch", "before_accept", "after_accept", "after_complete"} {
			t.Run(kind+"/"+phase, func(t *testing.T) {
				s := mockplatform.New(t)
				d := s.ServiceDependencies(t)
				c := client(t, s)
				baseline := s.OperationCount()
				spec := d.Spec(kind)
				intent := api.Intent{Kind: "create", ResourceKind: kind, ManagementKey: "owned", Spec: &spec}
				for _, action := range []string{"create", "update", "delete"} {
					intent.Kind = action
					if action == "delete" {
						intent.Spec = nil
					}
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					b := s.BlockNext(phase)
					done := make(chan error, 1)
					go func() { _, err := c.Mutate(ctx, intent); done <- err }()
					select {
					case <-b.Reached:
					case <-ctx.Done():
						cancel()
						t.Fatal("barrier not reached")
					}
					cancel()
					if err := <-done; err == nil {
						t.Fatal("interrupted request reported completion")
					}
					s.Restart(t)
					s.CompleteAll()
					op := mustMutate(t, c, intent)
					if action != "delete" {
						o, err := c.GetObject(context.Background(), kind, op.TargetID)
						if err != nil {
							t.Fatal(err)
						}
						if o.Name != spec.Name {
							t.Fatal("wrong settled intent")
						}
						intent.TargetID = o.ID
						intent.ExpectedRevision = o.Revision
						spec.Name = "updated"
					}
				}
				if s.OperationCount() != baseline+3 {
					t.Fatal("duplicate effect after interruption")
				}
				if _, err := c.GetObject(context.Background(), kind, intent.TargetID); !api.IsCode(err, "resource_not_found") {
					t.Fatalf("delete not authoritative: %v", err)
				}
			})
		}
	}
}
func TestArtifactDiscoveryAndProtection(t *testing.T) {
	s := mockplatform.New(t)
	d := s.ServiceDependencies(t)
	c := client(t, s)
	if err := s.PublishArtifact(s.Scope, d.Registry.ID, "app/web", mockplatform.TestDigest, "latest", 128); err != nil {
		t.Fatal(err)
	}
	before := s.OperationCount()
	a, err := c.GetArtifact(context.Background(), d.Registry.ID, "app/web", "", "latest")
	if err != nil || a.Digest != mockplatform.TestDigest {
		t.Fatalf("digest discovery: %+v %v", a, err)
	}
	if err = s.PublishArtifact(s.Scope, d.Registry.ID, "app/web", mockplatform.NextDigest, "latest", 256); err != nil {
		t.Fatal(err)
	}
	a, err = c.GetArtifact(context.Background(), d.Registry.ID, "app/web", mockplatform.TestDigest, "")
	if err != nil || a.Digest != mockplatform.TestDigest {
		t.Fatal("tag mutation changed pinned digest")
	}
	a, err = c.GetArtifact(context.Background(), d.Registry.ID, "app/web", "", "latest")
	if err != nil || a.Digest != mockplatform.NextDigest {
		t.Fatal("tag not resolved")
	}
	if s.OperationCount() != before {
		t.Fatal("discovery mutated management state")
	}
	for _, args := range [][2]string{{"", ""}, {mockplatform.TestDigest, "latest"}, {"bad", ""}} {
		if _, err = c.GetArtifact(context.Background(), d.Registry.ID, "app/web", args[0], args[1]); err == nil {
			t.Fatal("invalid selector")
		}
	}
	if _, err = c.GetArtifact(context.Background(), d.Registry.ID, "missing", mockplatform.TestDigest, ""); !api.IsCode(err, "resource_not_found") {
		t.Fatal(err)
	}
	if err = s.PublishArtifact(s.Scope, d.Registry.ID, "big", mockplatform.TestDigest, "", api.DefaultCapacity); err == nil {
		t.Fatal("quota ignored")
	}
	spec := d.Registry.Spec
	spec.Registry.RetentionDays = 0
	if _, err = c.Mutate(context.Background(), api.Intent{Kind: "update", ResourceKind: "artifact_registry", ManagementKey: d.Registry.ManagementKey, TargetID: d.Registry.ID, ExpectedRevision: 1, Spec: &spec}); !api.IsCode(err, "dependency_conflict") {
		t.Fatalf("retention: %v", err)
	}
	// A separate retained registry proves content protection without a workload dependency.
	registry := s.Provision(t, "artifact_registry", "retained-registry", d.Spec("artifact_registry"))
	if err = s.PublishArtifact(s.Scope, registry.ID, "app", mockplatform.TestDigest, "", 0); err != nil {
		t.Fatal(err)
	}
	if _, err = c.Mutate(context.Background(), api.Intent{Kind: "delete", ResourceKind: "artifact_registry", ManagementKey: registry.ManagementKey, TargetID: registry.ID, ExpectedRevision: 1}); !api.IsCode(err, "not_empty") {
		t.Fatalf("retained artifacts: %v", err)
	}
}
func TestDataProtectionAndRevision(t *testing.T) {
	for _, kind := range []string{"database", "bucket"} {
		t.Run(kind, func(t *testing.T) {
			s := mockplatform.New(t)
			d := s.ServiceDependencies(t)
			c := client(t, s)
			o := s.Provision(t, kind, "owned-data", d.Spec(kind))
			s.SetUsage(s.Scope, kind, o.ID, 100)
			spec := o.Spec
			spec.Storage.CapacityBytes *= 2
			i := api.Intent{Kind: "update", ResourceKind: kind, ManagementKey: o.ManagementKey, TargetID: o.ID, ExpectedRevision: 99, Spec: &spec}
			if _, err := c.Mutate(context.Background(), i); !api.IsCode(err, "revision_conflict") {
				t.Fatal(err)
			}
			i.ExpectedRevision = 1
			mustMutate(t, c, i)
			o2, err := c.GetObject(context.Background(), kind, o.ID)
			if err != nil || o2.ID != o.ID || o2.Storage.CapacityBytes != 2*api.DefaultCapacity {
				t.Fatalf("growth: %+v %v", o2, err)
			}
			i.ExpectedRevision = 2
			spec.Storage.CapacityBytes = api.DefaultCapacity
			if _, err = c.Mutate(context.Background(), i); !api.IsCode(err, "invalid_request") {
				t.Fatalf("shrink accepted: %v", err)
			}
			spec.Storage.CapacityBytes = 2 * api.DefaultCapacity
			spec.Storage.RetentionDays = 0
			if _, err = c.Mutate(context.Background(), i); !api.IsCode(err, "dependency_conflict") {
				t.Fatalf("retention accepted: %v", err)
			}
			if _, err = c.Mutate(context.Background(), api.Intent{Kind: "delete", ResourceKind: kind, ManagementKey: o.ManagementKey, TargetID: o.ID, ExpectedRevision: 2}); !api.IsCode(err, "not_empty") {
				t.Fatalf("data deleted: %v", err)
			}
		})
	}
}
func TestRuntimePolicyAndControllerObservations(t *testing.T) {
	s := mockplatform.New(t)
	d := s.ServiceDependencies(t)
	c := client(t, s)
	if err := s.PublishArtifact(s.Scope, d.Registry.ID, "app/web", mockplatform.TestDigest, "", 64); err != nil {
		t.Fatal(err)
	}
	binding := s.Provision(t, "service_binding", "db-binding", d.Spec("service_binding"))
	endpoint := s.Provision(t, "endpoint", "route", d.Spec("endpoint"))
	s.AdvanceControllers()
	o, err := c.GetObject(context.Background(), "container_service", d.Consumer.ID)
	if err != nil || o.Status.Reason != "pull_denied" {
		t.Fatalf("implicit pull grant: %+v %v", o.Status, err)
	}
	s.RuntimeGrant(t, d, d.Registry, "artifact_pull")
	s.RuntimeGrant(t, d, d.Database, "manage")
	s.AdvanceControllers()
	o, _ = c.GetObject(context.Background(), "service_binding", binding.ID)
	if o.Status.Reason != "resolution_denied" || s.RuntimeAllowed(s.Scope, d.Account.ID, "database", d.Database.ID, "database_connect") {
		t.Fatal("management grant authorized runtime access")
	}
	grant := s.RuntimeGrant(t, d, d.Database, "database_connect")
	s.AdvanceControllers()
	o, _ = c.GetObject(context.Background(), "container_service", d.Consumer.ID)
	if o.Status.Readiness != "ready" || o.Status.ActiveRevision != o.Status.EffectiveRevision {
		t.Fatal("controller failed readiness")
	}
	s.SetHealthFailure(d.Consumer.ID, "health_failed")
	before, _ := c.GetObject(context.Background(), "container_service", d.Consumer.ID)
	after, _ := c.GetObject(context.Background(), "container_service", d.Consumer.ID)
	if before.Status.Readiness != after.Status.Readiness || after.Status.Readiness != "ready" {
		t.Fatal("GET advanced controller")
	}
	s.AdvanceControllers()
	o, _ = c.GetObject(context.Background(), "container_service", d.Consumer.ID)
	if o.Status.Readiness != "failed" || o.Status.ActiveRevision == 0 {
		t.Fatal("health failure lost prior active revision")
	}
	s.SetHealthFailure(d.Consumer.ID, "")
	s.AdvanceControllers()
	mustMutate(t, rootClient(t, s), api.Intent{Kind: "delete", ResourceKind: "grant", ManagementKey: grant.ManagementKey, TargetID: grant.ID, ExpectedRevision: 1})
	s.AdvanceControllers()
	o, _ = c.GetObject(context.Background(), "endpoint", endpoint.ID)
	if o.Status.Readiness == "ready" {
		t.Fatal("revocation did not invalidate endpoint")
	}
	for _, target := range []api.Object{d.Database, d.Consumer} {
		if _, err := c.Mutate(context.Background(), api.Intent{Kind: "delete", ResourceKind: target.ResourceKind, ManagementKey: target.ManagementKey, TargetID: target.ID, ExpectedRevision: 1}); !api.IsCode(err, "dependency_conflict") {
			t.Fatalf("bound target deletion: %v", err)
		}
	}
	if s.Count("database") != 1 || s.Count("artifact_registry") != 1 {
		t.Fatal("revocation deleted services")
	}
}
func TestServiceWireAndFingerprintValidation(t *testing.T) {
	for _, wire := range []string{`{"name":"x"}`, `{"name":"x","deletion_protection":null}`, `{"name":"x","deletion_protection":false,"registry":{"format":"oci","capacity_bytes":1}}`} {
		if api.ValidateSpecWire([]byte(wire)) == nil {
			t.Fatal("accepted missing required settings")
		}
	}
	spec := api.Spec{Name: "app", Container: &api.ContainerSpec{RegistryID: "r", Repository: "app", Digest: mockplatform.TestDigest, ServiceAccountID: "s", CPU: 100, Memory: 64, Configuration: map[string]string{"a": "1", "b": "2"}, SecretReferences: map[string]string{}, RequiredBindings: []string{}}}
	a := api.Intent{Kind: "create", ResourceKind: "container_service", ManagementKey: "key", Spec: &spec}
	first := a.Fingerprint()
	spec.Container.Configuration = map[string]string{"b": "2", "a": "1"}
	if first != a.Fingerprint() {
		t.Fatal("map order changed fingerprint")
	}
	spec.Container.Digest = mockplatform.NextDigest
	if first == a.Fingerprint() {
		t.Fatal("changed digest reused fingerprint")
	}
	data, _ := json.Marshal(spec)
	if strings.Contains(string(data), "password") {
		t.Fatal("unexpected credential field")
	}
}
