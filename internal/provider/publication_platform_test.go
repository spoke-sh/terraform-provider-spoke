package provider

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

type publicationPlatformConfig struct {
	Endpoint         string            `json:"endpoint"`
	CAFile           string            `json:"ca_file"`
	Tokens           map[string]string `json:"tokens"`
	ControlDirectory string            `json:"control_directory"`
}

func publicationRestartFixture(t *testing.T, f publicationPlatformConfig) {
	t.Helper()
	if !filepath.IsAbs(f.ControlDirectory) {
		t.Fatal("explicit isolated fixture control directory required")
	}
	request, response := filepath.Join(f.ControlDirectory, "restart.request"), filepath.Join(f.ControlDirectory, "restart.complete")
	if err := os.WriteFile(request, []byte("restart"), 0600); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(30 * time.Second)
	for {
		if data, err := os.ReadFile(response); err == nil {
			if string(data) != "restarted" {
				t.Fatal("fixture restart failed")
			}
			if err := os.Remove(response); err != nil {
				t.Fatal(err)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("fixture restart deadline")
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func publicationPlatformFixture(t *testing.T) publicationPlatformConfig {
	t.Helper()
	path := os.Getenv("SPOKE_PUBLICATION_TERRAFORM_CONFIG")
	if path == "" {
		t.Skip("explicit installed Platform publication fixture not selected")
	}
	if !filepath.IsAbs(path) || strings.HasPrefix(path, "/nix/store/") {
		t.Fatal("external fixture configuration required")
	}
	data, err := os.ReadFile(path)
	var c publicationPlatformConfig
	if err != nil || json.Unmarshal(data, &c) != nil || c.Endpoint == "" || !filepath.IsAbs(c.CAFile) {
		t.Fatal("invalid publication fixture configuration")
	}
	return c
}

func publicationPlatformClient(t *testing.T, c publicationPlatformConfig, role, environment string) *api.PublicationClient {
	t.Helper()
	token, err := os.ReadFile(c.Tokens[role])
	if err != nil {
		t.Fatal(err)
	}
	ca, err := os.ReadFile(c.CAFile)
	if err != nil {
		t.Fatal(err)
	}
	client, err := api.NewPublication(api.Config{Endpoint: c.Endpoint, Token: strings.TrimSpace(string(token)), CAPEM: ca, Scope: api.Scope{OrganizationID: "example", ProjectID: "saga", EnvironmentID: environment}})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func TestPublicationPlatformIdentityProtocol(t *testing.T) {
	f := publicationPlatformFixture(t)
	c := publicationPlatformClient(t, f, "project", "")
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	policy := api.PublicationPolicy{Name: "identity-a", RetentionDays: 1}
	key, _ := publicationSuccessor("application", "proof", "")
	created, err := c.CreateApplication(ctx, key, policy)
	if err != nil {
		t.Fatal(err)
	}
	if created.ResourceID == nil {
		t.Fatal("missing resource identity")
	}
	id := *created.ResourceID
	// A caller with no saved resource ID recovers the server-backed incarnation.
	observed, err := publicationInspect(ctx, c, "application", "proof")
	if err != nil || observed.Resource != id || observed.lastKey() != key {
		t.Fatalf("lost-state recovery: %+v %v", observed, err)
	}
	duplicate, err := c.CreateApplication(ctx, key, policy)
	if err != nil || duplicate.ID != created.ID {
		t.Fatal("create replay changed identity", err)
	}
	conflicting := policy
	conflicting.Name = "conflicting-create"
	if _, err := c.CreateApplication(ctx, key, conflicting); err == nil {
		t.Fatal("occupied create slot accepted different intent")
	}
	foreign := publicationPlatformClient(t, f, "foreign", "")
	if _, err := publicationInspect(ctx, foreign, "application", "proof"); err == nil {
		t.Fatal("another principal claimed management identity")
	}
	previous := key
	for _, name := range []string{"identity-b", "identity-a"} {
		value, err := c.Application(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		key, err = publicationSuccessor("application", "proof", previous)
		if err != nil {
			t.Fatal(err)
		}
		policy.Name = name
		operation, err := c.UpdateApplication(ctx, id, key, value.Revision, policy)
		if err != nil {
			t.Fatal(err)
		}
		replayed, err := c.UpdateApplication(ctx, id, key, value.Revision, policy)
		if err != nil || replayed.ID != operation.ID {
			t.Fatal("update retry lost original precondition", err)
		}
		previous = key
	}
	value, err := c.Application(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	deleteKey, _ := publicationSuccessor("application", "proof", previous)
	deleted, err := c.DeleteApplication(ctx, id, deleteKey, value.Revision)
	if err != nil {
		t.Fatal(err)
	}
	deleteReplay, err := c.DeleteApplication(ctx, id, deleteKey, value.Revision)
	if err != nil || deleteReplay.ID != deleted.ID {
		t.Fatal("delete retry replaced original identity", err)
	}
	createKey, _ := publicationSuccessor("application", "proof", deleteKey)
	recreated, err := c.CreateApplication(ctx, createKey, policy)
	if err != nil || recreated.ResourceID == nil || *recreated.ResourceID == id {
		t.Fatal("recreation reused deleted incarnation", err)
	}
	staleKey, _ := publicationSuccessor("application", "proof", "tf1a_proof_00")
	if _, err := c.UpdateApplication(ctx, id, staleKey, value.Revision, conflicting); err == nil {
		t.Fatal("stale state advanced its original fence")
	}
	observed, err = publicationInspect(ctx, c, "application", "proof")
	if err != nil || len(observed.Operations) != 5 || observed.Resource != *recreated.ResourceID {
		t.Fatalf("retained sequence: %+v %v", observed, err)
	}
}
