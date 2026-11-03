//go:build linux

package provider

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/spoke-sh/terraform-provider-spoke/internal/testutil/mockplatform"
)

func publicationBindingConfig(f publicationPlatformConfig, application, key, environment string) string {
	return fmt.Sprintf(`
provider "spoke" {
 endpoint = %q
 organization_id = "example"
 project_id = "saga"
 token_file = %q
 ca_file = %q
}
resource "spoke_static_target_binding" "test" {
 management_key = %q
 application_id = %q
 environment_id = %q
}
`, f.Endpoint, f.Tokens["project"], f.CAFile, key, application, environment)
}
func TestAccMockPublicationChildProcessRecovery(t *testing.T) {
	_ = mockAcceptance(t)
	binary := buildInstalledProvider(t)
	for _, resource := range []string{"binding", "target"} {
		for _, kind := range []string{"create", "update", "delete"} {
			if resource == "binding" && kind == "update" {
				continue
			}
			for _, phase := range []string{"before_accept", "after_accept", "after_complete"} {
				t.Run(resource+"/"+kind+"/"+phase, func(t *testing.T) {
					s := mockplatform.NewPublication(t)
					f := publicationPlatformConfig{Endpoint: s.HTTP.URL, CAFile: s.CAFile, Tokens: s.Tokens}
					var p, parent *consumer
					environment := ""
					baseline := 0
					var changed string
					if resource == "binding" {
						parent = newConsumer(t, binary, publicationApplicationConfig(f.Endpoint, f.CAFile, f.Tokens["project"], "parent", false))
						parent.apply()
						app := publicationResourceState(t, parent)["resource_id"].(string)
						p = newConsumer(t, binary, publicationBindingConfig(f, app, "reservation", "nightly"))
						baseline = s.OperationCount()
					} else {
						parent = newConsumer(t, binary, publicationCatalogConfig(f, "parent"))
						parent.apply()
						binding := publicationManagedState(t, parent, "spoke_static_target_binding.nightly")["resource_id"].(string)
						p = newConsumer(t, binary, publicationTargetConfig(f, "nightly", binding, "target", "site-a", false))
						changed = publicationTargetConfig(f, "nightly", binding, "target", "site-b", false)
						environment = "nightly"
					}
					if kind != "create" {
						p.apply()
					}
					if kind == "update" {
						p.config(changed)
					}
					b := s.Arm(kind, phase)
					args := []string{"apply", "-auto-approve", "-input=false", "-no-color"}
					if kind == "delete" {
						args[0] = "destroy"
					}
					op := p.killAt(b, args...)
					close(b.Release)
					s.Restart()
					p.run(args...)
					want := baseline + 1
					if kind != "create" {
						want++
					}
					if s.OperationCountIn(environment) != want {
						t.Fatalf("recovery duplicated %s: got %d want %d", resource, s.OperationCountIn(environment), want)
					}
					if phase != "before_accept" && op.ID == "" {
						t.Fatal("barrier preceded durable admission")
					}
					if kind != "delete" {
						p.run("plan", "-detailed-exitcode", "-input=false", "-no-color")
						p.destroy()
					}
					parent.destroy()
				})
			}
		}
	}
}
func TestAccMockPublicationBindingAdmissionFence(t *testing.T) {
	_ = mockAcceptance(t)
	s := mockplatform.NewPublication(t)
	f := publicationPlatformConfig{Endpoint: s.HTTP.URL, CAFile: s.CAFile, Tokens: s.Tokens}
	binary := buildInstalledProvider(t)
	parent := newConsumer(t, binary, publicationApplicationConfig(f.Endpoint, f.CAFile, f.Tokens["project"], "parent", false))
	parent.apply()
	app := publicationResourceState(t, parent)["resource_id"].(string)
	p := newConsumer(t, binary, publicationBindingConfig(f, app, "fenced", "nightly"))
	// Drift after the provider captured the parent revision but before the
	// server accepts the request. A retry must not renew this in-flight fence.
	barrier := s.Arm("create", "before_accept")
	done := make(chan error, 1)
	go func() { _, err := p.attempt("apply", "-auto-approve", "-input=false", "-no-color"); done <- err }()
	select {
	case <-barrier.Reached:
	case <-time.After(30 * time.Second):
		t.Fatal("admission barrier deadline")
	}
	s.Drift(app, "new-revision")
	close(barrier.Release)
	if err := <-done; err == nil {
		t.Fatal("binding renewed stale admission fence")
	}
	p.apply()
	state := publicationManagedState(t, p, "spoke_static_target_binding.test")
	if state["operation_key"] != "tf1b_fenced_01" || state["admission_revision"] != "2" {
		t.Fatal("new plan did not retain refusal before re-admission", state)
	}
	p.config(publicationBindingConfig(f, app, "fenced", "production"))
	if _, err := p.attempt("plan", "-input=false", "-no-color"); err == nil {
		t.Fatal("changed live reservation without a new management identity")
	}
	p.config(publicationBindingConfig(f, app, "fenced", "nightly"))
	p.destroy()
	parent.destroy()
}

