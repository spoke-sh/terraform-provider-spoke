//go:build linux

package provider

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

func TestPublicationPlatformAuthentication(t *testing.T) {
	f := publicationPlatformFixture(t)
	p := newConsumer(t, buildInstalledProvider(t), publicationApplicationConfig(f.Endpoint, f.CAFile, f.Tokens["project"], "auth-a", false))
	p.apply()
	before := publicationResourceState(t, p)
	publicationControlFixture(t, f, "rotate-project")
	p.run("plan", "-detailed-exitcode", "-input=false", "-no-color")
	publicationControlFixture(t, f, "revoke-project")
	if _, err := p.attempt("refresh", "-input=false", "-no-color"); err == nil {
		t.Fatal("revoked refresh succeeded")
	}
	if publicationResourceState(t, p)["id"] != before["id"] {
		t.Fatal("revocation removed state")
	}
	publicationControlFixture(t, f, "restore-project")
	p.config(publicationApplicationConfig(f.Endpoint, f.CAFile, f.Tokens["project"], "auth-b", false))
	p.apply()
	after := publicationResourceState(t, p)
	if after["owner_id"] != before["owner_id"] || after["resource_id"] != before["resource_id"] || after["operation_key"] != "tf1a_terraform-site_01" {
		t.Fatal("rotation changed ownership or mutation identity", after)
	}
	p.destroy()
}

func TestPublicationPlatformDeniedRelease(t *testing.T) {
	f := publicationPlatformFixture(t)
	binary := buildInstalledProvider(t)
	parent := newConsumer(t, binary, publicationApplicationConfig(f.Endpoint, f.CAFile, f.Tokens["project"], "parent", false))
	parent.apply()
	app := publicationResourceState(t, parent)["resource_id"].(string)
	reservation := newConsumer(t, binary, publicationBindingConfig(f, app, "release-hold", "nightly"))
	reservation.apply()
	binding := publicationManagedState(t, reservation, "spoke_static_target_binding.test")["resource_id"].(string)
	childConfig := strings.Split(publicationTargetConfig(f, "nightly", binding, "release-child", "target", false), `data "spoke_static_target"`)[0]
	child := newConsumer(t, binary, childConfig)
	child.apply()
	publicationControlFixture(t, f, "revoke-project")
	if _, err := child.attempt("destroy", "-input=false", "-auto-approve", "-no-color"); err == nil {
		t.Fatal("denied release reported successful settlement")
	}
	// The child's exact authorized tombstone permits refresh to remove only it.
	child.run("apply", "-refresh-only", "-input=false", "-auto-approve", "-no-color")
	if len(child.run("state", "list")) != 0 {
		t.Fatal("authoritative child tombstone was not observed")
	}
	publicationControlFixture(t, f, "restore-project")
	publicationRestartFixture(t, f)
	if _, err := parent.attempt("destroy", "-input=false", "-auto-approve", "-no-color"); err == nil {
		t.Fatal("lost release removed conservative application hold")
	}
	if publicationResourceState(t, parent) == nil {
		t.Fatal("blocked parent lost state")
	}
	reservation.destroy()
	parent.destroy()
}

