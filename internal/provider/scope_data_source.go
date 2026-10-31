package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSourceWithConfigure = (*scopeDataSource)(nil)

type scopeDataSource struct {
	environment bool
	config      *configured
}

func newProjectDataSource() datasource.DataSource     { return &scopeDataSource{} }
func newEnvironmentDataSource() datasource.DataSource { return &scopeDataSource{environment: true} }
func (d *scopeDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
	if d.environment {
		resp.TypeName = req.ProviderTypeName + "_environment"
	}
}
func (d *scopeDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := map[string]schema.Attribute{
		"id":              schema.StringAttribute{Computed: true, MarkdownDescription: "Stable ID of the explicitly selected scope."},
		"organization_id": schema.StringAttribute{Computed: true, MarkdownDescription: "Owning organization ID."},
		"name":            schema.StringAttribute{Computed: true, MarkdownDescription: "Observed display name."},
	}
	if d.environment {
		attrs["project_id"] = schema.StringAttribute{Computed: true, MarkdownDescription: "Owning project ID."}
	}
	resp.Schema = schema.Schema{MarkdownDescription: "Read the scope explicitly selected in provider configuration. Uses the provisional mock API and never provisions a scope.", Attributes: attrs}
}
func (d *scopeDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *scopeDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	ctx, cancel := context.WithTimeout(ctx, d.config.timeout)
	defer cancel()
	if d.environment {
		e, err := d.config.client.GetEnvironment(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Cannot read environment", err.Error())
			return
		}
		data := struct {
			ID             types.String `tfsdk:"id"`
			OrganizationID types.String `tfsdk:"organization_id"`
			ProjectID      types.String `tfsdk:"project_id"`
			Name           types.String `tfsdk:"name"`
		}{types.StringValue(e.ID), types.StringValue(e.OrganizationID), types.StringValue(e.ProjectID), types.StringValue(e.Name)}
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}
	p, err := d.config.client.GetProject(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Cannot read project", err.Error())
		return
	}
	data := struct {
		ID             types.String `tfsdk:"id"`
		OrganizationID types.String `tfsdk:"organization_id"`
		Name           types.String `tfsdk:"name"`
	}{types.StringValue(p.ID), types.StringValue(p.OrganizationID), types.StringValue(p.Name)}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
