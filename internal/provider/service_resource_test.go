package provider

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
	"github.com/spoke-sh/terraform-provider-spoke/internal/testutil/mockplatform"
)

func serviceFieldsConfig(kind string, d mockplatform.ServiceDependencies) string {
	switch kind {
	case "artifact_registry":
		return "retention_days = 7"
	case "database":
		return "profile = \"shared\"\nretention_days = 7"
	case "bucket":
		return "retention_days = 7"
	case "secret_reference":
		return "external_owner = \"external\"\nexternal_id = \"credential\"\nversion = \"v1\""
	case "service_binding":
		return fmt.Sprintf("consumer_id = %q\ntarget_kind = \"database\"\ntarget_id = %q\nslot = \"database\"\nintent = \"database_connect\"", d.Consumer.ID, d.Database.ID)
	case "container_service":
		return fmt.Sprintf("registry_id = %q\nrepository = \"app/web\"\ndigest = %q\nservice_account_id = %q", d.Registry.ID, mockplatform.TestDigest, d.Account.ID)
	case "endpoint":
		return fmt.Sprintf("service_id = %q\nport = 8080\nexposure = \"private\"\ntls_policy = \"managed\"", d.Consumer.ID)
	}
	panic("unknown kind")
}
func serviceConfig(s *mockplatform.Server, d mockplatform.ServiceDependencies, kind, name string) string {
	return providerConfig(s) + fmt.Sprintf(`
resource "spoke_%s" "test" {
 management_key = "acceptance-service"
 name = %q
 deletion_protection = false
 %s
}
data "spoke_%s" "one" { resource_id = spoke_%s.test.resource_id }
data "spoke_%s" "all" { depends_on = [spoke_%s.test] }
`, kind, name, serviceFieldsConfig(kind, d), kind, kind, api.Collection(kind), kind)
}
func testServiceLifecycle(t *testing.T, kind string) {
	s := mockAcceptance(t)
	d := s.ServiceDependencies(t)
	baseline := s.Count(kind)
	config := serviceConfig(s, d, kind, "initial")
	renamed := serviceConfig(s, d, kind, "renamed")
	address := "spoke_" + kind + ".test"
	var id string
	var before int
	identity := func(state *terraform.State) error {
		current := state.RootModule().Resources[address].Primary.ID
		if id != "" && current != id {
			return fmt.Errorf("replaced identity")
		}
		id = current
		return nil
	}
	s.LoseNextReply()
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: factories(), CheckDestroy: func(*terraform.State) error {
		if s.Count(kind) != baseline {
			return fmt.Errorf("deleted independent objects or retained owned object")
		}
		return nil
	}, Steps: []resource.TestStep{
		{Config: config, Check: resource.ComposeAggregateTestCheckFunc(identity, resource.TestCheckResourceAttr(address, "revision", "1"), resource.TestCheckResourceAttr("data.spoke_"+kind+".one", "name", "initial"), resource.TestCheckResourceAttr("data.spoke_"+api.Collection(kind)+".all", "items.#", fmt.Sprint(baseline+1)))},
		{Config: config, PlanOnly: true, PreConfig: func() { before = mutations(s) }},
		{ResourceName: address, ImportState: true, ImportStateVerify: true, PreConfig: func() {
			if mutations(s) != before {
				t.Fatal("plan mutated service")
			}
		}},
		{Config: renamed, PreConfig: func() { s.LoseNextReply() }, Check: resource.ComposeAggregateTestCheckFunc(identity, resource.TestCheckResourceAttr(address, "revision", "2"))},
		{Config: renamed, PreConfig: func() {
			o, ok := s.Object(s.Scope, kind, strings.Split(id, "/")[3])
			if !ok {
				t.Fatal("missing fixture object")
			}
			spec := o.Spec
			spec.Name = "external-drift"
			s.Drift(o.ID, spec)
		}, Check: resource.ComposeAggregateTestCheckFunc(identity, resource.TestCheckResourceAttr(address, "revision", "4"))},
		{Config: renamed, PlanOnly: true, PreConfig: func() { s.DenyAccess(true) }, ExpectError: regexp.MustCompile("forbidden")},
		{Config: renamed, PreConfig: func() { s.DenyAccess(false) }, Check: identity},
		{Config: renamed, PreConfig: func() { s.Remove(strings.Split(id, "/")[3]) }, Check: func(state *terraform.State) error {
			newID := state.RootModule().Resources[address].Primary.ID
			if id == newID {
				return fmt.Errorf("authoritative absence did not recreate")
			}
			id = newID
			return nil
		}},
	}})
}
func TestAccMockArtifactRegistryLifecycle(t *testing.T) { testServiceLifecycle(t, "artifact_registry") }

