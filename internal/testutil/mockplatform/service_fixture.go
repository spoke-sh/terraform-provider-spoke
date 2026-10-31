package mockplatform

import (
	"context"
	"strings"
	"testing"
	"time"

	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

const TestDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const NextDigest = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

// Provision is test setup through the actual HTTP client, not a private-table write.
func (s *Server) Provision(t testing.TB, kind, key string, spec api.Spec) api.Object {
	t.Helper()
	scope := s.Scope
	if kind == "service_account" || kind == "grant" || kind == "project" {
		scope = api.Scope{OrganizationID: scope.OrganizationID}
	}
	if kind == "environment" {
		scope.EnvironmentID = ""
	}
	c, err := api.New(api.Config{Endpoint: s.HTTP.URL, Scope: scope, Token: s.Token, HTTPClient: s.HTTP.Client()})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	op, err := c.Mutate(ctx, api.Intent{Kind: "create", ResourceKind: kind, ManagementKey: key, Spec: &spec})
	if err != nil {
		t.Fatal(err)
	}
	o, err := c.GetObject(ctx, kind, op.TargetID)
	if err != nil {
		t.Fatal(err)
	}
	return o
}

type ServiceDependencies struct{ Registry, Account, Database, Bucket, Secret, Consumer api.Object }

func (s *Server) ServiceDependencies(t testing.TB) ServiceDependencies {
	t.Helper()
	d := ServiceDependencies{}
	d.Account = s.Provision(t, "service_account", "fixture-account", api.Spec{Name: "runtime"})
	d.Registry = s.Provision(t, "artifact_registry", "fixture-registry", api.Spec{Name: "images", Registry: &api.RegistrySpec{Format: "oci", CapacityBytes: api.DefaultCapacity, RetentionDays: 7}})
	d.Database = s.Provision(t, "database", "fixture-database", api.Spec{Name: "data", Storage: &api.StorageSpec{Profile: "shared", CapacityBytes: api.DefaultCapacity, RetentionDays: 7}})
	d.Bucket = s.Provision(t, "bucket", "fixture-bucket", api.Spec{Name: "files", Storage: &api.StorageSpec{Profile: "object", CapacityBytes: api.DefaultCapacity, RetentionDays: 7}})
	d.Secret = s.Provision(t, "secret_reference", "fixture-secret", api.Spec{Name: "secret", Secret: &api.SecretSpec{ExternalOwner: "external", ExternalID: "credential", Version: "v1"}})
	s.SetSecretVersion(*d.Secret.Secret, true)
	d.Consumer = s.Provision(t, "container_service", "fixture-consumer", d.Spec("container_service"))
	return d
}
func (d ServiceDependencies) Spec(kind string) api.Spec {
	spec := api.Spec{Name: "example"}
	switch kind {
	case "artifact_registry":
		spec.Registry = &api.RegistrySpec{Format: "oci", CapacityBytes: api.DefaultCapacity, RetentionDays: 7}
	case "database", "bucket":
		profile := "shared"
		if kind == "bucket" {
			profile = "object"
		}
		spec.Storage = &api.StorageSpec{Profile: profile, CapacityBytes: api.DefaultCapacity, RetentionDays: 7}
	case "secret_reference":
		spec.Secret = &api.SecretSpec{ExternalOwner: "external", ExternalID: "credential", Version: "v1"}
	case "service_binding":
		spec.Binding = &api.BindingSpec{ConsumerID: d.Consumer.ID, TargetKind: "database", TargetID: d.Database.ID, Slot: "database", Intent: "database_connect"}
	case "container_service":
		spec.Container = &api.ContainerSpec{RegistryID: d.Registry.ID, Repository: "app/web", Digest: TestDigest, ServiceAccountID: d.Account.ID, CPU: 500, Memory: 256, Configuration: map[string]string{"MODE": "test"}, SecretReferences: map[string]string{}, RequiredBindings: []string{}}
	case "endpoint":
		spec.Endpoint = &api.EndpointSpec{ServiceID: d.Consumer.ID, Port: 8080, Exposure: "private", TLS: "managed"}
	default:
		panic("invalid service kind " + kind)
	}
	return spec
}
func (s *Server) RuntimeGrant(t testing.TB, d ServiceDependencies, target api.Object, permission string) api.Object {
	t.Helper()
	return s.Provision(t, "grant", "grant-"+strings.ReplaceAll(permission, "_", "-"), api.Spec{Name: permission, SubjectID: d.Account.ID, Target: objectPath(target.Scope, target.ResourceKind, target.ID), Permission: permission})
}
