package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
	"net/netip"
	"strings"
)

// Cross-field checks defer unknown references until apply, but reject known
// invalid policies before producing any management effect.
func (r *scopeResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	check := func(name string, valid func(string) bool, description string) {
		var v types.String
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root(name), &v)...)
		if !v.IsNull() && !v.IsUnknown() && !valid(v.ValueString()) {
			resp.Diagnostics.AddAttributeError(path.Root(name), "Invalid lifecycle settings", description)
		}
	}
	switch r.kind {
	case "custom_domain":
		check("domain", api.ValidDomain, "Use a lowercase ASCII DNS name of at most 253 bytes.")
	case "volume_attachment":
		check("mount_path", func(v string) bool {
			return len(v) <= 256 && strings.HasPrefix(v, "/data/") && !strings.Contains(v, "..") && !strings.ContainsAny(v, "\x00\r\n\t ")
		}, "Use an absolute path beneath /data/ without traversal or whitespace.")
	case "private_connection":
		check("cidr", func(v string) bool {
			p, e := netip.ParsePrefix(v)
			return e == nil && p == p.Masked() && p.Addr().Is4() && p.Addr().IsPrivate() && p.Bits() >= 16 && p.Bits() <= 30
		}, "Use a canonical private IPv4 prefix from /16 through /30.")
	case "scaling_policy":
		var lo, hi types.Int64
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("min_instances"), &lo)...)
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("max_instances"), &hi)...)
		if !lo.IsUnknown() && !hi.IsUnknown() {
			l, h := int64(1), int64(3)
			if !lo.IsNull() {
				l = lo.ValueInt64()
			}
			if !hi.IsNull() {
				h = hi.ValueInt64()
			}
			if l > h {
				resp.Diagnostics.AddAttributeError(path.Root("max_instances"), "Invalid capacity bounds", "max_instances must be at least min_instances.")
			}
		}
	}
}
