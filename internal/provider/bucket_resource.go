package provider

import (
	"context"
	"errors"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

var (
	_ resource.Resource                = (*bucketResource)(nil)
	_ resource.ResourceWithConfigure   = (*bucketResource)(nil)
	_ resource.ResourceWithImportState = (*bucketResource)(nil)
)

type bucketResource struct{ config *configured }
type bucketModel struct {
	ID                 types.String `tfsdk:"id"`
	ManagementKey      types.String `tfsdk:"management_key"`
	Name               types.String `tfsdk:"name"`
	DeletionProtection types.Bool   `tfsdk:"deletion_protection"`
	Revision           types.Int64  `tfsdk:"revision"`
}

func newBucketResource() resource.Resource { return &bucketResource{} }
func (r *bucketResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket"
}
func (r *bucketResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Experimental managed bucket for the provisional mock API. No live object storage is provisioned by the test fixture. Deletion protection defaults to true; disable it in an applied configuration before destroying an empty bucket.",
		Attributes: map[string]schema.Attribute{
			"id":                  schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}, MarkdownDescription: "Canonical import identity: `organization_id/project_id/environment_id/bucket_id`."},
			"management_key":      schema.StringAttribute{Required: true, Validators: idValidators(), PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, MarkdownDescription: "Durable, non-secret ownership key. Keep it stable across retries and unique per independently managed resource. Changing it replaces the bucket."},
			"name":                schema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.LengthBetween(1, 63)}, MarkdownDescription: "Mutable display name; never used for resource adoption."},
			"deletion_protection": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true), MarkdownDescription: "Refuse deletion while enabled. Even when disabled, nonempty buckets cannot be deleted."},
			"revision":            schema.Int64Attribute{Computed: true, MarkdownDescription: "Last observed native revision, used to fence concurrent mutations."},
		},
	}
}
func (r *bucketResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*configured)
	if !ok {
		resp.Diagnostics.AddError("Invalid provider client", "Expected the configured Spoke API client.")
		return
	}
	r.config = c
}

func canonicalID(scope api.Scope, id string) string {
	return strings.Join([]string{scope.OrganizationID, scope.ProjectID, scope.EnvironmentID, id}, "/")
}
func parseID(scope api.Scope, id string) (string, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 4 {
		return "", errors.New("bucket ID must be organization_id/project_id/environment_id/bucket_id")
	}
	for _, part := range parts {
		if !api.ValidID(part) {
			return "", errors.New("bucket ID contains an invalid scope or object identifier")
		}
	}
	if parts[0] != scope.OrganizationID || parts[1] != scope.ProjectID || parts[2] != scope.EnvironmentID {
		return "", errors.New("bucket identity does not match the configured organization/project/environment")
	}
	return parts[3], nil
}
func (m *bucketModel) observe(b api.Bucket) {
	m.ID = types.StringValue(canonicalID(b.Scope, b.ID))
	m.ManagementKey = types.StringValue(b.ManagementKey)
	m.Name = types.StringValue(b.Name)
	m.DeletionProtection = types.BoolValue(b.DeletionProtection)
	m.Revision = types.Int64Value(b.Revision)
}

func (r *bucketResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data bucketModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, r.config.timeout)
	defer cancel()
	intent := api.Intent{Kind: "create", ManagementKey: data.ManagementKey.ValueString(), Spec: &api.BucketSpec{Name: data.Name.ValueString(), DeletionProtection: data.DeletionProtection.ValueBool()}}
	op, err := r.config.client.Mutate(ctx, intent)
	if err != nil {
		resp.Diagnostics.AddError("Cannot create bucket", err.Error())
		return
	}
	b, err := r.config.client.GetBucket(ctx, op.TargetID)
	if err != nil {
		resp.Diagnostics.AddError("Cannot observe accepted bucket", "The original operation remains recoverable using management_key. "+err.Error())
		return
	}
	data.observe(b)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *bucketResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data bucketModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, err := parseID(r.config.client.Scope(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid bucket scope", err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(ctx, r.config.timeout)
	defer cancel()
	b, err := r.config.client.GetBucket(ctx, id)
	if api.IsCode(err, "resource_not_found") {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Cannot read bucket", err.Error())
		return
	}
	if !data.ManagementKey.IsNull() && !data.ManagementKey.IsUnknown() && data.ManagementKey.ValueString() != b.ManagementKey {
		resp.Diagnostics.AddError("Bucket ownership changed", "The observed management key differs from the resource in state.")
		return
	}
	data.observe(b)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *bucketResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, prior bucketModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, err := parseID(r.config.client.Scope(), prior.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid bucket scope", err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(ctx, r.config.timeout)
	defer cancel()
	_, err = r.config.client.Mutate(ctx, api.Intent{Kind: "update", ManagementKey: prior.ManagementKey.ValueString(), TargetID: id, ExpectedRevision: prior.Revision.ValueInt64(), Spec: &api.BucketSpec{Name: data.Name.ValueString(), DeletionProtection: data.DeletionProtection.ValueBool()}})
	if err != nil {
		resp.Diagnostics.AddError("Cannot update bucket", err.Error())
		return
	}
	b, err := r.config.client.GetBucket(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Cannot observe updated bucket", err.Error())
		return
	}
	data.observe(b)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *bucketResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data bucketModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if data.DeletionProtection.ValueBool() {
		resp.Diagnostics.AddError("Bucket deletion protected", "Apply deletion_protection = false before destroying this bucket. Nonempty buckets remain protected.")
		return
	}
	id, err := parseID(r.config.client.Scope(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid bucket scope", err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(ctx, r.config.timeout)
	defer cancel()
	_, err = r.config.client.Mutate(ctx, api.Intent{Kind: "delete", ManagementKey: data.ManagementKey.ValueString(), TargetID: id, ExpectedRevision: data.Revision.ValueInt64()})
	if err != nil {
		resp.Diagnostics.AddError("Cannot delete bucket", err.Error())
	}
}

func (r *bucketResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := parseID(r.config.client.Scope(), req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid bucket import identity", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
