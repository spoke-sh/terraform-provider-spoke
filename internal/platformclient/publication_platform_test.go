package platformclient

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Selected only by scripts/qualify-publication-client.py against a new isolated
// Platform-owned fixture. This is real HTTP client evidence, not installed
// Terraform resource qualification. Ordinary checks do not start native services.
type publicationFixtureConfig struct {
	Endpoint          string            `json:"endpoint"`
	CAFile            string            `json:"ca_file"`
	Tokens            map[string]string `json:"tokens"`
	Barrier           string            `json:"barrier"`
	ExpectedOperation string            `json:"expected_operation"`
}

func publicationFixture(t *testing.T) publicationFixtureConfig {
	t.Helper()
	path := os.Getenv("SPOKE_PUBLICATION_TEST_CONFIG")
	if path == "" {
		t.Skip("explicit Platform publication fixture not selected")
	}
	if !filepath.IsAbs(path) || strings.HasPrefix(filepath.Clean(path), "/nix/store/") {
		t.Fatal("external fixture config required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal("cannot read publication fixture config")
	}
	var config publicationFixtureConfig
	if json.Unmarshal(data, &config) != nil || !filepath.IsAbs(config.CAFile) || !filepath.IsAbs(config.Barrier) {
		t.Fatal("invalid publication fixture config")
	}
	return config
}

func publicationFixtureClient(t *testing.T, config publicationFixtureConfig, role, environment string) *PublicationClient {
	t.Helper()
	token, err := os.ReadFile(config.Tokens[role])
	if err != nil {
		t.Fatal("missing synthetic role credential")
	}
	ca, err := os.ReadFile(config.CAFile)
	if err != nil {
		t.Fatal("missing fixture trust")
	}
	c, err := NewPublication(Config{Endpoint: config.Endpoint, Scope: Scope{OrganizationID: "example", ProjectID: "saga", EnvironmentID: environment}, Token: strings.TrimSpace(string(token)), CAPEM: ca})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func publicationResource(t *testing.T, op PublicationOperation, err error) string {
	t.Helper()
	if err != nil || op.ResourceID == nil || op.Status != "applied" {
		t.Fatalf("publication declaration did not apply: %v", err)
	}
	return *op.ResourceID
}

func TestPublicationPlatformInterruptedCaller(t *testing.T) {
	config := publicationFixture(t)
	c := publicationFixtureClient(t, config, "project", "")
	op, err := c.CreateApplication(context.Background(), "client_crash_create", PublicationPolicy{"client_crash", false, 7})
	_ = publicationResource(t, op, err)
	// A test-only barrier permits actual process death before any caller state
	// is saved. Recovery's next process uses only the original key and intent.
	if os.WriteFile(config.Barrier, []byte(op.ID), 0600) != nil {
		t.Fatal("cannot notify crash barrier")
	}
	time.Sleep(2 * time.Minute)
	t.Fatal("fixture did not kill the independent Go caller")
}

func TestPublicationPlatformRecoverOriginalCaller(t *testing.T) {
	config := publicationFixture(t)
	c := publicationFixtureClient(t, config, "project", "")
	op, err := c.CreateApplication(context.Background(), "client_crash_create", PublicationPolicy{"client_crash", false, 7})
	id := publicationResource(t, op, err)
	if op.ID != config.ExpectedOperation {
		t.Fatal("caller death replaced the original accepted operation")
	}
	app, err := c.Application(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	op, err = c.DeleteApplication(context.Background(), id, "client_crash_delete", app.Revision)
	_ = publicationResource(t, op, err)
}

func TestPublicationPlatformDeclarationLifecycle(t *testing.T) {
	config := publicationFixture(t)
	project := publicationFixtureClient(t, config, "project", "")
	nightly := publicationFixtureClient(t, config, "nightly", "nightly")
	reader := publicationFixtureClient(t, config, "reader", "")
	foreign := publicationFixtureClient(t, config, "foreign", "")
	wrongEnvironment := publicationFixtureClient(t, config, "nightly", "production")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	policy := PublicationPolicy{"client_site", true, 7}
	op, err := project.CreateApplication(ctx, "client_app_create", policy)
	appID := publicationResource(t, op, err)
	if replay, err := project.CreateApplication(ctx, "client_app_create", policy); err != nil || replay.ID != op.ID {
		t.Fatal("exact create replay did not retain its receipt")
	}
	if _, err := project.CreateApplication(ctx, "client_app_create", PublicationPolicy{"changed", true, 7}); err == nil {
		t.Fatal("conflicting create key accepted")
	}
	app, err := reader.Application(ctx, appID)
	if err != nil || app.ID != appID {
		t.Fatal("authorized read/import identity failed")
	}
	if _, err := reader.UpdateApplication(ctx, appID, "client_reader_write", app.Revision, policy); err == nil {
		t.Fatal("reader gained policy authority")
	}
	if _, err := foreign.Application(ctx, appID); err == nil {
		t.Fatal("foreign principal observed application")
	}
	if _, err := wrongEnvironment.Targets(ctx); err == nil {
		t.Fatal("nightly principal observed production")
	}
	if _, err := project.DeleteApplication(ctx, appID, "client_protected_delete", app.Revision); err == nil {
		t.Fatal("protected application deleted")
	}
	policy.DeletionProtection = false
	op, err = project.UpdateApplication(ctx, appID, "client_app_unprotect", app.Revision, policy)
	_ = publicationResource(t, op, err)
	app, err = project.Application(ctx, appID)
	if err != nil {
		t.Fatal(err)
	}
	op, err = project.CreateBinding(ctx, appID, "client_binding", app.Revision, "nightly")
	bindingID := publicationResource(t, op, err)
	binding, err := project.Binding(ctx, bindingID)
	if err != nil || binding.State != "reserved" {
		t.Fatal("project reservation missing")
	}
	op, err = nightly.CreateTarget(ctx, bindingID, "client_target_create", PublicationPolicy{"nightly_site", false, 7})
	targetID := publicationResource(t, op, err)
	target, err := nightly.Target(ctx, targetID)
	if err != nil || target.State != "declared" || target.ApplicationID == nil || *target.ApplicationID != appID {
		t.Fatalf("target binding did not settle: %v", err)
	}
	if _, err := project.DeleteApplication(ctx, appID, "client_parent_delete", app.Revision); err == nil {
		t.Fatal("parent deletion ignored the bound target")
	}
	// An independent client causes policy drift. Old revision fences must fail;
	// the caller then observes and submits a new, explicitly distinct intent.
	external := publicationFixtureClient(t, config, "project", "")
	op, err = external.UpdateApplication(ctx, appID, "client_external_drift", app.Revision, PublicationPolicy{"external_drift", false, 7})
	_ = publicationResource(t, op, err)
	if _, err := project.UpdateApplication(ctx, appID, "client_stale_update", app.Revision, policy); err == nil {
		t.Fatal("stale revision update applied")
	}
	app, err = project.Application(ctx, appID)
	if err != nil || app.Name != "external_drift" {
		t.Fatal("drift was hidden")
	}
	op, err = project.UpdateApplication(ctx, appID, "client_converge", app.Revision, policy)
	_ = publicationResource(t, op, err)
	op, err = nightly.UpdateTarget(ctx, targetID, "client_target_policy", target.Revision, PublicationPolicy{"nightly_renamed", false, 14})
	_ = publicationResource(t, op, err)
	updated, err := nightly.Target(ctx, targetID)
	if err != nil || updated.ID != target.ID || updated.DesiredGeneration != target.DesiredGeneration || updated.Endpoint != nil {
		t.Fatal("target policy changed runtime selection/readiness")
	}
	if apps, err := project.Applications(ctx); err != nil || len(apps) != 1 || apps[0].ID != appID {
		t.Fatalf("application discovery failed: %v", err)
	}
	if bindings, err := project.Bindings(ctx, appID); err != nil || len(bindings) != 1 || bindings[0].TargetID == nil || *bindings[0].TargetID != targetID {
		t.Fatalf("binding discovery failed: %v", err)
	}
	if targets, err := nightly.Targets(ctx); err != nil || len(targets) != 1 || targets[0].ID != targetID {
		t.Fatalf("target discovery failed: %v", err)
	}
	if history, err := project.History(ctx, "all"); err != nil || len(history) < 9 {
		t.Fatalf("multi-page original history failed: count=%d err=%v", len(history), err)
	}
	op, err = nightly.DeleteTarget(ctx, targetID, "client_target_delete", updated.Revision)
	_ = publicationResource(t, op, err)
	if _, err := nightly.Target(ctx, targetID); err == nil {
		t.Fatal("deleted target still present")
	} else {
		var remote *PublicationError
		if !errors.As(err, &remote) || remote.Code != "resource_not_found" {
			t.Fatal("target absence was not authoritative")
		}
	}
	// Native target retirement releases its exact project binding. Observe the
	// release before attempting parent deletion; never infer it from child loss.
	deadline := time.Now().Add(10 * time.Second)
	for {
		bindings, err := project.Bindings(ctx, appID)
		if err != nil {
			t.Fatal(err)
		}
		if len(bindings) == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("target did not release its project hold")
		}
		time.Sleep(50 * time.Millisecond)
	}
	app, err = project.Application(ctx, appID)
	if err != nil {
		t.Fatal(err)
	}
	op, err = project.DeleteApplication(ctx, appID, "client_app_delete", app.Revision)
	_ = publicationResource(t, op, err)
	if apps, err := project.Applications(ctx); err != nil || len(apps) != 0 {
		t.Fatal("application retirement did not settle")
	}
}
