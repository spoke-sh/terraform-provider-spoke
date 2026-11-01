package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Exact service discovery exposes the same typed observations, without ownership adoption.
type serviceDataSource struct {
	kind   string
	config *configured
}

func (d *serviceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + d.kind
}
func (d *serviceDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	var managed resource.SchemaResponse
	(&scopeResource{kind: d.kind}).Schema(ctx, resource.SchemaRequest{}, &managed)
	attrs := map[string]dschema.Attribute{}
	for name, attribute := range managed.Schema.Attributes {
		switch a := attribute.(type) {
		case rschema.StringAttribute:
			attrs[name] = dschema.StringAttribute{Computed: true, MarkdownDescription: a.MarkdownDescription}
		case rschema.Int64Attribute:
			attrs[name] = dschema.Int64Attribute{Computed: true, MarkdownDescription: a.MarkdownDescription}
		case rschema.BoolAttribute:
			attrs[name] = dschema.BoolAttribute{Computed: true, MarkdownDescription: a.MarkdownDescription}
		case rschema.MapAttribute:
			attrs[name] = dschema.MapAttribute{Computed: true, ElementType: a.ElementType, MarkdownDescription: a.MarkdownDescription}
		case rschema.SetAttribute:
			attrs[name] = dschema.SetAttribute{Computed: true, ElementType: a.ElementType, MarkdownDescription: a.MarkdownDescription}
		}
	}
	attrs["resource_id"] = dschema.StringAttribute{Required: true, Validators: idValidators(), MarkdownDescription: "Exact opaque resource ID in the configured environment. Discovery never adopts ownership."}
	resp.Schema = dschema.Schema{MarkdownDescription: "Observational exact-ID " + d.kind + " discovery against the provisional mock-v5 API. No live service is qualified.", Attributes: attrs}
}
func (d *serviceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*configured)
	if !ok {
		resp.Diagnostics.AddError("Invalid provider client", "Expected the configured Spoke API client.")
		return
	}
	d.config = c
}
func (d *serviceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var id types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("resource_id"), &id)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, d.config.timeout)
	defer cancel()
	o, err := d.config.client.GetObject(ctx, d.kind, id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Cannot discover "+d.kind, err.Error())
		return
	}
	resp.Diagnostics.Append((&scopeResource{kind: d.kind}).observe(ctx, &resp.State, o)...)
}
