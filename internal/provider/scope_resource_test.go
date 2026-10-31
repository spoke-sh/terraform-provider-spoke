package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
	"github.com/spoke-sh/terraform-provider-spoke/internal/testutil/mockplatform"
)

func organizationConfig(s *mockplatform.Server) string {
	return fmt.Sprintf(`
provider "spoke" {
 endpoint = %q
 organization_id = %q
 token_file = %q
 ca_file = %q
}
`, s.HTTP.URL, s.Scope.OrganizationID, s.TokenFile, s.CAFile)
}
func scopeGraphConfig(s *mockplatform.Server, name string) string {
	return organizationConfig(s) + fmt.Sprintf(`
resource "spoke_project" "test" {
 management_key = "project-key"
 name = %q
 deletion_protection = false
}
resource "spoke_environment" "test" {
 project_id = spoke_project.test.resource_id
 management_key = "environment-key"
 name = %q
 deletion_protection = false
}
resource "spoke_service_account" "test" {
 management_key = "account-key"
 name = %q
 deletion_protection = false
}
resource "spoke_grant" "test" {
 management_key = "grant-key"
 name = %q
 subject_id = spoke_service_account.test.resource_id
 target = spoke_environment.test.target_path
 permission = "read"
}
data "spoke_projects" "all" { depends_on = [spoke_project.test] }
data "spoke_service_accounts" "all" { depends_on = [spoke_service_account.test] }
data "spoke_grants" "all" { depends_on = [spoke_grant.test] }
`, name, name, name, name)
}
func TestAccMockScopeAccessLifecycle(t *testing.T) {
	s := mockAcceptance(t)
	config := scopeGraphConfig(s, "initial")
	renamed := scopeGraphConfig(s, "renamed")
	ids := map[string]string{}
	kinds := []string{"project", "environment", "service_account", "grant"}
	checkIDs := func(state *terraform.State) error {
		for _, kind := range kinds {
			id := state.RootModule().Resources["spoke_"+kind+".test"].Primary.ID
			if old := ids[kind]; old != "" && old != id {
				return fmt.Errorf("%s replaced identity", kind)
			}
			ids[kind] = id
		}
		return nil
	}
	steps := []resource.TestStep{{Config: config, Check: resource.ComposeAggregateTestCheckFunc(checkIDs,
		resource.TestCheckResourceAttr("data.spoke_projects.all", "items.#", "2"), resource.TestCheckResourceAttr("data.spoke_service_accounts.all", "items.#", "1"), resource.TestCheckResourceAttr("data.spoke_grants.all", "items.#", "1"))}, {Config: config, PlanOnly: true}}
	for _, kind := range kinds {
		steps = append(steps, resource.TestStep{ResourceName: "spoke_" + kind + ".test", ImportState: true, ImportStateVerify: true})
	}
	steps = append(steps, resource.TestStep{Config: renamed, Check: checkIDs}, resource.TestStep{Config: renamed, PreConfig: func() {
		c, err := api.New(api.Config{Endpoint: s.HTTP.URL, Scope: api.Scope{OrganizationID: s.Scope.OrganizationID}, Token: s.Token, HTTPClient: s.HTTP.Client()})
		if err != nil {
			t.Fatal(err)
		}
		for _, kind := range kinds {
			parts := strings.Split(ids[kind], "/")
			client := c
			if kind == "environment" {
				client, err = c.WithScope(api.Scope{OrganizationID: s.Scope.OrganizationID, ProjectID: parts[1]})
				if err != nil {
					t.Fatal(err)
				}
			}
			o, err := client.GetObject(context.Background(), kind, parts[len(parts)-1])
			if err != nil {
				t.Fatal(err)
			}
			spec := o.BucketSpec
			spec.Name = "external"
			s.Drift(o.ID, spec)
		}
	}, Check: checkIDs}, resource.TestStep{Config: renamed, PlanOnly: true, PreConfig: func() { s.DenyAccess(true) }, ExpectError: regexp.MustCompile("forbidden")}, resource.TestStep{Config: renamed, PreConfig: func() { s.DenyAccess(false) }, Check: checkIDs}, resource.TestStep{Config: renamed, PlanOnly: true})
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: factories(), CheckDestroy: func(*terraform.State) error {
		if s.Count("project") != 1 || s.Count("environment") != 1 || s.Count("service_account") != 0 || s.Count("grant") != 0 {
			return fmt.Errorf("did not preserve seeded scopes or retire owned graph")
		}
		return nil
	}, Steps: steps})
}

