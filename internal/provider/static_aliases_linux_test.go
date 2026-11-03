//go:build linux

package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/spoke-sh/terraform-provider-spoke/internal/testutil/mockplatform"
)

func publicationAliases(t *testing.T, f publicationPlatformConfig, binary string) {
	catalog := newConsumer(t, binary, publicationCatalogConfig(f, "alias-catalog"))
	catalog.apply()
	configuration := func(nightlyName string) string {
		result := ""
		for _, environment := range []string{"nightly", "production"} {
			binding := publicationManagedState(t, catalog, "spoke_static_target_binding."+environment)["resource_id"].(string)
			name := "production"
			if environment == "nightly" {
				name = nightlyName
			}
			result += fmt.Sprintf(`
provider "spoke" {
 alias = %q
 endpoint = %q
 organization_id = "example"
 project_id = "saga"
 environment_id = %q
 token_file = %q
 ca_file = %q
}
resource "spoke_static_target" %q {
 provider = spoke.%s
 management_key = "aliased-target"
 binding_id = %q
 name = %q
 deletion_protection = false
 retention_days = 30
}
`, environment, f.Endpoint, environment, f.Tokens[environment], f.CAFile, environment, environment, binding, name)
		}
		return result
	}
	p := newConsumer(t, binary, configuration("nightly"))
	p.apply()
	p.config(configuration("nightly-updated"))
	p.apply()
	p.run("plan", "-detailed-exitcode", "-input=false", "-no-color")
	nightly := publicationManagedState(t, p, "spoke_static_target.nightly")
	production := publicationManagedState(t, p, "spoke_static_target.production")
	if nightly["owner_id"] != "202" || production["owner_id"] != "203" || production["name"] != "production" || nightly["name"] != "nightly-updated" {
		t.Fatal("aliases shared provider scope or policy")
	}
	p.destroy()
	catalog.destroy()
}
func TestAccMockPublicationAliases(t *testing.T) {
	_ = mockAcceptance(t)
	s := mockplatform.NewPublication(t)
	publicationAliases(t, publicationPlatformConfig{Endpoint: s.HTTP.URL, CAFile: s.CAFile, Tokens: s.Tokens}, buildInstalledProvider(t))
}
func TestPublicationPlatformAliases(t *testing.T) {
	f := publicationPlatformFixture(t)
	if os.Getenv("SPOKE_TEST_PROVIDER_MIRROR") == "" {
		t.Fatal("exact artifact required")
	}
	publicationAliases(t, f, buildInstalledProvider(t))
}
