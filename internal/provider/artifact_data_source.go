package provider

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type artifactDataSource struct{ config *configured }
type artifactModel struct {
	ID             types.String `tfsdk:"id"`
	RegistryID     types.String `tfsdk:"registry_id"`
	Repository     types.String `tfsdk:"repository"`
	Digest         types.String `tfsdk:"digest"`
	Tag            types.String `tfsdk:"tag"`
	ResolvedDigest types.String `tfsdk:"resolved_digest"`
	SizeBytes      types.Int64  `tfsdk:"size_bytes"`
}

func newArtifactDataSource() datasource.DataSource { return &artifactDataSource{} }
func (d *artifactDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_artifact"
}
func (d *artifactDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Resolve exact artifact metadata by immutable digest or mutable tag in a registry/repository. Supply exactly one selector. Reads never build, upload or promote artifacts; pass resolved_digest to a workload to pin its desired image.", Attributes: map[string]schema.Attribute{
		"id":              schema.StringAttribute{Computed: true, MarkdownDescription: "Scope-qualified immutable artifact identity."},
		"registry_id":     schema.StringAttribute{Required: true, Validators: idValidators(), MarkdownDescription: "Exact registry ID in the configured environment."},
		"repository":      schema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.LengthBetween(1, 255)}, MarkdownDescription: "Exact OCI repository path."},
		"digest":          schema.StringAttribute{Optional: true, Validators: []validator.String{stringvalidator.RegexMatches(regexp.MustCompile(`^sha256:[0-9a-f]{64}$`), "must be a sha256 digest")}, MarkdownDescription: "Immutable digest selector. Mutually exclusive with tag."},
		"tag":             schema.StringAttribute{Optional: true, Validators: []validator.String{stringvalidator.RegexMatches(regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9_.-]{0,127}$`), "must be a valid artifact tag")}, MarkdownDescription: "Mutable tag selector. Resolution can change on a later read; existing pinned workloads are not mutated by discovery."},
		"resolved_digest": schema.StringAttribute{Computed: true, MarkdownDescription: "Resolved immutable sha256 digest."},
		"size_bytes":      schema.Int64Attribute{Computed: true, MarkdownDescription: "Observed artifact size from management metadata."},
	}}
}
func (d *artifactDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *artifactDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var m artifactModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, d.config.timeout)
	defer cancel()
	a, err := d.config.client.GetArtifact(ctx, m.RegistryID.ValueString(), m.Repository.ValueString(), m.Digest.ValueString(), m.Tag.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Cannot discover artifact", err.Error())
		return
	}
	m.ID = types.StringValue(a.Scope.Path() + "/artifact_registries/" + a.RegistryID + "/" + a.Repository + "@" + a.Digest)
	m.ResolvedDigest = types.StringValue(a.Digest)
	m.SizeBytes = types.Int64Value(a.SizeBytes)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}
