//go:build linux

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
	"github.com/spoke-sh/terraform-provider-spoke/internal/testutil/mockplatform"
)

func publicationApplicationConfig(endpoint, ca, token, name string, protection bool) string {
	return fmt.Sprintf(`
provider "spoke" {
 endpoint = %q
 organization_id = "example"
 project_id = "saga"
 token_file = %q
 ca_file = %q
}
resource "spoke_static_application" "test" {
 management_key = "terraform-site"
 name = %q
 deletion_protection = %t
 retention_days = 30
}
data "spoke_static_application" "observed" {
 resource_id = spoke_static_application.test.resource_id
}
`, endpoint, token, ca, name, protection)
}
func publicationResourceState(t *testing.T, p *consumer) map[string]any {
	t.Helper()
	var state struct {
		Resources []struct {
			Mode, Type string
			Instances  []struct{ Attributes map[string]any }
		}
	}
	if err := json.Unmarshal(p.run("state", "pull"), &state); err != nil {
		t.Fatal(err)
	}
	for _, r := range state.Resources {
		if r.Mode == "managed" && r.Type == "spoke_static_application" && len(r.Instances) == 1 {
			return r.Instances[0].Attributes
		}
	}
	return nil
}
func TestAccMockStaticApplicationLifecycle(t *testing.T) {
	_ = mockAcceptance(t)
	s := mockplatform.NewPublication(t)
	binary := buildInstalledProvider(t)
	config := func(name string, protection bool) string {
		return publicationApplicationConfig(s.HTTP.URL, s.CAFile, s.TokenFile, name, protection)
	}
	p := newConsumer(t, binary, config("site-a", true))
	p.run("validate")
	p.apply()
	state := publicationResourceState(t, p)
	id := state["id"].(string)
	resourceID := state["resource_id"].(string)
	if state["readiness"] != "pending" || state["operation_key"] != "tf1a_terraform-site_00" {
		t.Fatal(state)
	}
	count := s.OperationCount()
	p.run("plan", "-detailed-exitcode", "-input=false", "-no-color")
	p.run("refresh", "-input=false", "-no-color")
	if count != s.OperationCount() {
		t.Fatal("plan or refresh mutated API")
	}
	p.run("state", "rm", "spoke_static_application.test")
	p.run("import", "-input=false", "-no-color", "spoke_static_application.test", id)
	p.run("plan", "-detailed-exitcode", "-input=false", "-no-color")
	for _, name := range []string{"site-b", "site-a"} {
		p.config(config(name, true))
		p.apply()
	}
	if s.OperationCount() != count+2 {
		t.Fatal("A B A reused an operation")
	}
	s.Drift(resourceID, "external-drift")
	p.apply()
	if publicationResourceState(t, p)["name"] != "site-a" {
		t.Fatal("drift not reconciled")
	}
	s.Deny(true)
	out, err := p.attempt("refresh", "-input=false", "-no-color")
	s.Deny(false)
	if err == nil || !strings.Contains(string(out), "forbidden") || publicationResourceState(t, p)["id"] != id {
		t.Fatal("denied read lost state")
	}
	if _, err := p.attempt("destroy", "-auto-approve", "-input=false", "-no-color"); err == nil {
		t.Fatal("protected deletion succeeded")
	}
	p.config(config("site-a", false))
	p.apply()
	p.destroy()
	if s.Count() != 0 {
		t.Fatal("application retained after allowed destroy")
	}
	p.apply()
	if publicationResourceState(t, p)["id"] == id {
		t.Fatal("recreation reused deleted incarnation")
	}
	p.destroy()
}
func TestAccMockStaticApplicationProcessRecovery(t *testing.T) {
	_ = mockAcceptance(t)
	binary := buildInstalledProvider(t)
	for _, kind := range []string{"create", "update", "delete"} {
		for _, phase := range []string{"before_accept", "after_accept", "after_complete"} {
			t.Run(kind+"/"+phase, func(t *testing.T) {
				s := mockplatform.NewPublication(t)
				config := func(name string) string {
					return publicationApplicationConfig(s.HTTP.URL, s.CAFile, s.TokenFile, name, false)
				}
				p := newConsumer(t, binary, config("site-a"))
				if kind != "create" {
					p.apply()
				}
				if kind == "update" {
					p.config(config("site-b"))
				}
				b := s.Arm(kind, phase)
				args := []string{"apply", "-auto-approve", "-input=false", "-no-color"}
				if kind == "delete" {
					args[0] = "destroy"
				}
				op := p.killAt(b, args...)
				close(b.Release)
				// A separate service-ledger reload is not the provider-process kill.
				s.Restart()
				p.run(args...)
				want := 1
				if kind != "create" {
					want = 2
				}
				if s.OperationCount() != want {
					t.Fatalf("duplicate mutation after %s: %d", phase, s.OperationCount())
				}
				if phase != "before_accept" && op.ID == "" {
					t.Fatal("barrier preceded durable acceptance")
				}
				if kind != "delete" {
					p.run("plan", "-detailed-exitcode", "-input=false", "-no-color")
					p.destroy()
				}
			})
		}
	}
}

