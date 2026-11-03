//go:build linux

package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/spoke-sh/terraform-provider-spoke/internal/testutil/mockplatform"
)

func publicationCatalogConfig(f publicationPlatformConfig, name string) string {
	return fmt.Sprintf(`
provider "spoke" {
 endpoint = %q
 organization_id = "example"
 project_id = "saga"
 token_file = %q
 ca_file = %q
}
resource "spoke_static_application" "catalog" {
 management_key = "catalog"
 name = %q
 deletion_protection = false
 retention_days = 30
}
resource "spoke_static_target_binding" "nightly" {
 management_key = "nightly-binding"
 application_id = spoke_static_application.catalog.resource_id
 environment_id = "nightly"
}
resource "spoke_static_target_binding" "production" {
 management_key = "production-binding"
 application_id = spoke_static_application.catalog.resource_id
 environment_id = "production"
}
data "spoke_static_applications" "all" { depends_on = [spoke_static_application.catalog] }
data "spoke_static_target_binding" "nightly" { resource_id = spoke_static_target_binding.nightly.resource_id }
data "spoke_static_target_bindings" "all" {
 application_id = spoke_static_application.catalog.resource_id
 depends_on = [spoke_static_target_binding.nightly, spoke_static_target_binding.production]
}
`, f.Endpoint, f.Tokens["project"], f.CAFile, name)
}
func publicationTargetConfig(f publicationPlatformConfig, environment, binding, key, name string, protection bool) string {
	return fmt.Sprintf(`
provider "spoke" {
 endpoint = %q
 organization_id = "example"
 project_id = "saga"
 environment_id = %q
 token_file = %q
 ca_file = %q
}
resource "spoke_static_target" "test" {
 management_key = %q
 binding_id = %q
 name = %q
 deletion_protection = %t
 retention_days = 30
}
data "spoke_static_target" "observed" { resource_id = spoke_static_target.test.resource_id }
data "spoke_static_targets" "all" { depends_on = [spoke_static_target.test] }
`, f.Endpoint, environment, f.Tokens[environment], f.CAFile, key, binding, name, protection)
}
func publicationManagedState(t *testing.T, p *consumer, address string) map[string]any {
	t.Helper()
	var state struct {
		Resources []struct {
			Mode, Type, Name string
			Instances        []struct{ Attributes map[string]any }
		}
	}
	if err := json.Unmarshal(p.run("state", "pull"), &state); err != nil {
		t.Fatal(err)
	}
	for _, r := range state.Resources {
		if r.Mode == "managed" && r.Type+"."+r.Name == address && len(r.Instances) == 1 {
			return r.Instances[0].Attributes
		}
	}
	return nil
}
func publicationGraphLifecycle(t *testing.T, f publicationPlatformConfig, binary string) {
	t.Helper()
	proxy := newPublicationProxy(t, f)
	f.Endpoint, f.CAFile = proxy.server.URL, proxy.ca
	catalog := newConsumer(t, binary, publicationCatalogConfig(f, "catalog-a"))
	catalog.run("validate")
	catalog.apply()
	bindings := map[string]string{}
	roots := map[string]*consumer{}
	for _, environment := range []string{"nightly", "production"} {
		binding := publicationManagedState(t, catalog, "spoke_static_target_binding."+environment)
		bindings[environment] = binding["resource_id"].(string)
		p := newConsumer(t, binary, publicationTargetConfig(f, environment, bindings[environment], environment+"-target", "site-a", true))
		roots[environment] = p
		p.run("validate")
		p.apply()
		state := publicationManagedState(t, p, "spoke_static_target.test")
		if state["bundle_digest"] != nil || state["endpoint"] != nil || state["desired_generation"] != "0" {
			t.Fatal("declaration invented runtime delivery", state)
		}
		id := state["id"].(string)
		p.run("state", "rm", "spoke_static_target.test")
		p.run("import", "-input=false", "-no-color", "spoke_static_target.test", id)
		p.run("plan", "-detailed-exitcode", "-input=false", "-no-color")
		for _, name := range []string{"site-b", "site-a"} {
			p.config(publicationTargetConfig(f, environment, bindings[environment], environment+"-target", name, true))
			p.apply()
		}
		if _, err := p.attempt("destroy", "-auto-approve", "-input=false", "-no-color"); err == nil {
			t.Fatal("protected target destroyed")
		}
		if state := publicationManagedState(t, p, "spoke_static_target.test"); state["id"] != id {
			t.Fatal("protected destroy lost target")
		}
	}
	// Consumption changes reservation revision/state, not desired ownership.
	catalog.run("refresh", "-input=false", "-no-color")
	before := publicationManagedState(t, catalog, "spoke_static_target_binding.nightly")
	catalog.run("state", "rm", "spoke_static_target_binding.nightly")
	catalog.run("import", "-input=false", "-no-color", "spoke_static_target_binding.nightly", before["id"].(string))
	catalog.config(publicationCatalogConfig(f, "catalog-b"))
	catalog.apply()
	for environment, bid := range bindings {
		if publicationManagedState(t, catalog, "spoke_static_target_binding."+environment)["resource_id"] != bid {
			t.Fatal("application policy replaced reservation")
		}
	}
	if _, err := catalog.attempt("destroy", "-auto-approve", "-input=false", "-no-color"); err == nil {
		t.Fatal("catalog deleted active environment holds")
	}
	if publicationManagedState(t, catalog, "spoke_static_application.catalog") == nil {
		t.Fatal("dependency refusal erased application")
	}
	p := roots["nightly"]
	prior := publicationManagedState(t, p, "spoke_static_target.test")["id"]
	denied := publicationTargetConfig(f, "nightly", bindings["nightly"], "nightly-target", "site-a", true)
	p.config(strings.ReplaceAll(denied, f.Tokens["nightly"], f.Tokens["production"]))
	if _, err := p.attempt("refresh", "-input=false", "-no-color"); err == nil {
		t.Fatal("foreign environment principal refreshed target")
	}
	if publicationManagedState(t, p, "spoke_static_target.test")["id"] != prior {
		t.Fatal("denial erased target")
	}
	p.config(denied)
	// Three independent states share only explicit catalog IDs. Observations must
	// issue no mutation, including after the refused parent deletion.
	writes := proxy.mutationCount()
	for _, p := range []*consumer{catalog, roots["nightly"], roots["production"]} {
		p.run("refresh", "-input=false", "-no-color")
		p.run("plan", "-detailed-exitcode", "-input=false", "-no-color")
		state := string(p.run("state", "pull"))
		for _, file := range f.Tokens {
			token, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(state, strings.TrimSpace(string(token))) {
				t.Fatal("credential entered Terraform state")
			}
		}
	}
	if proxy.mutationCount() != writes {
		t.Fatal("plan/refresh submitted publication work")
	}
	for _, environment := range []string{"nightly", "production"} {
		p := roots[environment]
		p.config(publicationTargetConfig(f, environment, bindings[environment], environment+"-target", "site-a", false))
		p.apply()
		p.destroy()
	}
	catalog.destroy()
}
func TestAccMockPublicationGraph(t *testing.T) {
	_ = mockAcceptance(t)
	s := mockplatform.NewPublication(t)
	publicationGraphLifecycle(t, publicationPlatformConfig{Endpoint: s.HTTP.URL, CAFile: s.CAFile, Tokens: s.Tokens}, buildInstalledProvider(t))
}
func TestPublicationPlatformGraph(t *testing.T) {
	f := publicationPlatformFixture(t)
	if os.Getenv("SPOKE_TEST_PROVIDER_MIRROR") == "" {
		t.Fatal("exact artifact required")
	}
	publicationGraphLifecycle(t, f, buildInstalledProvider(t))
}

