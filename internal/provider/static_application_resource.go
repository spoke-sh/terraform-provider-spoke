package provider

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

type staticApplicationResource struct {
	config *configured
	client *api.PublicationClient
}
type staticApplicationModel struct {
	ID           types.String `tfsdk:"id"`
	ResourceID   types.String `tfsdk:"resource_id"`
	Owner        types.String `tfsdk:"owner_id"`
	Key          types.String `tfsdk:"management_key"`
	OperationKey types.String `tfsdk:"operation_key"`
	Name         types.String `tfsdk:"name"`
	Protection   types.Bool   `tfsdk:"deletion_protection"`
	Retention    types.Int64  `tfsdk:"retention_days"`
	Revision     types.String `tfsdk:"revision"`
	Readiness    types.String `tfsdk:"readiness"`
	Reason       types.String `tfsdk:"reason"`
}

var _ resource.ResourceWithModifyPlan = (*staticApplicationResource)(nil)
var _ resource.ResourceWithImportState = (*staticApplicationResource)(nil)
var _ resource.ResourceWithConfigure = (*staticApplicationResource)(nil)

func newStaticApplicationResource() resource.Resource { return &staticApplicationResource{} }
func (r *staticApplicationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_static_application"
}
func publicationNameValidators() []validator.String {
	return []validator.String{stringvalidator.RegexMatches(regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,62}$`), "must contain 1–63 identifier characters")}
}
func (r *staticApplicationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Experimental project-scoped application policy through Platform's publication-v1 declaration API. Requires a project provider without environment_id. Management completion does not provide uploads, hosted bytes or a website endpoint. Import preserves principal ownership and retained operation keys.", Attributes: map[string]schema.Attribute{
		"id":                  schema.StringAttribute{Computed: true, MarkdownDescription: "Canonical import identity: organization_id/project_id/application_id."},
		"resource_id":         schema.StringAttribute{Computed: true, MarkdownDescription: "Exact application ID for binding reservations."},
		"owner_id":            schema.StringAttribute{Computed: true, MarkdownDescription: "Principal that owns the retained management operation namespace."},
		"management_key":      schema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.RegexMatches(publicationManagementKey, "must contain 1–24 identifier characters")}, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, MarkdownDescription: "Stable non-secret recovery identity, unique per application in the project. Changing it replaces the application. Keep this value to recover after state loss."},
		"operation_key":       schema.StringAttribute{Computed: true, MarkdownDescription: "Last observed retained mutation key, or the next mutation reserved by this plan. Each logical mutation uses a distinct key; expiry never permits reuse."},
		"name":                schema.StringAttribute{Required: true, Validators: publicationNameValidators(), MarkdownDescription: "Application label; updates in place and is never used for adoption."},
		"deletion_protection": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true), MarkdownDescription: "Defaults to true. Apply false before destroy. Binding dependencies still restrict deletion."},
		"retention_days":      schema.Int64Attribute{Required: true, Validators: []validator.Int64{int64validator.Between(0, 36500)}, MarkdownDescription: "Declared retention policy in days. This declaration does not upload or delete independent artifacts."},
		"revision":            schema.StringAttribute{Computed: true, MarkdownDescription: "Canonical decimal revision used to fence updates and deletion."},
		"readiness":           schema.StringAttribute{Computed: true, MarkdownDescription: "Observed runtime readiness, independent of management settlement."},
		"reason":              schema.StringAttribute{Computed: true, MarkdownDescription: "Observed readiness reason."},
	}}
}

func publicationConfigured(value any, environment bool) (*configured, *api.PublicationClient, error) {
	c, ok := value.(*configured)
	if !ok || c == nil {
		return nil, nil, errors.New("configured Spoke client required")
	}
	scope := c.publicationConfig.Scope
	if scope.ProjectID == "" || (scope.EnvironmentID != "") != environment {
		return nil, nil, errors.New("publication applications/bindings require a project-only provider; targets require an explicit environment provider")
	}
	client, err := api.NewPublication(c.publicationConfig)
	return c, client, err
}
func (r *staticApplicationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var err error
	r.config, r.client, err = publicationConfigured(req.ProviderData, false)
	if err != nil {
		resp.Diagnostics.AddError("Invalid publication scope", err.Error())
	}
}
func (m staticApplicationModel) policy() (api.PublicationPolicy, error) {
	if m.Name.IsNull() || m.Name.IsUnknown() || m.Protection.IsNull() || m.Protection.IsUnknown() || m.Retention.IsNull() || m.Retention.IsUnknown() || m.Key.IsNull() || m.Key.IsUnknown() {
		return api.PublicationPolicy{}, errors.New("application policy and management_key must be known before mutation")
	}
	return api.PublicationPolicy{Name: m.Name.ValueString(), DeletionProtection: m.Protection.ValueBool(), RetentionDays: m.Retention.ValueInt64()}, nil
}
func publicationCanonical(scope api.Scope, id string) string {
	parts := []string{scope.OrganizationID, scope.ProjectID}
	if scope.EnvironmentID != "" {
		parts = append(parts, scope.EnvironmentID)
	}
	return strings.Join(append(parts, id), "/")
}

var publicationID = regexp.MustCompile(`^[1-9][0-9]{0,15}$`)

func publicationParse(scope api.Scope, canonical string) (string, error) {
	parts := strings.Split(canonical, "/")
	id := parts[len(parts)-1]
	n, err := strconv.ParseUint(id, 10, 64)
	if !publicationID.MatchString(id) || err != nil || n > 9007199254740991 || publicationCanonical(scope, id) != canonical {
		return "", errors.New("publication import identity must match the explicit provider scope")
	}
	return id, nil
}
func (r *staticApplicationResource) observe(ctx context.Context, state *tfsdk.State, key string, identity publicationIdentity, value api.PublicationApplication) diag.Diagnostics {
	m := staticApplicationModel{ID: types.StringValue(publicationCanonical(r.client.Scope(), value.ID)), ResourceID: types.StringValue(value.ID), Owner: types.StringValue(identity.Owner), Key: types.StringValue(key), OperationKey: types.StringValue(identity.lastKey()), Name: types.StringValue(value.Name), Protection: types.BoolValue(value.DeletionProtection), Retention: types.Int64Value(value.RetentionDays), Revision: types.StringValue(value.Revision), Readiness: types.StringValue(value.Readiness), Reason: types.StringValue(value.Reason)}
	return state.Set(ctx, &m)
}
func (r *staticApplicationResource) identity(ctx context.Context, m staticApplicationModel) (publicationIdentity, error) {
	i, err := publicationInspect(ctx, r.client, "application", m.Key.ValueString())
	if err == nil && ((!m.Owner.IsNull() && !m.Owner.IsUnknown() && m.Owner.ValueString() != i.Owner) || (!m.ResourceID.IsNull() && !m.ResourceID.IsUnknown() && m.ResourceID.ValueString() != i.Resource)) {
		err = errors.New("publication principal or resource incarnation differs from Terraform state")
	}
	return i, err
}
func (r *staticApplicationResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || r.client == nil {
		return
	}
	var plan staticApplicationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || plan.Key.IsUnknown() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, r.config.timeout)
	defer cancel()
	var prior staticApplicationModel
	if !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if prior.Key.Equal(plan.Key) {
			key := prior.OperationKey.ValueString()
			if !prior.Name.Equal(plan.Name) || !prior.Protection.Equal(plan.Protection) || !prior.Retention.Equal(plan.Retention) {
				var err error
				key, err = publicationSuccessor("application", plan.Key.ValueString(), key)
				if err != nil {
					resp.Diagnostics.AddError("Cannot plan application mutation", err.Error())
					return
				}
			}
			resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("operation_key"), key)...)
			return
		}
	}
	identity, err := publicationInspect(ctx, r.client, "application", plan.Key.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Cannot inspect application recovery identity", err.Error())
		return
	}
	key, err := publicationKey("application", plan.Key.ValueString(), len(identity.Operations))
	if identity.Resource != "" {
		value, readErr := r.client.Application(ctx, identity.Resource)
		if readErr == nil {
			policy, policyErr := plan.policy()
			if policyErr == nil && policy != value.PublicationPolicy {
				resp.Diagnostics.AddError("Recover existing application before changing intent", "The management_key already owns an application with different policy. Import its canonical ID, or restore the accepted policy before retrying creation.")
				return
			}
			key, err = identity.lastKey(), nil
		} else if !publicationAbsent(readErr) {
			err = readErr
		}
	}
	if err != nil {
		resp.Diagnostics.AddError("Cannot plan application identity", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("operation_key"), key)...)
}
func (r *staticApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m staticApplicationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, r.config.timeout)
	defer cancel()
	policy, err := m.policy()
	if err != nil {
		resp.Diagnostics.AddError("Invalid application intent", err.Error())
		return
	}
	identity, err := publicationInspect(ctx, r.client, "application", m.Key.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Cannot recover application", err.Error())
		return
	}
	if identity.Resource != "" {
		value, readErr := r.client.Application(ctx, identity.Resource)
		if readErr == nil {
			if m.OperationKey.ValueString() != identity.lastKey() {
				resp.Diagnostics.AddError("Stale application creation plan", "Existing retained intent differs from this plan. Refresh and replan without changing management_key.")
				return
			}
			resp.Diagnostics.Append(r.observe(ctx, &resp.State, m.Key.ValueString(), identity, value)...)
			if policy != value.PublicationPolicy {
				resp.Diagnostics.AddError("Recovered application policy changed", "The owned application was retained. Refresh and replan its current policy.")
			}
			return
		}
		if !publicationAbsent(readErr) {
			resp.Diagnostics.AddError("Cannot inspect retained application", readErr.Error())
			return
		}
	}
	key, err := publicationKey("application", m.Key.ValueString(), len(identity.Operations))
	if err != nil || m.OperationKey.IsUnknown() || key != m.OperationKey.ValueString() {
		resp.Diagnostics.AddError("Stale application creation plan", "The retained operation sequence changed. Refresh and replan; no new operation was submitted.")
		return
	}
	op, err := r.client.CreateApplication(ctx, key, policy)
	if err != nil {
		resp.Diagnostics.AddError("Cannot create application", err.Error())
		return
	}
	identity.Operations = append(identity.Operations, op)
	identity.Resource = *op.ResourceID
	value, err := r.client.Application(ctx, identity.Resource)
	if err != nil {
		resp.Diagnostics.AddError("Cannot observe accepted application", "Recover with the original management_key. "+err.Error())
		return
	}
	resp.Diagnostics.Append(r.observe(ctx, &resp.State, m.Key.ValueString(), identity, value)...)
}
func (r *staticApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m staticApplicationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, r.config.timeout)
	defer cancel()
	id, err := publicationParse(r.client.Scope(), m.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid application identity", err.Error())
		return
	}
	identity, err := r.identity(ctx, m)
	if err != nil {
		resp.Diagnostics.AddError("Cannot inspect application ownership", err.Error())
		return
	}
	if identity.Resource != id {
		resp.Diagnostics.AddError("Application ownership mismatch", "The management namespace does not own this application.")
		return
	}
	value, err := r.client.Application(ctx, id)
	if publicationAbsent(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Cannot read application", err.Error())
		return
	}
	resp.Diagnostics.Append(r.observe(ctx, &resp.State, m.Key.ValueString(), identity, value)...)
}
func (r *staticApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var m, prior staticApplicationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, r.config.timeout)
	defer cancel()
	policy, err := m.policy()
	if err != nil {
		resp.Diagnostics.AddError("Invalid application intent", err.Error())
		return
	}
	key, err := publicationSuccessor("application", prior.Key.ValueString(), prior.OperationKey.ValueString())
	if err != nil || key != m.OperationKey.ValueString() {
		resp.Diagnostics.AddError("Invalid application mutation fence", "Replan using the retained operation sequence.")
		return
	}
	identity, err := r.identity(ctx, prior)
	if err != nil {
		resp.Diagnostics.AddError("Cannot inspect application ownership", err.Error())
		return
	}
	if identity.lastKey() != prior.OperationKey.ValueString() && identity.lastKey() != key {
		resp.Diagnostics.AddError("Stale application mutation plan", "The retained operation sequence advanced; replan before submitting new intent.")
		return
	}
	op, err := r.client.UpdateApplication(ctx, identity.Resource, key, prior.Revision.ValueString(), policy)
	if err != nil {
		resp.Diagnostics.AddError("Cannot update application", err.Error())
		return
	}
	if identity.lastKey() != key {
		identity.Operations = append(identity.Operations, op)
	}
	value, err := r.client.Application(ctx, identity.Resource)
	if err != nil {
		resp.Diagnostics.AddError("Cannot observe updated application", err.Error())
		return
	}
	resp.Diagnostics.Append(r.observe(ctx, &resp.State, m.Key.ValueString(), identity, value)...)
}
func (r *staticApplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var m staticApplicationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if m.Protection.ValueBool() {
		resp.Diagnostics.AddError("Application deletion protected", "Apply deletion_protection = false before destroying the application. Binding dependencies still restrict deletion.")
		return
	}
	ctx, cancel := context.WithTimeout(ctx, r.config.timeout)
	defer cancel()
	key, err := publicationSuccessor("application", m.Key.ValueString(), m.OperationKey.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Cannot identify application deletion", err.Error())
		return
	}
	identity, err := r.identity(ctx, m)
	if err != nil {
		resp.Diagnostics.AddError("Cannot inspect application ownership", err.Error())
		return
	}
	if identity.lastKey() != m.OperationKey.ValueString() && identity.lastKey() != key {
		resp.Diagnostics.AddError("Stale application deletion plan", "Replan using the retained operation sequence.")
		return
	}
	if _, err := r.client.DeleteApplication(ctx, identity.Resource, key, m.Revision.ValueString()); err != nil {
		resp.Diagnostics.AddError("Cannot delete application", err.Error())
	}
}
func (r *staticApplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ctx, cancel := context.WithTimeout(ctx, r.config.timeout)
	defer cancel()
	id, err := publicationParse(r.client.Scope(), req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid application import identity", err.Error())
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
		if op.Kind != "application_create" || op.PrincipalID != caps.PrincipalID || op.ResourceID == nil || *op.ResourceID != id {
			continue
		}
		key, _, err := publicationKeyParts("application", op.Key)
		if err != nil {
			continue
		}
		identity, err := publicationInspect(ctx, r.client, "application", key)
		if err != nil {
			resp.Diagnostics.AddError("Cannot recover imported application", err.Error())
			return
		}
		if identity.Resource != id {
			continue
		}
		value, err := r.client.Application(ctx, id)
		if err != nil {
			resp.Diagnostics.AddError("Cannot read imported application", err.Error())
			return
		}
		resp.Diagnostics.Append(r.observe(ctx, &resp.State, key, identity, value)...)
		return
	}
	resp.Diagnostics.AddError("Cannot adopt application ownership", "No complete retained creation receipt binds this application to this principal and a supported Terraform management identity. Import cannot transfer ownership or claim unmanaged declarations.")
}
