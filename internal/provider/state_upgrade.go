package provider

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// Version 1 migrates v2 transport-qualified targets without replacing grants.
// The fixture's explicit ledger migration must precede provider/state migration.
func (r *scopeResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{0: {StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
		var fields map[string]json.RawMessage
		if req.RawState == nil || json.Unmarshal(req.RawState.JSON, &fields) != nil {
			resp.Diagnostics.AddError("Cannot upgrade state", "Expected complete prior JSON state.")
			return
		}
		for _, name := range []string{"target_path", "target"} {
			var value string
			if raw, ok := fields[name]; ok && json.Unmarshal(raw, &value) == nil && strings.HasPrefix(value, "/v2/organizations/") {
				fields[name], _ = json.Marshal(strings.Replace(value, "/v2/organizations/", "/v3/organizations/", 1))
			}
		}
		var schema resource.SchemaResponse
		r.Schema(ctx, resource.SchemaRequest{}, &schema)
		raw, err := json.Marshal(fields)
		if err != nil {
			resp.Diagnostics.AddError("Cannot upgrade state", "Cannot encode prior state.")
			return
		}
		value, err := (&tfprotov6.RawState{JSON: raw}).Unmarshal(schema.Schema.Type().TerraformType(ctx))
		if err != nil {
			resp.Diagnostics.AddError("Cannot upgrade state", "Prior state does not match the migration schema.")
			return
		}
		dynamic, err := tfprotov6.NewDynamicValue(schema.Schema.Type().TerraformType(ctx), value)
		if err != nil {
			resp.Diagnostics.AddError("Cannot upgrade state", "Cannot encode upgraded state.")
			return
		}
		resp.DynamicValue = &dynamic
	}}}
}