func singleScopeConfig(s *mockplatform.Server, kind, name string, protected bool) string {
	extra := ""
	deps := ""
	if kind == "environment" {
		extra = fmt.Sprintf("project_id = %q", s.Scope.ProjectID)
	}
	if kind == "grant" {
		deps = `resource "spoke_service_account" "subject" {
 management_key = "subject"
 name = "subject"
 deletion_protection = false
}
`
		extra = fmt.Sprintf("subject_id = spoke_service_account.subject.resource_id\n target = %q\n permission = \"read\"", api.Scope{OrganizationID: s.Scope.OrganizationID}.Path())
	}
	return organizationConfig(s) + deps + fmt.Sprintf(`
resource "spoke_%s" "test" {
 management_key = "managed-test"
 name = %q
 deletion_protection = %t
 %s
}
`, kind, name, protected, extra)
}
func TestAccMockScopeProtectionRecoveryAndAbsence(t *testing.T) {
	for _, kind := range []string{"project", "environment", "service_account", "grant"} {
		t.Run(kind, func(t *testing.T) {
			s := mockAcceptance(t)
			protected := singleScopeConfig(s, kind, "initial", true)
			unprotected := singleScopeConfig(s, kind, "renamed", false)
			address := "spoke_" + kind + ".test"
			var firstID string
			resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: factories(), Steps: []resource.TestStep{
				{Config: protected, PreConfig: func() { s.LoseNextReply() }, Check: func(state *terraform.State) error {
					firstID = state.RootModule().Resources[address].Primary.ID
					return nil
				}},
				{Config: protected, Destroy: true, ExpectError: regexp.MustCompile("deletion protected")},
				{Config: unprotected, PreConfig: func() { s.LoseNextReply() }, Check: resource.TestCheckResourceAttrPtr(address, "id", &firstID)},
				{Config: unprotected, PreConfig: func() { parts := strings.Split(firstID, "/"); s.Remove(parts[len(parts)-1]) }, Check: func(state *terraform.State) error {
					if state.RootModule().Resources[address].Primary.ID == firstID {
						return fmt.Errorf("adopted deleted identity")
					}
					return nil
				}},
				{Config: unprotected, PlanOnly: true},
				{Config: unprotected, Destroy: true, PreConfig: func() { s.LoseNextReply() }},
			}})
		})
	}
}

func collectionConfig(s *mockplatform.Server) string {
	return providerConfig(s) + fmt.Sprintf(`
provider "spoke" {
 alias = "organization"
 endpoint = %q
 organization_id = %q
 token_file = %q
 ca_file = %q
}
provider "spoke" {
 alias = "project"
 endpoint = %q
 organization_id = %q
 project_id = %q
 token_file = %q
 ca_file = %q
}
resource "spoke_bucket" "all" {
 for_each = toset(["one", "two", "three"])
 management_key = each.key
 name = each.key
 deletion_protection = false
}
data "spoke_buckets" "all" { depends_on = [spoke_bucket.all] }
data "spoke_projects" "all" { provider = spoke.organization }
data "spoke_environments" "all" { provider = spoke.project }
data "spoke_service_accounts" "all" { provider = spoke.organization }
data "spoke_grants" "all" { provider = spoke.organization }
`, s.HTTP.URL, s.Scope.OrganizationID, s.TokenFile, s.CAFile, s.HTTP.URL, s.Scope.OrganizationID, s.Scope.ProjectID, s.TokenFile, s.CAFile)
}