func TestPublicationPlatformChildRecovery(t *testing.T) {
	f := publicationPlatformFixture(t)
	if os.Getenv("SPOKE_TEST_PROVIDER_MIRROR") == "" {
		t.Fatal("exact artifact required")
	}
	resource, kind := os.Getenv("SPOKE_PUBLICATION_RECOVERY_RESOURCE"), os.Getenv("SPOKE_PUBLICATION_RECOVERY_KIND")
	if (resource != "binding" && resource != "target") || (kind != "create" && kind != "update" && kind != "delete") || resource == "binding" && kind == "update" {
		t.Fatal("explicit supported scenario required")
	}
	binary := buildInstalledProvider(t)
	// Fresh owner per resource/action group keeps all retained IDs below bounds.
	parent := newConsumer(t, binary, publicationApplicationConfig(f.Endpoint, f.CAFile, f.Tokens["project"], "parent", false))
	parent.apply()
	app := publicationResourceState(t, parent)["resource_id"].(string)
	for n, phase := range []string{"before_accept", "after_accept", "after_complete"} {
		t.Run(phase, func(t *testing.T) {
			proxy := newPublicationProxy(t, f)
			caller := f
			caller.Endpoint, caller.CAFile = proxy.server.URL, proxy.ca
			key := fmt.Sprintf("%s-%s-%d", resource, kind, n)
			environment := ""
			var p, reservation *consumer
			var changed string
			if resource == "binding" {
				p = newConsumer(t, binary, publicationBindingConfig(caller, app, key, "nightly"))
			} else {
				reservation = newConsumer(t, binary, publicationBindingConfig(f, app, key+"-b", "nightly"))
				reservation.apply()
				binding := publicationManagedState(t, reservation, "spoke_static_target_binding.test")["resource_id"].(string)
				p = newConsumer(t, binary, publicationTargetConfig(caller, "nightly", binding, key, "site-a", false))
				changed = publicationTargetConfig(caller, "nightly", binding, key, "site-b", false)
				environment = "nightly"
			}
			if kind != "create" {
				p.apply()
			}
			if kind == "update" {
				p.config(changed)
			}
			b := proxy.arm(phase)
			args := []string{"apply", "-auto-approve", "-input=false", "-no-color"}
			if kind == "delete" {
				args[0] = "destroy"
			}
			op := p.killAt(b, args...)
			close(b.Release)
			if phase != "before_accept" && op.ID == "" {
				t.Fatal("missing retained receipt")
			}
			publicationRestartFixture(t, f)
			if resource == "binding" && kind == "create" && phase != "before_accept" {
				client := publicationPlatformClient(t, f, "project", "")
				current, err := client.Application(t.Context(), app)
				if err != nil {
					t.Fatal(err)
				}
				policy := current.PublicationPolicy
				policy.Name = fmt.Sprintf("drift-%d", n)
				if _, err := client.UpdateApplication(t.Context(), app, fmt.Sprintf("external-%d", n), current.Revision, policy); err != nil {
					t.Fatal(err)
				}
			}
			p.run(args...)
			role := "project"
			if environment != "" {
				role = environment
			}
			i, err := publicationInspect(t.Context(), publicationPlatformClient(t, f, role, environment), resource, key)
			want := 1
			if kind != "create" {
				want = 2
			}
			if err != nil || len(i.Operations) != want {
				t.Fatalf("child recovery: %+v %v", i, err)
			}
			if kind != "delete" {
				p.run("plan", "-detailed-exitcode", "-input=false", "-no-color")
				p.destroy()
			}
			if reservation != nil {
				reservation.destroy()
			}
		})
	}
	parent.destroy()
}
