package provider

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

func TestProtocolSchema(t *testing.T) {
	server := providerserver.NewProtocol6(New("test")())()
	resp, err := server.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range resp.Diagnostics {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("%s: %s", d.Summary, d.Detail)
		}
	}
	if resp.Provider == nil {
		t.Fatal("missing provider schema")
	}
}

func validProviderModel() providerModel {
	return providerModel{
		Endpoint:       types.StringValue("https://example.invalid"),
		OrganizationID: types.StringValue("org-test"), ProjectID: types.StringValue("project-test"), EnvironmentID: types.StringValue("env-test"),
		TokenFile: types.StringNull(), CAFile: types.StringNull(), TimeoutSeconds: types.Int64Null(),
	}
}

func TestConfigureValidation(t *testing.T) {
	t.Setenv("SPOKE_TOKEN", "synthetic-test-token")
	cases := map[string]func(*providerModel){
		"HTTP":               func(m *providerModel) { m.Endpoint = types.StringValue("http://example.invalid") },
		"URL credentials":    func(m *providerModel) { m.Endpoint = types.StringValue("https://secret@example.invalid") },
		"unknown":            func(m *providerModel) { m.ProjectID = types.StringUnknown() },
		"scope":              func(m *providerModel) { m.EnvironmentID = types.StringValue("../another") },
		"timeout":            func(m *providerModel) { m.TimeoutSeconds = types.Int64Value(0) },
		"missing token file": func(m *providerModel) { m.TokenFile = types.StringValue(filepath.Join(t.TempDir(), "absent")) },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			m := validProviderModel()
			change(&m)
			if _, err := configure(m); err == nil {
				t.Fatal("accepted invalid configuration")
			}
		})
	}
	if _, err := configure(validProviderModel()); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SPOKE_TOKEN", "")
	if _, err := configure(validProviderModel()); err == nil {
		t.Fatal("accepted missing credentials")
	}
}

func TestTokenFilePrecedenceAndRedaction(t *testing.T) {
	t.Setenv("SPOKE_TOKEN", "invalid token")
	path := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(path, []byte("synthetic-file-token\n"), 0600); err != nil {
		t.Fatal(err)
	}
	m := validProviderModel()
	m.TokenFile = types.StringValue(path)
	if _, err := configure(m); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("private token"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := configure(m)
	if err == nil || strings.Contains(err.Error(), "private") {
		t.Fatalf("bad credential diagnostic: %v", err)
	}
}

func TestConfigureOrganizationBootstrap(t *testing.T) {
	t.Setenv("SPOKE_TOKEN", "synthetic")
	m := validProviderModel()
	m.ProjectID = types.StringNull()
	m.EnvironmentID = types.StringNull()
	if _, err := configure(m); err != nil {
		t.Fatalf("organization bootstrap: %v", err)
	}
	m.EnvironmentID = types.StringValue("env")
	if _, err := configure(m); err == nil {
		t.Fatal("environment without project accepted")
	}
	m.ProjectID = types.StringValue("project")
	m.EnvironmentID = types.StringNull()
	if _, err := configure(m); err != nil {
		t.Fatalf("project-scoped discovery: %v", err)
	}
}
