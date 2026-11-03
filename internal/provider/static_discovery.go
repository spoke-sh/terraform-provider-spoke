package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

type staticDiscovery struct {
	kind       string
	collection bool
	config     *configured
	client     *api.PublicationClient
}

func publicationTypeName(kind string) string {
	return map[string]string{"application": "static_application", "binding": "static_target_binding", "target": "static_target"}[kind]
}
func (d *staticDiscovery) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + publicationTypeName(d.kind)
	if d.collection {
		resp.TypeName += "s"
	}
}
func publicationDataAttributes(ctx context.Context, kind string) map[string]schema.Attribute {
	var r resource.Resource
	switch kind {
	case "application":
		r = newStaticApplicationResource()
	case "binding":
		r = newStaticBindingResource()
	case "target":
		r = newStaticTargetResource()
	default:
		panic("unknown publication kind")
	}
	var result resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &result)
	attrs := map[string]schema.Attribute{}
	for name, a := range result.Schema.Attributes {
		switch name {
		case "owner_id", "management_key", "operation_key", "admission_revision":
			continue
		}
		switch a := a.(type) {
		case rschema.StringAttribute:
			attrs[name] = schema.StringAttribute{Computed: true, MarkdownDescription: a.MarkdownDescription}
		case rschema.BoolAttribute:
			attrs[name] = schema.BoolAttribute{Computed: true, MarkdownDescription: a.MarkdownDescription}
		case rschema.Int64Attribute:
			attrs[name] = schema.Int64Attribute{Computed: true, MarkdownDescription: a.MarkdownDescription}
		default:
			panic("unsupported publication observation type")
		}
	}
	return attrs
}
func (d *staticDiscovery) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := publicationDataAttributes(ctx, d.kind)
	description := "Observe exact publication declarations through the normal scoped API. Requires read authority, never claims management ownership, submits work or restores runtime heads."
	if d.collection {
		items := schema.ListNestedAttribute{Computed: true, MarkdownDescription: "One complete bounded, authorized snapshot. Partial or stale results are never exposed as absence.", NestedObject: schema.NestedAttributeObject{Attributes: attrs}}
		attrs = map[string]schema.Attribute{"id": schema.StringAttribute{Computed: true, MarkdownDescription: "Explicit scoped collection identity."}, "items": items}
		if d.kind == "binding" {
			attrs["application_id"] = schema.StringAttribute{Required: true, MarkdownDescription: "Exact parent application ID. The API exposes reservations only through this parent collection."}
		}
		description = "Discover a complete bounded publication collection. Reads are observational. A missing parent or denied scope is an error; it cannot prove child absence."
	} else {
		id := attrs["resource_id"].(schema.StringAttribute)
		id.Computed = false
		id.Required = true
		attrs["resource_id"] = id
	}
	resp.Schema = schema.Schema{MarkdownDescription: description, Attributes: attrs}
}
func (d *staticDiscovery) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var err error
	d.config, d.client, err = publicationConfigured(req.ProviderData, d.kind == "target")
	if err != nil {
		resp.Diagnostics.AddError("Invalid publication scope", err.Error())
	}
}
func publicationObservation(scope api.Scope, value any) map[string]attr.Value {
	attrs := map[string]attr.Value{}
	var app *api.PublicationApplication
	id, revision := "", ""
	switch v := value.(type) {
	case api.PublicationApplication:
		app = &v
		id, revision = v.ID, v.Revision
	case api.PublicationBinding:
		id, revision = v.ID, v.Revision
		attrs["application_id"] = types.StringValue(v.ApplicationID)
		attrs["environment_id"] = types.StringValue(v.EnvironmentID)
		attrs["target_id"] = types.StringPointerValue(v.TargetID)
		attrs["state"] = types.StringValue(v.State)
	case api.PublicationTarget:
		app = &v.PublicationApplication
		id, revision = v.ID, v.Revision
		for k, v := range map[string]*string{"application_id": v.ApplicationID, "observed_generation": v.ObservedGeneration, "desired_configuration_digest": v.DesiredConfigurationDigest, "observed_configuration_digest": v.ObservedConfigurationDigest, "observed_bundle_digest": v.ObservedBundleDigest, "bundle_digest": v.BundleDigest, "last_observation_ms": v.LastObservationMS, "endpoint": v.Endpoint, "expires_at_ms": v.ExpiresAtMS} {
			attrs[k] = types.StringPointerValue(v)
		}
		for k, v := range map[string]string{"binding_id": v.BindingID, "state": v.State, "desired_generation": v.DesiredGeneration, "configuration_digest": v.ConfigurationDigest} {
			attrs[k] = types.StringValue(v)
		}
	default:
		panic("unknown publication value")
	}
	if app != nil {
		attrs["name"] = types.StringValue(app.Name)
		attrs["deletion_protection"] = types.BoolValue(app.DeletionProtection)
		attrs["retention_days"] = types.Int64Value(app.RetentionDays)
		attrs["readiness"] = types.StringValue(app.Readiness)
		attrs["reason"] = types.StringValue(app.Reason)
	}
	attrs["id"] = types.StringValue(publicationCanonical(scope, id))
	attrs["resource_id"] = types.StringValue(id)
	attrs["revision"] = types.StringValue(revision)
	return attrs
}
func (d *staticDiscovery) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	ctx, cancel := context.WithTimeout(ctx, d.config.timeout)
	defer cancel()
	var values []any
	var err error
	var parent types.String
	if d.collection {
		switch d.kind {
		case "application":
			var list []api.PublicationApplication
			list, err = d.client.Applications(ctx)
			for _, v := range list {
				values = append(values, v)
			}
		case "binding":
			resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("application_id"), &parent)...)
			if resp.Diagnostics.HasError() {
				return
			}
			var list []api.PublicationBinding
			list, err = d.client.Bindings(ctx, parent.ValueString())
			for _, v := range list {
				values = append(values, v)
			}
		case "target":
			var list []api.PublicationTarget
			list, err = d.client.Targets(ctx)
			for _, v := range list {
				values = append(values, v)
			}
		}
	} else {
		var id types.String
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("resource_id"), &id)...)
		if resp.Diagnostics.HasError() {
			return
		}
		var v any
		switch d.kind {
		case "application":
			v, err = d.client.Application(ctx, id.ValueString())
		case "binding":
			v, err = d.client.Binding(ctx, id.ValueString())
		case "target":
			v, err = d.client.Target(ctx, id.ValueString())
		}
		values = append(values, v)
	}
	if err != nil {
		resp.Diagnostics.AddError("Cannot discover publication declarations", err.Error())
		return
	}
	if !d.collection {
		for key, value := range publicationObservation(d.client.Scope(), values[0]) {
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(key), value)...)
		}
		return
	}
	attrs := publicationDataAttributes(ctx, d.kind)
	itemTypes := map[string]attr.Type{}
	for k, v := range attrs {
		itemTypes[k] = v.GetType()
	}
	items := make([]attr.Value, 0, len(values))
	for _, v := range values {
		item, diags := types.ObjectValue(itemTypes, publicationObservation(d.client.Scope(), v))
		resp.Diagnostics.Append(diags...)
		items = append(items, item)
	}
	list, diags := types.ListValue(types.ObjectType{AttrTypes: itemTypes}, items)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := publicationCanonical(d.client.Scope(), publicationTypeName(d.kind)+"s")
	result := map[string]attr.Value{"id": types.StringValue(id), "items": list}
	if d.kind == "binding" {
		result["application_id"] = parent
		result["id"] = types.StringValue(fmt.Sprintf("%s/%s", id, parent.ValueString()))
	}
	for key, value := range result {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(key), value)...)
	}
}
