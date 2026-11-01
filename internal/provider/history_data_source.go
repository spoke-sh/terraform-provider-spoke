package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type historyDataSource struct {
	kind   string
	config *configured
}
type historyModel struct {
	ID       types.String       `tfsdk:"id"`
	ParentID types.String       `tfsdk:"parent_id"`
	Items    []historyItemModel `tfsdk:"items"`
}
type historyItemModel struct {
	ID        types.String `tfsdk:"id"`
	SourceID  types.String `tfsdk:"source_id"`
	Revision  types.Int64  `tfsdk:"revision"`
	Timestamp types.Int64  `tfsdk:"timestamp"`
	State     types.String `tfsdk:"state"`
	Attempt   types.Int64  `tfsdk:"attempt"`
}

func (d *historyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	suffix := "job_executions"
	if d.kind == "backup_policy" {
		suffix = "recovery_points"
	}
	resp.TypeName = req.ProviderTypeName + "_" + suffix
}
func (d *historyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Complete bounded mock-v5 metadata for an exact parent, limited to 2048 retained entries. Reads do not execute jobs or capture backups. A denied, unavailable or incomplete observation fails. Outputs and backup contents never enter state.", Attributes: map[string]schema.Attribute{
		"id":        schema.StringAttribute{Computed: true, MarkdownDescription: "Scope-qualified history identity."},
		"parent_id": schema.StringAttribute{Required: true, Validators: idValidators(), MarkdownDescription: "Exact job or backup policy ID in the configured environment."},
		"items": schema.ListNestedAttribute{Computed: true, MarkdownDescription: "Complete retained metadata, ordered by immutable identity.", NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
			"id":        schema.StringAttribute{Computed: true, MarkdownDescription: "Immutable execution or recovery-point ID."},
			"source_id": schema.StringAttribute{Computed: true, MarkdownDescription: "Originating schedule or source database ID."},
			"revision":  schema.Int64Attribute{Computed: true, MarkdownDescription: "Captured job or database revision."},
			"timestamp": schema.Int64Attribute{Computed: true, MarkdownDescription: "Scheduled occurrence or capture time in Unix seconds."},
			"state":     schema.StringAttribute{Computed: true, MarkdownDescription: "Observed execution outcome or completed recovery point."},
			"attempt":   schema.Int64Attribute{Computed: true, MarkdownDescription: "Execution attempt number; zero for recovery points."},
		}}},
	}}
}
func (d *historyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*configured)
	if !ok {
		resp.Diagnostics.AddError("Invalid provider client", "Expected configured Spoke client.")
		return
	}
	d.config = c
}
func (d *historyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var m historyModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, d.config.timeout)
	defer cancel()
	items, err := d.config.client.History(ctx, d.kind, m.ParentID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Cannot discover history", err.Error())
		return
	}
	m.ID = types.StringValue(d.config.client.Scope().Path() + "/" + d.kind + "/" + m.ParentID.ValueString() + "/history")
	m.Items = []historyItemModel{}
	for _, i := range items {
		m.Items = append(m.Items, historyItemModel{types.StringValue(i.ID), types.StringValue(i.SourceID), types.Int64Value(i.Revision), types.Int64Value(i.Timestamp), types.StringValue(i.State), types.Int64Value(i.Attempt)})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}
