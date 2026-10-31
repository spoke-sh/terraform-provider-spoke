package provider

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
	"github.com/spoke-sh/terraform-provider-spoke/internal/testutil/mockplatform"
)

func mockAcceptance(t *testing.T) *mockplatform.Server {
	t.Helper()
	if os.Getenv("TF_ACC") == "" {
		t.Skip("acceptance tests require TF_ACC")
	}
	if os.Getenv("SPOKE_TEST_TARGET") != "mock" {
		t.Fatal("mock acceptance requires explicit SPOKE_TEST_TARGET=mock")
	}
	if os.Getenv("TF_ACC_TERRAFORM_PATH") == "" {
		t.Fatal("use the pinned Terraform executable via TF_ACC_TERRAFORM_PATH")
	}
	return mockplatform.New(t)
}

func factories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){"spoke": providerserver.NewProtocol6WithError(New("test")())}
}

func providerConfig(s *mockplatform.Server) string {
	return fmt.Sprintf(`
provider "spoke" {
  endpoint = %q
  organization_id = %q
  project_id = %q
  environment_id = %q
  token_file = %q
  ca_file = %q
}
`, s.HTTP.URL, s.Scope.OrganizationID, s.Scope.ProjectID, s.Scope.EnvironmentID, s.TokenFile, s.CAFile)
}

func bucketConfig(s *mockplatform.Server, name, protection string) string {
	return providerConfig(s) + fmt.Sprintf(`
data "spoke_project" "current" {}
data "spoke_environment" "current" {}
resource "spoke_bucket" "test" {
  management_key = "acceptance-bucket"
  name = %q
  %s
}
`, name, protection)
}

func destroyed(s *mockplatform.Server) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if n := s.BucketCount(); n != 0 {
			return fmt.Errorf("fixture retained %d buckets", n)
		}
		return nil
	}
}
func mutations(s *mockplatform.Server) int {
	n := 0
	for _, r := range s.Requests() {
		if r.Method != "GET" {
			n++
		}
	}
	return n
}

func TestAccMockBucketLifecycle(t *testing.T) {
	s := mockAcceptance(t)
	s.LoseNextReply()
	config := bucketConfig(s, "assets", "deletion_protection = false")
	renamed := bucketConfig(s, "renamed", "deletion_protection = false")
	var originalID string
	var beforePlan int
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: factories(), CheckDestroy: destroyed(s), Steps: []resource.TestStep{
		{Config: config, Check: resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr("spoke_bucket.test", "name", "assets"),
			resource.TestCheckResourceAttr("spoke_bucket.test", "revision", "1"),
			resource.TestCheckResourceAttr("data.spoke_project.current", "id", s.Scope.ProjectID),
			resource.TestCheckResourceAttr("data.spoke_environment.current", "id", s.Scope.EnvironmentID),
			func(state *terraform.State) error {
				originalID = state.RootModule().Resources["spoke_bucket.test"].Primary.ID
				if s.BucketCount() != 1 || s.OperationCount() != 1 {
					return fmt.Errorf("lost reply duplicated work")
				}
				return nil
			},
		)},
		{Config: config, PlanOnly: true, PreConfig: func() { beforePlan = mutations(s) }},
		{ResourceName: "spoke_bucket.test", ImportState: true, ImportStateVerify: true, PreConfig: func() {
			if mutations(s) != beforePlan {
				t.Fatal("plan submitted a mutation")
			}
		}},
		{Config: renamed, Check: resource.ComposeAggregateTestCheckFunc(resource.TestCheckResourceAttrPtr("spoke_bucket.test", "id", &originalID), resource.TestCheckResourceAttr("spoke_bucket.test", "revision", "2"))},
		{Config: renamed, PreConfig: func() {
			parts := strings.Split(originalID, "/")
			s.Drift(parts[3], api.BucketSpec{Name: "external-drift"})
		}, Check: resource.ComposeAggregateTestCheckFunc(resource.TestCheckResourceAttrPtr("spoke_bucket.test", "id", &originalID), resource.TestCheckResourceAttr("spoke_bucket.test", "revision", "4"))},
		{Config: renamed, PlanOnly: true},
	}})
}

