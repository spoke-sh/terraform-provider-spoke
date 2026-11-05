package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

func TestPublicationApplicationUnknownIntentAndScopedIdentity(t *testing.T) {
	valid := staticApplicationModel{Key: types.StringValue("key"), Name: types.StringValue("site"), Protection: types.BoolValue(true), Retention: types.Int64Value(30)}
	for name, change := range map[string]func(*staticApplicationModel){
		"null key":           func(m *staticApplicationModel) { m.Key = types.StringNull() },
		"unknown key":        func(m *staticApplicationModel) { m.Key = types.StringUnknown() },
		"null name":          func(m *staticApplicationModel) { m.Name = types.StringNull() },
		"unknown name":       func(m *staticApplicationModel) { m.Name = types.StringUnknown() },
		"null protection":    func(m *staticApplicationModel) { m.Protection = types.BoolNull() },
		"unknown protection": func(m *staticApplicationModel) { m.Protection = types.BoolUnknown() },
		"null retention":     func(m *staticApplicationModel) { m.Retention = types.Int64Null() },
		"unknown retention":  func(m *staticApplicationModel) { m.Retention = types.Int64Unknown() },
	} {
		t.Run(name, func(t *testing.T) {
			m := valid
			change(&m)
			if _, err := m.policy(); err == nil {
				t.Fatal("unresolved mutation accepted")
			}
		})
	}
	scope := api.Scope{OrganizationID: "example", ProjectID: "saga"}
	for _, id := range []string{"example/saga/0", "example/saga/01", "example/saga/9007199254740992", "foreign/saga/1", "example/saga/nightly/1", ""} {
		if _, err := publicationParse(scope, id); err == nil {
			t.Fatalf("accepted %q", id)
		}
	}
	if _, err := publicationParse(scope, "example/saga/9007199254740991"); err != nil {
		t.Fatal(err)
	}
	for _, scope := range []api.Scope{{OrganizationID: "example"}, {OrganizationID: "example", ProjectID: "saga", EnvironmentID: "nightly"}} {
		if _, _, err := publicationConfigured(&configured{httpConfig: api.Config{Scope: scope}}, false); err == nil {
			t.Fatal("broadened publication scope")
		}
	}
}
