//go:build linux

package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAccMockV2StateMigration(t *testing.T) {
	mockAcceptance(t)
	binary := buildInstalledProvider(t)
	for _, literal := range []bool{false, true} {
		t.Run(fmt.Sprintf("literal_target_%t", literal), func(t *testing.T) {
			s := mockAcceptance(t)
			ledger, err := os.ReadFile("../../testdata/migration/mock-v2.ledger.json")
			if err != nil {
				t.Fatal(err)
			}
			if err = s.LoadV2Ledger(ledger); err != nil {
				t.Fatal(err)
			}
			state, err := os.ReadFile("../../testdata/migration/mock-v2-state.json")
			if err != nil {
				t.Fatal(err)
			}
			config := scopeGraphConfig(s, "baseline")
			if literal {
				var legacy struct {
					Resources []struct {
						Type      string
						Instances []struct{ Attributes struct{ Target string } }
					}
				}
				if err = json.Unmarshal(state, &legacy); err != nil {
					t.Fatal(err)
				}
				for _, r := range legacy.Resources {
					if r.Type == "spoke_grant" {
						config = strings.Replace(config, "spoke_environment.test.target_path", fmt.Sprintf("%q", strings.Replace(r.Instances[0].Attributes.Target, "/v2/", "/v3/", 1)), 1)
					}
				}
			}
			config += fmt.Sprintf(`
provider "spoke" {
 alias = "service"
 endpoint = %q
 organization_id = %q
 project_id = %q
 environment_id = %q
 token_file = %q
 ca_file = %q
}
resource "spoke_bucket" "test" {
 provider = spoke.service
 management_key = "bucket-key"
 name = "baseline"
 deletion_protection = false
}
`, s.HTTP.URL, s.Scope.OrganizationID, s.Scope.ProjectID, s.Scope.EnvironmentID, s.TokenFile, s.CAFile)
			p := newConsumer(t, binary, config)
			if err = os.WriteFile(filepath.Join(p.dir, "terraform.tfstate"), state, 0600); err != nil {
				t.Fatal(err)
			}
			before := s.OperationCount()
			p.run("plan", "-detailed-exitcode", "-input=false", "-no-color")
			p.apply()
			p.run("plan", "-detailed-exitcode", "-input=false", "-no-color")
			if s.OperationCount() != before {
				t.Fatal("migration performed a mutation or replaced a grant")
			}
			s.Restart(t)
			p.run("plan", "-detailed-exitcode", "-input=false", "-no-color")
		})
	}
}