func TestAccMockBucketDeletionProtection(t *testing.T) {
	s := mockAcceptance(t)
	protected := bucketConfig(s, "assets", "")
	unprotected := bucketConfig(s, "assets", "deletion_protection = false")
	var id string
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: factories(), CheckDestroy: destroyed(s), Steps: []resource.TestStep{
		{Config: protected, Check: resource.ComposeAggregateTestCheckFunc(resource.TestCheckResourceAttr("spoke_bucket.test", "deletion_protection", "true"), func(state *terraform.State) error {
			id = strings.Split(state.RootModule().Resources["spoke_bucket.test"].Primary.ID, "/")[3]
			return nil
		})},
		{Config: protected, Destroy: true, ExpectError: regexp.MustCompile("Bucket deletion protected")},
		{Config: unprotected},
		{Config: unprotected, Destroy: true, PreConfig: func() { s.SetNonempty(id, true) }, ExpectError: regexp.MustCompile("not_empty")},
		{Config: unprotected, PreConfig: func() { s.SetNonempty(id, false) }},
	}})
}

func TestAccMockBucketDeniedReadPreservesState(t *testing.T) {
	s := mockAcceptance(t)
	config := bucketConfig(s, "assets", "deletion_protection = false")
	var originalID string
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: factories(), CheckDestroy: destroyed(s), Steps: []resource.TestStep{
		{Config: config, Check: func(state *terraform.State) error {
			originalID = state.RootModule().Resources["spoke_bucket.test"].Primary.ID
			return nil
		}},
		{Config: config, PlanOnly: true, PreConfig: func() { s.DenyAccess(true) }, ExpectError: regexp.MustCompile("forbidden")},
		{Config: config, PreConfig: func() { s.DenyAccess(false) }, Check: resource.ComposeAggregateTestCheckFunc(resource.TestCheckResourceAttrPtr("spoke_bucket.test", "id", &originalID), func(*terraform.State) error {
			if s.OperationCount() != 1 {
				return fmt.Errorf("denial caused recreation")
			}
			return nil
		})},
	}})
}

func TestAccMockBucketExternalDeletion(t *testing.T) {
	s := mockAcceptance(t)
	config := bucketConfig(s, "assets", "deletion_protection = false")
	var originalID string
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: factories(), CheckDestroy: destroyed(s), Steps: []resource.TestStep{
		{Config: config, Check: func(state *terraform.State) error {
			originalID = state.RootModule().Resources["spoke_bucket.test"].Primary.ID
			return nil
		}},
		{Config: config, PreConfig: func() { s.Remove(strings.Split(originalID, "/")[3]) }, Check: func(state *terraform.State) error {
			if state.RootModule().Resources["spoke_bucket.test"].Primary.ID == originalID {
				return fmt.Errorf("recreation reused a deleted ID")
			}
			return nil
		}},
	}})
}

func TestAccMockBucketValidationAndImportScope(t *testing.T) {
	s := mockAcceptance(t)
	config := bucketConfig(s, "assets", "deletion_protection = false")
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: factories(), CheckDestroy: destroyed(s), Steps: []resource.TestStep{
		{Config: strings.Replace(config, `management_key = "acceptance-bucket"`, `management_key = "../invalid"`, 1), ExpectError: regexp.MustCompile("opaque identifier")},
		{Config: config},
		{ResourceName: "spoke_bucket.test", ImportState: true, ImportStateId: "other-org/project-test/env-test/bucket-1", ExpectError: regexp.MustCompile("does not match")},
	}})
}

func TestImportIdentityValidation(t *testing.T) {
	scope := api.Scope{OrganizationID: "org", ProjectID: "project", EnvironmentID: "env"}
	for _, id := range []string{"bucket", "org/project/env/../bucket", "other/project/env/bucket", "org/project/other/bucket", "org/project/env/%2F"} {
		if _, err := parseID(scope, id); err == nil {
			t.Fatalf("accepted %q", id)
		}
	}
	if id, err := parseID(scope, "org/project/env/bucket"); err != nil || id != "bucket" {
		t.Fatalf("valid import: %s %v", id, err)
	}
}
