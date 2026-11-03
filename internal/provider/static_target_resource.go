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

type staticTargetResource struct {
	config *configured
	client *api.PublicationClient
}
type staticTargetModel struct {
	staticApplicationModel
	Binding               types.String `tfsdk:"binding_id"`
	Application           types.String `tfsdk:"application_id"`
	State                 types.String `tfsdk:"state"`
	DesiredGeneration     types.String `tfsdk:"desired_generation"`
	ObservedGeneration    types.String `tfsdk:"observed_generation"`
	Configuration         types.String `tfsdk:"configuration_digest"`
	DesiredConfiguration  types.String `tfsdk:"desired_configuration_digest"`
	ObservedConfiguration types.String `tfsdk:"observed_configuration_digest"`
	ObservedBundle        types.String `tfsdk:"observed_bundle_digest"`
	Bundle                types.String `tfsdk:"bundle_digest"`
	LastObservation       types.String `tfsdk:"last_observation_ms"`
	Endpoint              types.String `tfsdk:"endpoint"`
	Expires               types.String `tfsdk:"expires_at_ms"`
}

var _ resource.ResourceWithModifyPlan = (*staticTargetResource)(nil)
var _ resource.ResourceWithImportState = (*staticTargetResource)(nil)
var _ resource.ResourceWithConfigure = (*staticTargetResource)(nil)

func newStaticTargetResource() resource.Resource { return &staticTargetResource{} }
func (r *staticTargetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_static_target"
}
func (r *staticTargetResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	base := &staticApplicationResource{}
	base.Schema(ctx, req, resp)
	resp.Schema.MarkdownDescription = "Experimental environment-scoped target policy through Platform's normal publication-v1 API. Consumes one project reservation under a separately authorized environment principal. Only name, protection and retention are writable; runtime heads are observed. Retirement releases its reservation; coordinate recreation with the project catalog."
	attrs := resp.Schema.Attributes
	protection := attrs["deletion_protection"].(schema.BoolAttribute)
	protection.MarkdownDescription = "Defaults to true. Apply false before destroy. Pending work and selected runtime references still block retirement; this profile has no withdrawal API."
	attrs["deletion_protection"] = protection
	id := attrs["id"].(schema.StringAttribute)
	id.MarkdownDescription = "Canonical import identity: organization_id/project_id/environment_id/target_id."
	attrs["id"] = id
	label := attrs["name"].(schema.StringAttribute)
	label.MarkdownDescription = "Target label; policy updates do not restore runtime heads."
	attrs["name"] = label
	key := attrs["management_key"].(schema.StringAttribute)
	key.MarkdownDescription = "Stable non-secret recovery identity unique per target in this environment. Use a new key when replacing a live target with another binding."
	attrs["management_key"] = key
	rid := attrs["resource_id"].(schema.StringAttribute)
	rid.MarkdownDescription = "Exact target ID."
	attrs["resource_id"] = rid
	attrs["binding_id"] = schema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.RegexMatches(publicationID, "must be a positive decimal ID")}, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, MarkdownDescription: "Exact project reservation for this environment. A binding can be consumed only once; changing it requires a new management_key and replacement."}
	for name, description := range map[string]string{
		"application_id":                "Observed application identity, nullable before binding settlement.",
		"state":                         "Observed declaration state.",
		"desired_generation":            "Observed runtime generation; Terraform never selects a head.",
		"observed_generation":           "Nullable observed runtime generation.",
		"configuration_digest":          "Current declaration configuration digest.",
		"desired_configuration_digest":  "Nullable selected runtime configuration commitment.",
		"observed_configuration_digest": "Nullable observed runtime configuration commitment.",
		"observed_bundle_digest":        "Nullable independently observed bundle commitment.",
		"bundle_digest":                 "Nullable selected bundle commitment; never reconciled by Terraform.",
		"last_observation_ms":           "Nullable last observation time as a decimal string.",
		"endpoint":                      "Nullable observed endpoint; this declaration profile supplies no hosted endpoint.",
		"expires_at_ms":                 "Nullable observed expiry time as a decimal string.",
	} {
		attrs[name] = schema.StringAttribute{Computed: true, MarkdownDescription: description}
	}
}

