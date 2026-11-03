package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

type staticApplicationDataSource struct {
	config *configured
	client *api.PublicationClient
}
type staticApplicationData struct {
	ID         types.String `tfsdk:"id"`
	ResourceID types.String `tfsdk:"resource_id"`
	Name       types.String `tfsdk:"name"`
	Protection types.Bool   `tfsdk:"deletion_protection"`
	Retention  types.Int64  `tfsdk:"retention_days"`
	Revision   types.String `tfsdk:"revision"`
	Readiness  types.String `tfsdk:"readiness"`
	Reason     types.String `tfsdk:"reason"`
}

func (d *staticApplicationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_static_application"
}
func (d *staticApplicationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Observe an exact application declaration in an explicitly configured project. Requires read authority, never claims management ownership and never provisions hosted delivery.", Attributes: map[string]schema.Attribute{
		"id":                  schema.StringAttribute{Computed: true, MarkdownDescription: "Canonical scoped application identity."},
		"resource_id":         schema.StringAttribute{Required: true, MarkdownDescription: "Exact decimal application ID."},
		"name":                schema.StringAttribute{Computed: true, MarkdownDescription: "Observed application label."},
		"deletion_protection": schema.BoolAttribute{Computed: true, MarkdownDescription: "Observed deletion policy."},
		"retention_days":      schema.Int64Attribute{Computed: true, MarkdownDescription: "Observed retention policy."},
		"revision":            schema.StringAttribute{Computed: true, MarkdownDescription: "Observed decimal declaration revision."},
		"readiness":           schema.StringAttribute{Computed: true, MarkdownDescription: "Observed runtime readiness."},
		"reason":              schema.StringAttribute{Computed: true, MarkdownDescription: "Observed readiness reason."},
	}}
}
func (d *staticApplicationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var err error
	d.config, d.client, err = publicationConfigured(req.ProviderData, false)
	if err != nil {
		resp.Diagnostics.AddError("Invalid publication scope", err.Error())
	}
}
func (d *staticApplicationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var m staticApplicationData
	resp.Diagnostics.Append(req.Config.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, d.config.timeout)
	defer cancel()
	v, err := d.client.Application(ctx, m.ResourceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Cannot read application", err.Error())
		return
	}
	m = staticApplicationData{ID: types.StringValue(publicationCanonical(d.client.Scope(), v.ID)), ResourceID: types.StringValue(v.ID), Name: types.StringValue(v.Name), Protection: types.BoolValue(v.DeletionProtection), Retention: types.Int64Value(v.RetentionDays), Revision: types.StringValue(v.Revision), Readiness: types.StringValue(v.Readiness), Reason: types.StringValue(v.Reason)}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}
