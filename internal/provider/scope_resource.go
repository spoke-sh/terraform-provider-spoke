package provider

import (
	"context"
	"errors"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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

// The lifecycle envelope is shared; schema and parent/spec rules remain explicit per kind.
type scopeResource struct {
	kind   string
	config *configured
}

var _ resource.ResourceWithConfigure = (*scopeResource)(nil)
var _ resource.ResourceWithImportState = (*scopeResource)(nil)

func newProjectResource() resource.Resource        { return &scopeResource{kind: "project"} }
func newEnvironmentResource() resource.Resource    { return &scopeResource{kind: "environment"} }
func newServiceAccountResource() resource.Resource { return &scopeResource{kind: "service_account"} }
func newGrantResource() resource.Resource          { return &scopeResource{kind: "grant"} }
func (r *scopeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + r.kind
}
func (r *scopeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	stable := []planmodifier.String{stringplanmodifier.UseStateForUnknown()}
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	attrs := map[string]schema.Attribute{
		"id":                  schema.StringAttribute{Computed: true, PlanModifiers: stable, MarkdownDescription: "Canonical import ID: `organization_id/resource_id`, or `organization_id/project_id/environment_id` for an environment."},
		"resource_id":         schema.StringAttribute{Computed: true, PlanModifiers: stable, MarkdownDescription: "Opaque ID for references and explicit provider target assertions."},
		"target_path":         schema.StringAttribute{Computed: true, PlanModifiers: stable, MarkdownDescription: "Exact management target path, usable by explicit grants."},
		"owner_id":            schema.StringAttribute{Computed: true, PlanModifiers: stable, MarkdownDescription: "Authenticated management principal. Import does not transfer ownership."},
		"management_key":      schema.StringAttribute{Required: true, Validators: idValidators(), PlanModifiers: replace, MarkdownDescription: "Stable, non-secret recovery key, unique per independently managed resource."},
		"name":                schema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.LengthBetween(1, 63)}, MarkdownDescription: "Mutable display name; never an adoption key."},
		"deletion_protection": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(r.kind != "grant"), MarkdownDescription: "Refuse deletion while enabled. Dependent resources restrict deletion even when protection is disabled. Defaults to true except for grants."},
		"revision":            schema.Int64Attribute{Computed: true, MarkdownDescription: "Observed native revision used for mutation preconditions."},
	}
	if r.kind == "environment" {
		attrs["project_id"] = schema.StringAttribute{Required: true, Validators: idValidators(), PlanModifiers: replace, MarkdownDescription: "Exact parent project ID. Can reference a project resource with an organization-scoped provider."}
	}
	if r.kind == "grant" {
		attrs["subject_id"] = schema.StringAttribute{Required: true, Validators: idValidators(), PlanModifiers: replace, MarkdownDescription: "Exact service-account ID in this organization. Credential issuance remains external."}
		attrs["target"] = schema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.LengthBetween(1, 2048)}, PlanModifiers: replace, MarkdownDescription: "Existing object's target_path or `/v4/organizations/{organization_id}`. Authority includes descendants. Changing the target replaces the owned grant."}
		attrs["permission"] = schema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.OneOf(api.Permissions()...)}, MarkdownDescription: "Explicit permission. manage includes read; no grant is created implicitly by a binding or login."}
	}
	for name, attribute := range serviceAttributes(r.kind) {
		attrs[name] = attribute
	}
	resp.Schema = schema.Schema{Version: 2, MarkdownDescription: "Experimental " + r.kind + " lifecycle against the provisional mock-v4 API. Requires organization-scoped configuration (or a matching project assertion for environments). No production platform service is qualified. Deletion never cascades to independent children or grants.", Attributes: attrs}
	if api.ServiceKind(r.kind) || r.kind == "bucket" {
		resp.Schema.MarkdownDescription = "Experimental " + r.kind + " management lifecycle against mock-v4. Requires a known environment scope. Management settlement and runtime readiness are separate; no live service is qualified. Import preserves principal ownership. Deletion refuses protection and independent dependencies."
		identity := attrs["id"].(schema.StringAttribute)
		identity.MarkdownDescription = "Canonical import identity: `organization_id/project_id/environment_id/resource_id`."
		attrs["id"] = identity
	}
}
func (r *scopeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

type attributeReader interface {
	GetAttribute(context.Context, path.Path, any) diag.Diagnostics
}
type attributeWriter interface {
	SetAttribute(context.Context, path.Path, any) diag.Diagnostics
}
type managedModel struct {
	ID, ResourceID, TargetPath, OwnerID, Key, Name, ProjectID, SubjectID, Target, Permission types.String
	Protection                                                                               types.Bool
	Revision                                                                                 types.Int64
	Settings                                                                                 api.Spec
}

func (r *scopeResource) readModel(ctx context.Context, source attributeReader) (managedModel, diag.Diagnostics) {
	var m managedModel
	var diags diag.Diagnostics
	fields := map[string]any{"id": &m.ID, "resource_id": &m.ResourceID, "target_path": &m.TargetPath, "owner_id": &m.OwnerID, "management_key": &m.Key, "name": &m.Name, "deletion_protection": &m.Protection, "revision": &m.Revision}
	if r.kind == "environment" {
		fields["project_id"] = &m.ProjectID
	}
	if r.kind == "grant" {
		fields["subject_id"] = &m.SubjectID
		fields["target"] = &m.Target
		fields["permission"] = &m.Permission
	}
	for key, value := range fields {
		diags.Append(source.GetAttribute(ctx, path.Root(key), value)...)
	}
	settings, extra := readServiceSettings(ctx, source, r.kind)
	m.Settings = settings
	diags.Append(extra...)
	return m, diags
}
func (r *scopeResource) client(m managedModel) (*api.Client, error) {
	if api.ServiceKind(r.kind) || r.kind == "bucket" {
		if !r.config.client.Scope().AllowsKind(r.kind) {
			return nil, errors.New("service requires a known organization/project/environment scope")
		}
		return r.config.client, nil
	}
	scope := api.Scope{OrganizationID: r.config.client.Scope().OrganizationID}
	if r.kind == "environment" {
		if m.ProjectID.IsUnknown() || !api.ValidID(m.ProjectID.ValueString()) {
			return nil, errors.New("environment requires a known parent project ID")
		}
		scope.ProjectID = m.ProjectID.ValueString()
	}
	return r.config.client.WithScope(scope)
}
func (r *scopeResource) spec(m managedModel) *api.Spec {
	spec := m.Settings
	spec.Name = m.Name.ValueString()
	spec.DeletionProtection = m.Protection.ValueBool()
	if r.kind == "grant" {
		spec.SubjectID = m.SubjectID.ValueString()
		spec.Target = m.Target.ValueString()
		spec.Permission = m.Permission.ValueString()
	}
	return &spec
}
func managedID(o api.Object) string {
	if api.ServiceKind(o.ResourceKind) || o.ResourceKind == "bucket" {
		return canonicalID(o.Scope, o.ID)
	}
	if o.ResourceKind == "environment" {
		return strings.Join([]string{o.OrganizationID, o.ProjectID, o.ID}, "/")
	}
	return o.OrganizationID + "/" + o.ID
}
func (r *scopeResource) parse(m managedModel) (string, error) {
	if api.ServiceKind(r.kind) || r.kind == "bucket" {
		return parseID(r.config.client.Scope(), m.ID.ValueString())
	}
	parts := strings.Split(m.ID.ValueString(), "/")
	expected := 2
	if r.kind == "environment" {
		expected = 3
	}
	if len(parts) != expected {
		return "", errors.New("invalid canonical resource ID")
	}
	for _, part := range parts {
		if !api.ValidID(part) {
			return "", errors.New("invalid scope or resource identifier")
		}
	}
	if parts[0] != r.config.client.Scope().OrganizationID || (r.kind == "environment" && parts[1] != m.ProjectID.ValueString()) {
		return "", errors.New("resource identity does not match its parent scope")
	}
	return parts[len(parts)-1], nil
}
func (r *scopeResource) observe(ctx context.Context, state attributeWriter, o api.Object) diag.Diagnostics {
	var diags diag.Diagnostics
	attrs := map[string]any{"id": types.StringValue(managedID(o)), "resource_id": types.StringValue(o.ID), "target_path": types.StringValue(o.Scope.Path() + "/" + api.Collection(o.ResourceKind) + "/" + o.ID), "owner_id": types.StringValue(o.OwnerID), "management_key": types.StringValue(o.ManagementKey), "name": types.StringValue(o.Name), "deletion_protection": types.BoolValue(o.DeletionProtection), "revision": types.Int64Value(o.Revision)}
	if r.kind == "environment" {
		attrs["project_id"] = types.StringValue(o.ProjectID)
	}
	if r.kind == "grant" {
		attrs["subject_id"] = types.StringValue(o.SubjectID)
		attrs["target"] = types.StringValue(o.Target)
		attrs["permission"] = types.StringValue(o.Permission)
	}
	for key, value := range observedServiceSettings(o) {
		attrs[key] = value
	}
	for key, value := range attrs {
		diags.Append(state.SetAttribute(ctx, path.Root(key), value)...)
	}
	return diags
}
func (r *scopeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	m, diags := r.readModel(ctx, req.Plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, r.config.timeout)
	defer cancel()
	c, err := r.client(m)
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource parent", err.Error())
		return
	}
	op, err := c.Mutate(ctx, api.Intent{ResourceKind: r.kind, Kind: "create", ManagementKey: m.Key.ValueString(), Spec: r.spec(m)})
	if err != nil {
		resp.Diagnostics.AddError("Cannot create "+r.kind, err.Error())
		return
	}
	o, err := c.GetObject(ctx, r.kind, op.TargetID)
	if err != nil {
		resp.Diagnostics.AddError("Cannot observe accepted resource", "Recover using the original management_key. "+err.Error())
		return
	}
	resp.Diagnostics.Append(r.observe(ctx, &resp.State, o)...)
}
func (r *scopeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	m, diags := r.readModel(ctx, req.State)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, r.config.timeout)
	defer cancel()
	c, err := r.client(m)
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource parent", err.Error())
		return
	}
	id, err := r.parse(m)
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource identity", err.Error())
		return
	}
	o, err := c.GetObject(ctx, r.kind, id)
	if api.IsCode(err, "resource_not_found") {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Cannot read "+r.kind, err.Error())
		return
	}
	if (!m.Key.IsNull() && !m.Key.IsUnknown() && m.Key.ValueString() != o.ManagementKey) || (!m.OwnerID.IsNull() && !m.OwnerID.IsUnknown() && m.OwnerID.ValueString() != o.OwnerID) {
		resp.Diagnostics.AddError("Resource ownership changed", "The observed owner or management key differs from state.")
		return
	}
	if err = c.AssertOwner(ctx, o.OwnerID); err != nil {
		resp.Diagnostics.AddError("Cannot adopt resource ownership", err.Error())
		return
	}
	resp.Diagnostics.Append(r.observe(ctx, &resp.State, o)...)
}
func (r *scopeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	m, diags := r.readModel(ctx, req.Plan)
	resp.Diagnostics.Append(diags...)
	prior, diags := r.readModel(ctx, req.State)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, r.config.timeout)
	defer cancel()
	c, err := r.client(prior)
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource parent", err.Error())
		return
	}
	id, err := r.parse(prior)
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource identity", err.Error())
		return
	}
	_, err = c.Mutate(ctx, api.Intent{ResourceKind: r.kind, Kind: "update", ManagementKey: prior.Key.ValueString(), TargetID: id, ExpectedRevision: prior.Revision.ValueInt64(), Spec: r.spec(m)})
	if err != nil {
		resp.Diagnostics.AddError("Cannot update "+r.kind, err.Error())
		return
	}
	o, err := c.GetObject(ctx, r.kind, id)
	if err != nil {
		resp.Diagnostics.AddError("Cannot observe updated resource", err.Error())
		return
	}
	resp.Diagnostics.Append(r.observe(ctx, &resp.State, o)...)
}
func (r *scopeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	m, diags := r.readModel(ctx, req.State)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if m.Protection.ValueBool() {
		if r.kind == "bucket" {
			resp.Diagnostics.AddError("Bucket deletion protected", "Apply deletion_protection = false before destroying this bucket. Nonempty buckets remain protected.")
		} else {
			resp.Diagnostics.AddError("Resource deletion protected", "Apply deletion_protection = false before destroying this resource. Dependencies remain protected.")
		}
		return
	}
	ctx, cancel := context.WithTimeout(ctx, r.config.timeout)
	defer cancel()
	c, err := r.client(m)
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource parent", err.Error())
		return
	}
	id, err := r.parse(m)
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource identity", err.Error())
		return
	}
	_, err = c.Mutate(ctx, api.Intent{ResourceKind: r.kind, Kind: "delete", ManagementKey: m.Key.ValueString(), TargetID: id, ExpectedRevision: m.Revision.ValueInt64()})
	if err != nil {
		resp.Diagnostics.AddError("Cannot delete "+r.kind, err.Error())
	}
}
func (r *scopeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	m := managedModel{ID: types.StringValue(req.ID)}
	parts := strings.Split(req.ID, "/")
	if r.kind == "environment" && len(parts) == 3 {
		m.ProjectID = types.StringValue(parts[1])
	}
	if _, err := r.client(m); err != nil {
		resp.Diagnostics.AddError("Invalid import parent", err.Error())
		return
	}
	if _, err := r.parse(m); err != nil {
		resp.Diagnostics.AddError("Invalid import identity", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	if r.kind == "environment" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), m.ProjectID)...)
	}
}