func (r *staticTargetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var err error
	r.config, r.client, err = publicationConfigured(req.ProviderData, true)
	if err != nil {
		resp.Diagnostics.AddError("Invalid publication scope", err.Error())
	}
}
func (m staticTargetModel) policy() (api.PublicationPolicy, error) {
	if m.Binding.IsNull() || m.Binding.IsUnknown() {
		return api.PublicationPolicy{}, errors.New("binding_id must be known before mutation")
	}
	return m.staticApplicationModel.policy()
}
func (r *staticTargetResource) observe(ctx context.Context, state *tfsdk.State, key string, identity publicationIdentity, value api.PublicationTarget) diag.Diagnostics {
	base := staticApplicationModel{ID: types.StringValue(publicationCanonical(r.client.Scope(), value.ID)), ResourceID: types.StringValue(value.ID), Owner: types.StringValue(identity.Owner), Key: types.StringValue(key), OperationKey: types.StringValue(identity.lastKey()), Name: types.StringValue(value.Name), Protection: types.BoolValue(value.DeletionProtection), Retention: types.Int64Value(value.RetentionDays), Revision: types.StringValue(value.Revision), Readiness: types.StringValue(value.Readiness), Reason: types.StringValue(value.Reason)}
	m := staticTargetModel{staticApplicationModel: base, Binding: types.StringValue(value.BindingID), Application: types.StringPointerValue(value.ApplicationID), State: types.StringValue(value.State), DesiredGeneration: types.StringValue(value.DesiredGeneration), ObservedGeneration: types.StringPointerValue(value.ObservedGeneration), Configuration: types.StringValue(value.ConfigurationDigest), DesiredConfiguration: types.StringPointerValue(value.DesiredConfigurationDigest), ObservedConfiguration: types.StringPointerValue(value.ObservedConfigurationDigest), ObservedBundle: types.StringPointerValue(value.ObservedBundleDigest), Bundle: types.StringPointerValue(value.BundleDigest), LastObservation: types.StringPointerValue(value.LastObservationMS), Endpoint: types.StringPointerValue(value.Endpoint), Expires: types.StringPointerValue(value.ExpiresAtMS)}
	return state.Set(ctx, &m)
}
func (r *staticTargetResource) identity(ctx context.Context, m staticTargetModel) (publicationIdentity, error) {
	i, err := publicationInspect(ctx, r.client, "target", m.Key.ValueString())
	if err == nil && ((!m.Owner.IsNull() && !m.Owner.IsUnknown() && m.Owner.ValueString() != i.Owner) || (!m.ResourceID.IsNull() && !m.ResourceID.IsUnknown() && m.ResourceID.ValueString() != i.Resource)) {
		err = errors.New("publication principal or resource incarnation differs from Terraform state")
	}
	return i, err
}
func (r *staticTargetResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || r.client == nil {
		return
	}
	var plan staticTargetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || plan.Key.IsUnknown() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, r.config.timeout)
	defer cancel()
	var prior staticTargetModel
	if !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if prior.Key.Equal(plan.Key) {
			if !prior.Binding.Equal(plan.Binding) {
				resp.Diagnostics.AddError("New target management identity required", "Changing the binding of a live target requires a distinct management_key so both replacement orders retain separate identities.")
				return
			}
			key := prior.OperationKey.ValueString()
			if !prior.Name.Equal(plan.Name) || !prior.Protection.Equal(plan.Protection) || !prior.Retention.Equal(plan.Retention) {
				var err error
				key, err = publicationSuccessor("target", plan.Key.ValueString(), key)
				if err != nil {
					resp.Diagnostics.AddError("Cannot pla target mutation", err.Error())
					return
				}
			}
			resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("operation_key"), key)...)
			return
		}
	}
	identity, err := publicationInspect(ctx, r.client, "target", plan.Key.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Cannot inspect target recovery identity", err.Error())
		return
	}
	key, err := publicationKey("target", plan.Key.ValueString(), len(identity.Operations))
	if identity.Resource != "" {
		value, readErr := r.client.Target(ctx, identity.Resource)
		if readErr == nil {
			policy, policyErr := plan.policy()
			if policyErr == nil && (policy != value.PublicationPolicy || plan.Binding.ValueString() != value.BindingID) {
				resp.Diagnostics.AddError("Recover existing target before changing intent", "The management_key already owns a target with different policy. Import its canonical ID, or restore the accepted policy before retrying creation.")
				return
			}
			key, err = identity.lastKey(), nil
		} else if !publicationAbsent(readErr) {
			err = readErr
		}
	}
	if err != nil {
		resp.Diagnostics.AddError("Cannot pla target identity", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("operation_key"), key)...)
}
func (r *staticTargetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m staticTargetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, r.config.timeout)
	defer cancel()
	policy, err := m.policy()
	if err != nil {
		resp.Diagnostics.AddError("Invalid target intent", err.Error())
		return
	}
	identity, err := publicationInspect(ctx, r.client, "target", m.Key.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Cannot recover target", err.Error())
		return
	}
	if identity.Resource != "" {
		value, readErr := r.client.Target(ctx, identity.Resource)
		if readErr == nil {
			if m.OperationKey.ValueString() != identity.lastKey() {
				resp.Diagnostics.AddError("Stale target creation plan", "Existing retained intent differs from this plan. Refresh and replan without changing management_key.")
				return
			}
			resp.Diagnostics.Append(r.observe(ctx, &resp.State, m.Key.ValueString(), identity, value)...)
			if policy != value.PublicationPolicy || m.Binding.ValueString() != value.BindingID {
				resp.Diagnostics.AddError("Recovered target policy changed", "The owned target was retained. Refresh and replan its current policy.")
			}
			return
		}
		if !publicationAbsent(readErr) {
			resp.Diagnostics.AddError("Cannot inspect retained target", readErr.Error())
			return
		}
	}
	key, err := publicationKey("target", m.Key.ValueString(), len(identity.Operations))
	if err != nil || m.OperationKey.IsUnknown() || key != m.OperationKey.ValueString() {
		resp.Diagnostics.AddError("Stale target creation plan", "The retained operation sequence changed. Refresh and replan; no new operation was submitted.")
		return
	}
	op, err := r.client.CreateTarget(ctx, m.Binding.ValueString(), key, policy)
	if err != nil {
		resp.Diagnostics.AddError("Cannot create target", err.Error())
		return
	}
	identity.Operations = append(identity.Operations, op)
	identity.Resource = *op.ResourceID
	value, err := r.client.Target(ctx, identity.Resource)
	if err != nil {
		resp.Diagnostics.AddError("Cannot observe accepted target", "Recover with the original management_key. "+err.Error())
		return
	}
	resp.Diagnostics.Append(r.observe(ctx, &resp.State, m.Key.ValueString(), identity, value)...)
}
func (r *staticTargetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m staticTargetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, r.config.timeout)
	defer cancel()
	id, err := publicationParse(r.client.Scope(), m.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid target identity", err.Error())
		return
	}
	identity, err := r.identity(ctx, m)
	if err != nil {
		resp.Diagnostics.AddError("Cannot inspect target ownership", err.Error())
		return
	}
	if identity.Resource != id {
		resp.Diagnostics.AddError("Target ownership mismatch", "The management namespace does not own this target.")
		return
	}
	value, err := r.client.Target(ctx, id)
	if publicationAbsent(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Cannot read target", err.Error())
		return
	}
	resp.Diagnostics.Append(r.observe(ctx, &resp.State, m.Key.ValueString(), identity, value)...)
}
func (r *staticTargetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var m, prior staticTargetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, r.config.timeout)
	defer cancel()
	policy, err := m.policy()
	if err != nil {
		resp.Diagnostics.AddError("Invalid target intent", err.Error())
		return
	}
	key, err := publicationSuccessor("target", prior.Key.ValueString(), prior.OperationKey.ValueString())
	if err != nil || key != m.OperationKey.ValueString() {
		resp.Diagnostics.AddError("Invalid target mutation fence", "Replan using the retained operation sequence.")
		return
	}
	identity, err := r.identity(ctx, prior)
	if err != nil {
		resp.Diagnostics.AddError("Cannot inspect target ownership", err.Error())
		return
	}
	if identity.lastKey() != prior.OperationKey.ValueString() && identity.lastKey() != key {
		resp.Diagnostics.AddError("Stale target mutation plan", "The retained operation sequence advanced; replan before submitting new intent.")
		return
	}
	op, err := r.client.UpdateTarget(ctx, identity.Resource, key, prior.Revision.ValueString(), policy)
	if err != nil {
		resp.Diagnostics.AddError("Cannot update target", err.Error())
		return
	}
	if identity.lastKey() != key {
		identity.Operations = append(identity.Operations, op)
	}
	value, err := r.client.Target(ctx, identity.Resource)
	if err != nil {
		resp.Diagnostics.AddError("Cannot observe updated target", err.Error())
		return
	}
	resp.Diagnostics.Append(r.observe(ctx, &resp.State, m.Key.ValueString(), identity, value)...)
}
func (r *staticTargetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var m staticTargetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if m.Protection.ValueBool() {
		resp.Diagnostics.AddError("Target deletion protected", "Apply deletion_protection = false before destroying the target. Pending work or selected runtime references can still restrict deletion.")
		return
	}
	ctx, cancel := context.WithTimeout(ctx, r.config.timeout)
	defer cancel()
	key, err := publicationSuccessor("target", m.Key.ValueString(), m.OperationKey.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Cannot identify target deletion", err.Error())
		return
	}
	identity, err := r.identity(ctx, m)
	if err != nil {
		resp.Diagnostics.AddError("Cannot inspect target ownership", err.Error())
		return
	}
	if identity.lastKey() != m.OperationKey.ValueString() && identity.lastKey() != key {
		resp.Diagnostics.AddError("Stale target deletion plan", "Replan using the retained operation sequence.")
		return
	}
	if _, err := r.client.DeleteTarget(ctx, identity.Resource, key, m.Revision.ValueString()); err != nil {
		resp.Diagnostics.AddError("Cannot delete target", err.Error())
	}
}
func (r *staticTargetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ctx, cancel := context.WithTimeout(ctx, r.config.timeout)
	defer cancel()
	id, err := publicationParse(r.client.Scope(), req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid target import identity", err.Error())
		return
	}
	caps, err := r.client.Capabilities(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Cannot verify import principal", err.Error())
		return
	}
	history, err := r.client.History(ctx, "all")
	if err != nil {
		resp.Diagnostics.AddError("Cannot inspect import ownership", err.Error())
		return
	}
	for _, op := range history {
		if op.Kind != "target_create" || op.PrincipalID != caps.PrincipalID || op.ResourceID == nil || *op.ResourceID != id {
			continue
		}
		key, _, err := publicationKeyParts("target", op.Key)
		if err != nil {
			continue
		}
		identity, err := publicationInspect(ctx, r.client, "target", key)
		if err != nil {
			resp.Diagnostics.AddError("Cannot recover imported target", err.Error())
			return
		}
		if identity.Resource != id {
			continue
		}
		value, err := r.client.Target(ctx, id)
		if err != nil {
			resp.Diagnostics.AddError("Cannot read imported target", err.Error())
			return
		}
		resp.Diagnostics.Append(r.observe(ctx, &resp.State, key, identity, value)...)
		return
	}
	resp.Diagnostics.AddError("Cannot adopt target ownership", "No complete retained creation receipt binds this target to this principal and a supported Terraform management identity. Import cannot transfer ownership or claim unmanaged declarations.")
}
