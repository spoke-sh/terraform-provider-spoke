package provider

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

type staticBindingResource struct {
	config *configured
	client *api.PublicationClient
}
type staticBindingModel struct {
	ID                types.String `tfsdk:"id"`
	ResourceID        types.String `tfsdk:"resource_id"`
	Owner             types.String `tfsdk:"owner_id"`
	Key               types.String `tfsdk:"management_key"`
	OperationKey      types.String `tfsdk:"operation_key"`
	Application       types.String `tfsdk:"application_id"`
	Environment       types.String `tfsdk:"environment_id"`
	AdmissionRevision types.String `tfsdk:"admission_revision"`
	Revision          types.String `tfsdk:"revision"`
	State             types.String `tfsdk:"state"`
	Target            types.String `tfsdk:"target_id"`
}

var _ resource.ResourceWithModifyPlan = (*staticBindingResource)(nil)
var _ resource.ResourceWithImportState = (*staticBindingResource)(nil)

func newStaticBindingResource() resource.Resource { return &staticBindingResource{} }
func (r *staticBindingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_static_target_binding"
}
func (r *staticBindingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{MarkdownDescription: "Experimental project-owned reservation for one application and destination environment. Requires a project-only provider; the environment principal separately consumes it through spoke_static_target. Retirement cancels only through Platform's authoritative protocol. A live target or uncertain release keeps the parent application held. Target retirement releases the reservation, so coordinate catalog and environment states before recreation.", Attributes: map[string]schema.Attribute{
		"id":                 schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}, MarkdownDescription: "Canonical import identity: organization_id/project_id/binding_id."},
		"resource_id":        schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}, MarkdownDescription: "Exact reservation ID supplied to an environment target."},
		"owner_id":           schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}, MarkdownDescription: "Principal owning the retained reservation management namespace."},
		"management_key":     schema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.RegexMatches(publicationManagementKey, "must contain 1–24 identifier characters")}, PlanModifiers: replace, MarkdownDescription: "Stable non-secret reservation recovery identity. Use a new key when changing a live reservation's application or destination."},
		"operation_key":      schema.StringAttribute{Computed: true, MarkdownDescription: "Last retained mutation key or next creation slot selected by this plan."},
		"application_id":     schema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.RegexMatches(publicationID, "must be a positive decimal ID")}, PlanModifiers: replace, MarkdownDescription: "Exact application ID in this project; changing it requires replacement and a new management_key."},
		"environment_id":     schema.StringAttribute{Required: true, Validators: publicationNameValidators(), PlanModifiers: replace, MarkdownDescription: "Destination environment identifier. Changing it requires replacement and a new management_key."},
		"admission_revision": schema.StringAttribute{Computed: true, MarkdownDescription: "Application revision captured immediately before original admission and retained with that request. Not an ongoing dependency; nullable after import."},
		"revision":           schema.StringAttribute{Computed: true, MarkdownDescription: "Observed reservation revision, including consumption/cancellation transitions."},
		"state":              schema.StringAttribute{Computed: true, MarkdownDescription: "Observed reserved, consumed or cancelling state."},
		"target_id":          schema.StringAttribute{Computed: true, MarkdownDescription: "Nullable consuming target ID. A missing/denied child read cannot release this reservation."},
	}}
}
func (r *staticBindingResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var err error
	r.config, r.client, err = publicationConfigured(req.ProviderData, false)
	if err != nil {
		resp.Diagnostics.AddError("Invalid publication scope", err.Error())
	}
}
func (m staticBindingModel) known() bool {
	return !m.Key.IsNull() && !m.Key.IsUnknown() && !m.Application.IsNull() && !m.Application.IsUnknown() && !m.Environment.IsNull() && !m.Environment.IsUnknown()
}
func (r *staticBindingResource) observe(ctx context.Context, state *tfsdk.State, key string, i publicationIdentity, v api.PublicationBinding, admission types.String) diag.Diagnostics {
	m := staticBindingModel{ID: types.StringValue(publicationCanonical(r.client.Scope(), v.ID)), ResourceID: types.StringValue(v.ID), Owner: types.StringValue(i.Owner), Key: types.StringValue(key), OperationKey: types.StringValue(i.lastKey()), Application: types.StringValue(v.ApplicationID), Environment: types.StringValue(v.EnvironmentID), AdmissionRevision: admission, Revision: types.StringValue(v.Revision), State: types.StringValue(v.State), Target: types.StringPointerValue(v.TargetID)}
	return state.Set(ctx, &m)
}
func (r *staticBindingResource) identity(ctx context.Context, m staticBindingModel) (publicationIdentity, error) {
	i, err := publicationInspect(ctx, r.client, "binding", m.Key.ValueString())
	if err == nil && ((!m.Owner.IsNull() && !m.Owner.IsUnknown() && m.Owner.ValueString() != i.Owner) || (!m.ResourceID.IsNull() && !m.ResourceID.IsUnknown() && m.ResourceID.ValueString() != i.Resource)) {
		err = errors.New("publication principal or binding incarnation differs from state")
	}
	return i, err
}
func (r *staticBindingResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || r.client == nil {
		return
	}
	var m, prior staticBindingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() || m.Key.IsUnknown() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, r.config.timeout)
	defer cancel()
	if !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if prior.Key.Equal(m.Key) {
			if !prior.Application.Equal(m.Application) || !prior.Environment.Equal(m.Environment) {
				resp.Diagnostics.AddError("New binding management identity required", "Changing a live reservation's application or environment requires a distinct management_key to preserve both replacement orders.")
				return
			}
			resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("operation_key"), prior.OperationKey)...)
			resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("admission_revision"), prior.AdmissionRevision)...)
			return
		}
	}
	i, err := publicationInspect(ctx, r.client, "binding", m.Key.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Cannot inspect binding recovery identity", err.Error())
		return
	}
	key, err := publicationKey("binding", m.Key.ValueString(), len(i.Operations))
	if i.Resource != "" {
		v, readErr := r.client.Binding(ctx, i.Resource)
		if readErr == nil {
			if m.known() && (m.Application.ValueString() != v.ApplicationID || m.Environment.ValueString() != v.EnvironmentID) {
				resp.Diagnostics.AddError("Binding management identity already exists", "Import the owned reservation or use a distinct management_key for a different application/environment.")
				return
			}
			resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("operation_key"), i.lastKey())...)
			resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("admission_revision"), types.StringNull())...)
			return
		}
		if !publicationAbsent(readErr) {
			err = readErr
		}
	}
	if err != nil {
		resp.Diagnostics.AddError("Cannot plan binding identity", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("operation_key"), key)...)
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("admission_revision"), types.StringUnknown())...)
}
func (r *staticBindingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m staticBindingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !m.known() {
		resp.Diagnostics.AddError("Invalid binding intent", "management_key, application_id and environment_id must be known before mutation.")
		return
	}
	ctx, cancel := context.WithTimeout(ctx, r.config.timeout)
	defer cancel()
	i, err := publicationInspect(ctx, r.client, "binding", m.Key.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Cannot recover binding", err.Error())
		return
	}
	if i.Resource != "" {
		v, readErr := r.client.Binding(ctx, i.Resource)
		if readErr == nil {
			if m.OperationKey.ValueString() != i.lastKey() || m.Application.ValueString() != v.ApplicationID || m.Environment.ValueString() != v.EnvironmentID {
				resp.Diagnostics.AddError("Stale binding creation plan", "The retained reservation differs from this plan. Replan or import its exact identity.")
				return
			}
			resp.Diagnostics.Append(r.observe(ctx, &resp.State, m.Key.ValueString(), i, v, m.AdmissionRevision)...)
			return
		}
		if !publicationAbsent(readErr) {
			resp.Diagnostics.AddError("Cannot inspect retained binding", readErr.Error())
			return
		}
	}
	key, err := publicationKey("binding", m.Key.ValueString(), len(i.Operations))
	if err != nil || key != m.OperationKey.ValueString() {
		resp.Diagnostics.AddError("Stale binding creation fence", "Replan using the next retained operation slot.")
		return
	}
	// This revision belongs to the first dispatch, not to ongoing configuration.
	// Retained receipt discovery above happens before reading a fresh fence.
	app, err := r.client.Application(ctx, m.Application.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Cannot capture binding admission fence", err.Error())
		return
	}
	m.AdmissionRevision = types.StringValue(app.Revision)
	op, err := r.client.CreateBinding(ctx, m.Application.ValueString(), key, m.AdmissionRevision.ValueString(), m.Environment.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Cannot create binding", err.Error())
		return
	}
	i.Operations = append(i.Operations, op)
	i.Resource = *op.ResourceID
	v, err := r.client.Binding(ctx, i.Resource)
	if err != nil {
		resp.Diagnostics.AddError("Cannot observe accepted binding", "Retain the original management_key. "+err.Error())
		return
	}
	resp.Diagnostics.Append(r.observe(ctx, &resp.State, m.Key.ValueString(), i, v, m.AdmissionRevision)...)
}
func (r *staticBindingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m staticBindingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, r.config.timeout)
	defer cancel()
	id, err := publicationParse(r.client.Scope(), m.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid binding identity", err.Error())
		return
	}
	i, err := r.identity(ctx, m)
	if err != nil {
		resp.Diagnostics.AddError("Cannot inspect binding ownership", err.Error())
		return
	}
	if i.Resource != id {
		resp.Diagnostics.AddError("Binding ownership mismatch", "The management namespace does not own this reservation.")
		return
	}
	v, err := r.client.Binding(ctx, id)
	if publicationAbsent(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Cannot read binding", err.Error())
		return
	}
	resp.Diagnostics.Append(r.observe(ctx, &resp.State, m.Key.ValueString(), i, v, m.AdmissionRevision)...)
}
func (r *staticBindingResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Binding updates are unsupported", "Reservations are immutable. Replace using a distinct management_key; observation never cancels or recreates one.")
}
func (r *staticBindingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var m staticBindingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, r.config.timeout)
	defer cancel()
	key, err := publicationSuccessor("binding", m.Key.ValueString(), m.OperationKey.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Cannot identify binding cancellation", err.Error())
		return
	}
	i, err := r.identity(ctx, m)
	if err != nil {
		resp.Diagnostics.AddError("Cannot inspect binding ownership", err.Error())
		return
	}
	if i.lastKey() != m.OperationKey.ValueString() && i.lastKey() != key {
		resp.Diagnostics.AddError("Stale binding cancellation plan", "Replan using the retained operation sequence.")
		return
	}
	if _, err := r.client.DeleteBinding(ctx, i.Resource, key, m.Revision.ValueString()); err != nil {
		resp.Diagnostics.AddError("Cannot cancel binding", err.Error())
	}
}
func (r *staticBindingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ctx, cancel := context.WithTimeout(ctx, r.config.timeout)
	defer cancel()
	id, err := publicationParse(r.client.Scope(), req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid binding import identity", err.Error())
		return
	}
	key, i, err := publicationImportIdentity(ctx, r.client, "binding", id)
	if err != nil {
		resp.Diagnostics.AddError("Cannot adopt binding ownership", err.Error())
		return
	}
	v, err := r.client.Binding(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Cannot read imported binding", err.Error())
		return
	}
	resp.Diagnostics.Append(r.observe(ctx, &resp.State, key, i, v, types.StringNull())...)
}
func publicationImportIdentity(ctx context.Context, c *api.PublicationClient, kind, id string) (string, publicationIdentity, error) {
	var zero publicationIdentity
	caps, err := c.Capabilities(ctx)
	if err != nil {
		return "", zero, err
	}
	history, err := c.History(ctx, "all")
	if err != nil {
		return "", zero, err
	}
	for _, op := range history {
		if op.Kind != kind+"_create" || op.PrincipalID != caps.PrincipalID || op.ResourceID == nil || *op.ResourceID != id {
			continue
		}
		key, _, err := publicationKeyParts(kind, op.Key)
		if err != nil {
			continue
		}
		i, err := publicationInspect(ctx, c, kind, key)
		if err != nil {
			return "", zero, err
		}
		if i.Resource == id {
			return key, i, nil
		}
	}
	return "", zero, errors.New("no complete retained creation receipt binds this resource to this principal and a supported management identity; import cannot transfer ownership")
}