func TestPublicationPlatformBindingAdmission(t *testing.T) {
	f := publicationPlatformFixture(t)
	binary := buildInstalledProvider(t)
	parent := newConsumer(t, binary, publicationApplicationConfig(f.Endpoint, f.CAFile, f.Tokens["project"], "parent", false))
	parent.apply()
	app := publicationResourceState(t, parent)["resource_id"].(string)
	proxy := newPublicationProxy(t, f)
	caller := f
	caller.Endpoint, caller.CAFile = proxy.server.URL, proxy.ca
	p := newConsumer(t, binary, publicationBindingConfig(caller, app, "admission", "nightly"))
	barrier := proxy.arm("before_accept")
	done := make(chan error, 1)
	go func() { _, err := p.attempt("apply", "-auto-approve", "-input=false", "-no-color"); done <- err }()
	select {
	case <-barrier.Reached:
	case <-time.After(30 * time.Second):
		t.Fatal("admission barrier deadline")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c := publicationPlatformClient(t, f, "project", "")
	value, err := c.Application(ctx, app)
	if err != nil {
		close(barrier.Release)
		t.Fatal(err)
	}
	_, err = c.UpdateApplication(ctx, app, "outside-admission-drift", value.Revision, api.PublicationPolicy{Name: "drift", RetentionDays: 30})
	close(barrier.Release)
	if err != nil {
		t.Fatal(err)
	}
	if err = <-done; err == nil {
		t.Fatal("stale admission fence silently renewed")
	}
	p.apply()
	state := publicationManagedState(t, p, "spoke_static_target_binding.test")
	if state["operation_key"] != "tf1b_admission_01" || state["admission_revision"] != "2" {
		t.Fatal("fresh admission discarded retained refusal", state)
	}
	p.destroy()
	parent.destroy()
}

func TestPublicationPlatformExpiredIdentity(t *testing.T) {
	f := publicationPlatformFixture(t)
	proxy := newPublicationProxy(t, f)
	p := newConsumer(t, buildInstalledProvider(t), publicationApplicationConfig(proxy.server.URL, proxy.ca, f.Tokens["project"], "expiry", false))
	p.apply()
	before := publicationResourceState(t, p)
	publicationControlFixture(t, f, "expire-create")
	proxy.mu.Lock()
	writes := proxy.mutations
	proxy.mu.Unlock()
	if _, err := p.attempt("plan", "-input=false", "-no-color"); err == nil {
		t.Fatal("expired identity authorized a fresh plan")
	}
	if publicationResourceState(t, p)["id"] != before["id"] {
		t.Fatal("expired identity erased state")
	}
	proxy.mu.Lock()
	defer proxy.mu.Unlock()
	if proxy.mutations != writes {
		t.Fatal("expiry inspection admitted a new mutation")
	}
}

func TestPublicationPlatformCapacity(t *testing.T) {
	f := publicationPlatformFixture(t)
	p := newConsumer(t, buildInstalledProvider(t), publicationApplicationConfig(f.Endpoint, f.CAFile, f.Tokens["project"], "capacity", false))
	p.apply()
	before := publicationResourceState(t, p)
	id := before["resource_id"].(string)
	c := publicationPlatformClient(t, f, "project", "")
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	for n := 0; n < 7; n++ {
		if _, err := c.CreateApplication(ctx, fmt.Sprintf("capacity-app-%d", n), api.PublicationPolicy{Name: "filler", RetentionDays: 1}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := c.CreateApplication(ctx, "capacity-app-overflow", api.PublicationPolicy{Name: "overflow", RetentionDays: 1}); err == nil {
		t.Fatal("lifetime application capacity exceeded")
	}
	// Fill retained receipts through real HTTP mutations in this isolated owner.
	for n := 0; ; n++ {
		history, err := c.History(ctx, "terminal")
		if err != nil {
			t.Fatal(err)
		}
		if len(history) == 64 {
			break
		}
		if n >= 64 {
			t.Fatal("receipt budget not enforced")
		}
		value, err := c.Application(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = c.UpdateApplication(ctx, id, fmt.Sprintf("capacity-update-%d", n), value.Revision, api.PublicationPolicy{Name: "capacity", RetentionDays: 30}); err != nil {
			t.Fatal(err)
		}
	}
	p.config(publicationApplicationConfig(f.Endpoint, f.CAFile, f.Tokens["project"], "overflow", false))
	if _, err := p.attempt("apply", "-auto-approve", "-input=false", "-no-color"); err == nil {
		t.Fatal("full owner admitted a new Terraform receipt")
	}
	if publicationResourceState(t, p)["id"] != before["id"] {
		t.Fatal("capacity refusal lost resource identity")
	}
	history, err := c.History(ctx, "terminal")
	if err != nil || len(history) != 64 {
		t.Fatal("capacity reused or omitted retained receipts", err)
	}
}

func TestPublicationPlatformConcurrency(t *testing.T) {
	f := publicationPlatformFixture(t)
	binary := buildInstalledProvider(t)
	config := func(key, name string) string {
		return strings.ReplaceAll(publicationApplicationConfig(f.Endpoint, f.CAFile, f.Tokens["project"], name, false), "terraform-site", key)
	}
	a := newConsumer(t, binary, config("same-intent", "same"))
	b := newConsumer(t, binary, config("same-intent", "same"))
	run := func(p *consumer, done chan<- error) {
		_, err := p.attempt("apply", "-auto-approve", "-input=false", "-no-color")
		done <- err
	}
	done := make(chan error, 2)
	go run(a, done)
	go run(b, done)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if publicationResourceState(t, a)["id"] != publicationResourceState(t, b)["id"] {
		t.Fatal("same intent allocated duplicate resources")
	}
	x := newConsumer(t, binary, config("conflicting", "one"))
	y := newConsumer(t, binary, config("conflicting", "two"))
	go run(x, done)
	go run(y, done)
	first, second := <-done, <-done
	if (first == nil) == (second == nil) {
		t.Fatal("conflicting intents did not select exactly one retained owner intent")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	apps, err := publicationPlatformClient(t, f, "project", "").Applications(ctx)
	if err != nil || len(apps) != 2 {
		t.Fatal("concurrency allocated duplicate declarations", err)
	}
	// Identical principal/key clones are indistinguishable from recovery; normal
	// usage still requires external state locking. This test does not transfer ownership.
}
