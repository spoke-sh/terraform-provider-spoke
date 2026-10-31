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
