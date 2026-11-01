package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

type collectionDataSource struct {
	kind   string
	config *configured
}

func newProjectsDataSource() datasource.DataSource { return &collectionDataSource{kind: "project"} }
func newEnvironmentsDataSource() datasource.DataSource {
	return &collectionDataSource{kind: "environment"}
}
func newBucketsDataSource() datasource.DataSource { return &collectionDataSource{kind: "bucket"} }
func newServiceAccountsDataSource() datasource.DataSource {
	return &collectionDataSource{kind: "service_account"}
}
func newGrantsDataSource() datasource.DataSource { return &collectionDataSource{kind: "grant"} }
func (d *collectionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + api.Collection(d.kind)
}
func (d *collectionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Complete bounded discovery of " + api.Collection(d.kind) + " in the configured parent scope, using the provisional mock-v5 API. An incomplete or denied page fails the entire read; discovery never mutates services.", Attributes: map[string]schema.Attribute{
		"id": schema.StringAttribute{Computed: true, MarkdownDescription: "Exact collection target path."},
		"items": schema.ListNestedAttribute{Computed: true, MarkdownDescription: "Complete snapshot, ordered by opaque ID. An empty list is returned only after complete authorized discovery.", NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Computed: true, MarkdownDescription: "Opaque resource ID."},
			"name":        schema.StringAttribute{Computed: true, MarkdownDescription: "Observed display name."},
			"target_path": schema.StringAttribute{Computed: true, MarkdownDescription: "Exact scoped resource path."},
			"revision":    schema.Int64Attribute{Computed: true, MarkdownDescription: "Observed native revision."},
		}}},
	}}
}
func (d *collectionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *collectionDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	ctx, cancel := context.WithTimeout(ctx, d.config.timeout)
	defer cancel()
	objects, err := d.config.client.List(ctx, d.kind)
	if err != nil {
		resp.Diagnostics.AddError("Cannot discover "+api.Collection(d.kind), err.Error())
		return
	}
	element := types.ObjectType{AttrTypes: map[string]attr.Type{"id": types.StringType, "name": types.StringType, "target_path": types.StringType, "revision": types.Int64Type}}
	values := make([]attr.Value, 0, len(objects))
	for _, o := range objects {
		value, diags := types.ObjectValue(element.AttrTypes, map[string]attr.Value{"id": types.StringValue(o.ID), "name": types.StringValue(o.Name), "target_path": types.StringValue(d.config.client.ObjectPath(o)), "revision": types.Int64Value(o.Revision)})
		resp.Diagnostics.Append(diags...)
		values = append(values, value)
	}
	items, diags := types.ListValue(element, values)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	model := struct {
		ID    types.String `tfsdk:"id"`
		Items types.List   `tfsdk:"items"`
	}{types.StringValue(d.config.client.CollectionPath(d.kind)), items}
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