func TestAccMockArtifactDiscovery(t *testing.T) {
	s := mockAcceptance(t)
	d := s.ServiceDependencies(t)
	if err := s.PublishArtifact(s.Scope, d.Registry.ID, "app/web", mockplatform.TestDigest, "release", 128); err != nil {
		t.Fatal(err)
	}
	base := providerConfig(s) + fmt.Sprintf(`
data "spoke_artifact" "test" {
 registry_id = %q
 repository = "app/web"
 %%s
}
`, d.Registry.ID)
	byTag := fmt.Sprintf(base, `tag = "release"`)
	byDigest := fmt.Sprintf(base, fmt.Sprintf("digest = %q", mockplatform.TestDigest))
	before := mutations(s)
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: factories(), Steps: []resource.TestStep{
		{Config: byTag, Check: resource.TestCheckResourceAttr("data.spoke_artifact.test", "resolved_digest", mockplatform.TestDigest)},
		{Config: byTag, PreConfig: func() {
			if err := s.PublishArtifact(s.Scope, d.Registry.ID, "app/web", mockplatform.NextDigest, "release", 256); err != nil {
				t.Fatal(err)
			}
		}, Check: resource.TestCheckResourceAttr("data.spoke_artifact.test", "resolved_digest", mockplatform.NextDigest)},
		{Config: byDigest, Check: resource.TestCheckResourceAttr("data.spoke_artifact.test", "resolved_digest", mockplatform.TestDigest)},
		{Config: fmt.Sprintf(base, `tag = "missing"`), ExpectError: regexp.MustCompile("resource_not_found")},
		{Config: byDigest, PreConfig: func() { s.DenyAccess(true) }, ExpectError: regexp.MustCompile("forbidden")},
		{Config: byDigest, PreConfig: func() { s.DenyAccess(false) }},
	}})
	if mutations(s) != before {
		t.Fatal("artifact lookup mutated management state")
	}
}

func TestAccMockDatabaseLifecycle(t *testing.T)       { testServiceLifecycle(t, "database") }
func TestAccMockExpandedBucketLifecycle(t *testing.T) { testServiceLifecycle(t, "bucket") }

func TestAccMockDataSettingsProtection(t *testing.T) {
	for _, kind := range []string{"database", "bucket"} {
		t.Run(kind, func(t *testing.T) {
			s := mockAcceptance(t)
			d := s.ServiceDependencies(t)
			config := serviceConfig(s, d, kind, "data")
			address := "spoke_" + kind + ".test"
			var id string
			expanded := strings.Replace(config, "retention_days = 7", "retention_days = 14\ncapacity_bytes = 2147483648", 1)
			reduced := strings.Replace(expanded, "capacity_bytes = 2147483648", "capacity_bytes = 1073741824", 1)
			resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: factories(), Steps: []resource.TestStep{
				{Config: config, Check: func(state *terraform.State) error { id = state.RootModule().Resources[address].Primary.ID; return nil }},
				{Config: expanded, Check: resource.ComposeAggregateTestCheckFunc(resource.TestCheckResourceAttrPtr(address, "id", &id), resource.TestCheckResourceAttr(address, "capacity_bytes", "2147483648"))},
				{Config: reduced, ExpectError: regexp.MustCompile("invalid_request")},
				{Config: expanded, PreConfig: func() { s.SetUsage(s.Scope, kind, strings.Split(id, "/")[3], 100) }},
				{Config: strings.Replace(expanded, "retention_days = 14", "retention_days = 0", 1), ExpectError: regexp.MustCompile("dependency_conflict")},
				{Config: expanded, Destroy: true, ExpectError: regexp.MustCompile("not_empty")},
				{Config: expanded, PreConfig: func() { s.SetUsage(s.Scope, kind, strings.Split(id, "/")[3], 0) }},
			}})
		})
	}
}

func TestAccMockSecretReferenceLifecycle(t *testing.T) { testServiceLifecycle(t, "secret_reference") }
func TestAccMockServiceBindingLifecycle(t *testing.T)  { testServiceLifecycle(t, "service_binding") }