func TestAccMockPublicationRuntimeHeadsAreObservational(t *testing.T) {
	_ = mockAcceptance(t)
	s := mockplatform.NewPublication(t)
	f := publicationPlatformConfig{Endpoint: s.HTTP.URL, CAFile: s.CAFile, Tokens: s.Tokens}
	binary := buildInstalledProvider(t)
	catalog := newConsumer(t, binary, publicationCatalogConfig(f, "catalog"))
	catalog.apply()
	binding := publicationManagedState(t, catalog, "spoke_static_target_binding.nightly")["resource_id"].(string)
	config := func(name string) string {
		return publicationTargetConfig(f, "nightly", binding, "observational", name, false)
	}
	p := newConsumer(t, binary, config("site"))
	p.apply()
	before := publicationManagedState(t, p, "spoke_static_target.test")
	id := before["resource_id"].(string)
	bundle := strings.Repeat("a", 64)
	s.ObserveTargetHead("nightly", id, bundle)
	writes := s.OperationCountIn("nightly")
	p.run("refresh", "-input=false", "-no-color")
	p.run("plan", "-detailed-exitcode", "-input=false", "-no-color")
	if s.OperationCountIn("nightly") != writes {
		t.Fatal("runtime observation submitted policy")
	}
	p.config(config("new-policy"))
	p.apply()
	state := publicationManagedState(t, p, "spoke_static_target.test")
	if state["bundle_digest"] != bundle || state["observed_bundle_digest"] != bundle || state["desired_generation"] != "1" {
		t.Fatal("policy restored runtime head", state)
	}
	if _, err := p.attempt("destroy", "-auto-approve", "-input=false", "-no-color"); err == nil {
		t.Fatal("selected target deleted")
	}
	if publicationManagedState(t, p, "spoke_static_target.test")["id"] != before["id"] {
		t.Fatal("blocked selection lost state")
	}
	s.ObserveTargetHead("nightly", id, "")
	p.destroy()
	catalog.destroy()
}