func TestPublicationPlatformApplication(t *testing.T) {
	f := publicationPlatformFixture(t)
	if os.Getenv("SPOKE_TEST_PROVIDER_MIRROR") == "" || os.Getenv("SPOKE_TEST_PROVIDER_VERSION") == "" || os.Getenv("TF_ACC_TERRAFORM_PATH") == "" {
		t.Fatal("exact installed provider mirror/version and Terraform required")
	}
	binary := buildInstalledProvider(t)
	config := func(name string, protection bool) string {
		return publicationApplicationConfig(f.Endpoint, f.CAFile, f.Tokens["project"], name, protection)
	}
	p := newConsumer(t, binary, config("installed-a", true))
	p.run("validate")
	p.apply()
	id := publicationResourceState(t, p)["id"].(string)
	p.run("plan", "-detailed-exitcode", "-input=false", "-no-color")
	p.run("state", "rm", "spoke_static_application.test")
	p.run("import", "-input=false", "-no-color", "spoke_static_application.test", id)
	p.run("plan", "-detailed-exitcode", "-input=false", "-no-color")
	for _, name := range []string{"installed-b", "installed-a"} {
		p.config(config(name, true))
		p.apply()
	}

	// An out-of-band policy edit makes a saved Terraform revision stale. The
	// failed operation is retained; a fresh plan observes it before retrying.
	client := publicationPlatformClient(t, f, "project", "")
	resourceID := publicationResourceState(t, p)["resource_id"].(string)
	p.config(config("planned-change", true))
	saved := filepath.Join(p.dir, "saved.plan")
	p.run("plan", "-input=false", "-no-color", "-out="+saved)
	current, err := client.Application(context.Background(), resourceID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.UpdateApplication(context.Background(), resourceID, "external-drift", current.Revision, api.PublicationPolicy{Name: "external", DeletionProtection: true, RetentionDays: 30})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = p.attempt("apply", "-input=false", "-no-color", saved); err == nil {
		t.Fatal("stale revision applied")
	}
	if publicationResourceState(t, p)["id"] != id {
		t.Fatal("conflict erased state")
	}
	p.apply()
	// A foreign principal cannot refresh or import the existing ownership.
	p.config(strings.ReplaceAll(config("planned-change", true), f.Tokens["project"], f.Tokens["foreign"]))
	if _, err = p.attempt("refresh", "-input=false", "-no-color"); err == nil {
		t.Fatal("foreign refresh succeeded")
	}
	if publicationResourceState(t, p)["id"] != id {
		t.Fatal("denial erased state")
	}
	p.config(config("planned-change", true))
	if _, err := p.attempt("destroy", "-auto-approve", "-input=false", "-no-color"); err == nil {
		t.Fatal("protected destroy succeeded")
	}
	p.config(config("installed-a", false))
	p.apply()
	p.destroy()
	p.apply()
	if publicationResourceState(t, p)["id"] == id {
		t.Fatal("deleted incarnation reused")
	}
	p.destroy()
	// Both replacement orders are supported with a new stable management key.
	p.config(config("replacement", false))
	p.apply()
	for n, before := range []bool{false, true} {
		next := strings.ReplaceAll(config("replacement", false), "terraform-site", fmt.Sprintf("replacement-%d", n))
		if before {
			next = strings.Replace(next, "retention_days = 30", "retention_days = 30\n lifecycle { create_before_destroy = true }", 1)
		}
		previous := publicationResourceState(t, p)["id"]
		p.config(next)
		p.apply()
		if publicationResourceState(t, p)["id"] == previous {
			t.Fatal("replacement reused incarnation")
		}
		p.run("plan", "-detailed-exitcode", "-input=false", "-no-color")
	}
	p.destroy()
}