func TestAccMockSecretReferenceRotation(t *testing.T) {
	s := mockAcceptance(t)
	d := s.ServiceDependencies(t)
	config := serviceConfig(s, d, "secret_reference", "reference")
	rotated := strings.Replace(config, `version = "v1"`, `version = "v2"`, 1)
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: factories(), Steps: []resource.TestStep{
		{Config: config, Check: resource.TestCheckResourceAttr("spoke_secret_reference.test", "readiness", "pending")},
		{Config: config, PreConfig: func() { s.AdvanceControllers() }, Check: resource.TestCheckResourceAttr("spoke_secret_reference.test", "resolved_version", "v1")},
		{Config: rotated, Check: resource.ComposeAggregateTestCheckFunc(resource.TestCheckResourceAttr("spoke_secret_reference.test", "version", "v2"), resource.TestCheckResourceAttr("spoke_secret_reference.test", "resolved_version", "v1"), resource.TestCheckResourceAttr("spoke_secret_reference.test", "readiness", "pending"))},
		{Config: rotated, PreConfig: func() { s.AdvanceControllers() }, Check: resource.TestCheckResourceAttr("spoke_secret_reference.test", "reason", "dependency_unavailable")},
		{Config: rotated, PreConfig: func() {
			s.SetSecretVersion(api.SecretSpec{ExternalOwner: "external", ExternalID: "credential", Version: "v2"}, true)
			s.AdvanceControllers()
		}, Check: resource.TestCheckResourceAttr("spoke_secret_reference.test", "resolved_version", "v2")},
	}})
}

func TestAccMockBindingRequiresExplicitGrant(t *testing.T) {
	s := mockAcceptance(t)
	d := s.ServiceDependencies(t)
	config := serviceConfig(s, d, "service_binding", "binding")
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: factories(), Steps: []resource.TestStep{
		{Config: config},
		{Config: config, PreConfig: func() { s.AdvanceControllers() }, Check: resource.TestCheckResourceAttr("spoke_service_binding.test", "reason", "resolution_denied")},
		{Config: config, PreConfig: func() { s.RuntimeGrant(t, d, d.Database, "database_connect"); s.AdvanceControllers() }, Check: resource.TestCheckResourceAttr("spoke_service_binding.test", "readiness", "ready")},
	}})
	if s.Count("database") != 1 || s.Count("grant") != 1 || !s.RuntimeAllowed(s.Scope, d.Account.ID, "database", d.Database.ID, "database_connect") {
		t.Fatal("detaching binding removed independent service/grant")
	}
}

func TestAccMockContainerServiceLifecycle(t *testing.T) { testServiceLifecycle(t, "container_service") }
func TestAccMockEndpointLifecycle(t *testing.T)         { testServiceLifecycle(t, "endpoint") }

func TestAccMockServiceValidation(t *testing.T) {
	s := mockAcceptance(t)
	d := s.ServiceDependencies(t)
	for _, tc := range []struct{ kind, from, to, error string }{
		{"container_service", mockplatform.TestDigest, "app:latest", "sha256"},
		{"container_service", "repository = \"app/web\"", "repository = null", "required"},
		{"container_service", "deletion_protection = false", "deletion_protection = false\ncpu_millis = 99", "100"},
		{"endpoint", "port = 8080", "port = 65536", "65535"},
		{"endpoint", "exposure = \"private\"", "exposure = null", "required"},
		{"endpoint", "tls_policy = \"managed\"", "tls_policy = \"disabled\"", "managed"},
	} {
		t.Run(tc.kind+"/"+tc.error, func(t *testing.T) {
			config := strings.Replace(serviceConfig(s, d, tc.kind, "invalid"), tc.from, tc.to, 1)
			before := mutations(s)
			resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: factories(), Steps: []resource.TestStep{{Config: config, PlanOnly: true, ExpectError: regexp.MustCompile("(?i)" + tc.error)}}})
			if mutations(s) != before {
				t.Fatal("invalid configuration mutated platform")
			}
		})
	}
}

func TestAccMockServiceProtectionAndReplacement(t *testing.T) {
	for _, kind := range []string{"artifact_registry", "database", "secret_reference", "service_binding", "container_service", "endpoint"} {
		t.Run(kind, func(t *testing.T) {
			s := mockAcceptance(t)
			d := s.ServiceDependencies(t)
			unprotected := serviceConfig(s, d, kind, "protected")
			protected := strings.Replace(unprotected, "deletion_protection = false", "deletion_protection = true", 1)
			address := "spoke_" + kind + ".test"
			var id string
			steps := []resource.TestStep{
				{Config: protected, Check: func(state *terraform.State) error { id = state.RootModule().Resources[address].Primary.ID; return nil }},
				{Config: protected, Destroy: true, ExpectError: regexp.MustCompile("deletion protected")},
			}
			if kind == "database" {
				replacement := strings.Replace(protected, `profile = "shared"`, `profile = "dedicated"`, 1)
				steps = append(steps, resource.TestStep{Config: replacement, ExpectError: regexp.MustCompile("deletion protected")})
			}
			steps = append(steps, resource.TestStep{Config: unprotected, Check: resource.TestCheckResourceAttrPtr(address, "id", &id)})
			resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: factories(), Steps: steps})
		})
	}
}
