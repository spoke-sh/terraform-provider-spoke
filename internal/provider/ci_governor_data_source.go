package provider

import (
	"context"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

type governorDataSource struct {
	collection bool
	config     *configured
	client     *api.GovernorClient
}

func (d *governorDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ci_governor"
	if d.collection {
		resp.TypeName += "s"
	}
}
func governorDataAttributes() map[string]schema.Attribute {
	attrs := map[string]schema.Attribute{}
	for name, a := range governorAttributes() {
		switch name {
		case "operation_id", "operation_fingerprint", "operation_sequence", "operation_status":
			continue
		}
		switch a := a.(type) {
		case rschema.StringAttribute:
			attrs[name] = schema.StringAttribute{Computed: true, MarkdownDescription: a.MarkdownDescription}
		case rschema.Int64Attribute:
			attrs[name] = schema.Int64Attribute{Computed: true, MarkdownDescription: a.MarkdownDescription}
		case rschema.BoolAttribute:
			attrs[name] = schema.BoolAttribute{Computed: true, MarkdownDescription: a.MarkdownDescription}
		case rschema.ListAttribute:
			attrs[name] = schema.ListAttribute{Computed: true, ElementType: a.ElementType, MarkdownDescription: a.MarkdownDescription}
		default:
			panic("unsupported governor observation type")
		}
	}
	return attrs
}
func (d *governorDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := governorDataAttributes()
	if d.collection {
		attrs = map[string]schema.Attribute{"id": schema.StringAttribute{Computed: true, MarkdownDescription: "Configured organization/project/environment collection identity."}, "governors": schema.ListNestedAttribute{Computed: true, MarkdownDescription: "Complete bounded snapshot ordered by stable ID. Partial or stale collections return diagnostics.", NestedObject: schema.NestedAttributeObject{Attributes: attrs}}}
	} else {
		attrs["id"] = schema.StringAttribute{Required: true, MarkdownDescription: "Canonical organization/project/environment/governor_id in the configured scope."}
	}
	resp.Schema = schema.Schema{Attributes: attrs, MarkdownDescription: "Experimental observational discovery through provisional ci-governor-v1. Reads never resume requests, claim ownership or settle work. Declaration observations do not qualify runner execution, warming, handoff or billing."}
}
func (d *governorDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var err error
	d.config, d.client, err = governorConfigured(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Invalid governor scope", err.Error())
	}
}
func governorObservation(ctx context.Context, scope api.Scope, g api.Governor) map[string]attr.Value {
	m := governorObserved(ctx, scope, g, api.GovernorOperation{})
	return map[string]attr.Value{"id": m.ID, "resource_id": m.ResourceID, "owner_id": m.Owner, "management_key": m.Key, "name": m.Name, "admission_policy_id": m.Admission, "budget_policy_id": m.Budget, "cache_policy_id": m.Cache, "execution_profile_ids": m.Profiles, "max_active_jobs": m.Jobs, "max_allocated_vcpus": m.CPUs, "max_warm_runners": m.Warm, "max_job_seconds": m.JobSeconds, "max_queue_seconds": m.QueueSeconds, "retirement_policy": m.Retirement, "deletion_protection": m.Protection, "revision": m.Revision, "active_revision": m.ActiveRevision, "readiness": m.Readiness, "blockers": m.Blockers, "active_jobs": m.ActiveJobs, "allocated_vcpus": m.AllocatedCPUs, "unsettled_usage": m.Usage}
}
func (d *governorDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	ctx, cancel := context.WithTimeout(ctx, d.config.timeout)
	defer cancel()
	if !d.collection {
		var id types.String
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("id"), &id)...)
		if resp.Diagnostics.HasError() {
			return
		}
		resourceID, err := governorParse(d.config.httpConfig.Scope, id.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid governor discovery ID", err.Error())
			return
		}
		g, err := d.client.Governor(ctx, resourceID)
		if err != nil {
			resp.Diagnostics.AddError("Cannot discover governor", err.Error())
			return
		}
		for name, v := range governorObservation(ctx, d.config.httpConfig.Scope, g) {
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(name), v)...)
		}
		return
	}
	values, err := d.client.Governors(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Cannot discover governors", err.Error())
		return
	}
	sort.Slice(values, func(i, j int) bool { return values[i].ID < values[j].ID })
	itemTypes := map[string]attr.Type{}
	for k, a := range governorDataAttributes() {
		itemTypes[k] = a.GetType()
	}
	items := make([]attr.Value, 0, len(values))
	for _, g := range values {
		v, ds := types.ObjectValue(itemTypes, governorObservation(ctx, d.config.httpConfig.Scope, g))
		resp.Diagnostics.Append(ds...)
		items = append(items, v)
	}
	list, ds := types.ListValue(types.ObjectType{AttrTypes: itemTypes}, items)
	resp.Diagnostics.Append(ds...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), governorCanonical(d.config.httpConfig.Scope, "governors"))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("governors"), list)...)
}
