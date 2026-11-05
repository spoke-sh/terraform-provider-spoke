package provider

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

type governorModel struct {
	ID              types.String `tfsdk:"id"`
	ResourceID      types.String `tfsdk:"resource_id"`
	Owner           types.String `tfsdk:"owner_id"`
	Key             types.String `tfsdk:"management_key"`
	Name            types.String `tfsdk:"name"`
	Admission       types.String `tfsdk:"admission_policy_id"`
	Budget          types.String `tfsdk:"budget_policy_id"`
	Cache           types.String `tfsdk:"cache_policy_id"`
	Profiles        types.List   `tfsdk:"execution_profile_ids"`
	Jobs            types.Int64  `tfsdk:"max_active_jobs"`
	CPUs            types.Int64  `tfsdk:"max_allocated_vcpus"`
	Warm            types.Int64  `tfsdk:"max_warm_runners"`
	JobSeconds      types.Int64  `tfsdk:"max_job_seconds"`
	QueueSeconds    types.Int64  `tfsdk:"max_queue_seconds"`
	Retirement      types.String `tfsdk:"retirement_policy"`
	Protection      types.Bool   `tfsdk:"deletion_protection"`
	Revision        types.Int64  `tfsdk:"revision"`
	ActiveRevision  types.Int64  `tfsdk:"active_revision"`
	Readiness       types.String `tfsdk:"readiness"`
	Blockers        types.List   `tfsdk:"blockers"`
	ActiveJobs      types.Int64  `tfsdk:"active_jobs"`
	AllocatedCPUs   types.Int64  `tfsdk:"allocated_vcpus"`
	Usage           types.Int64  `tfsdk:"unsettled_usage"`
	OperationID     types.String `tfsdk:"operation_id"`
	Fingerprint     types.String `tfsdk:"operation_fingerprint"`
	Sequence        types.Int64  `tfsdk:"operation_sequence"`
	OperationStatus types.String `tfsdk:"operation_status"`
}

type governorResource struct {
	config *configured
	client *api.GovernorClient
}

var _ resource.ResourceWithModifyPlan = (*governorResource)(nil)
var _ resource.ResourceWithConfigure = (*governorResource)(nil)
var _ resource.ResourceWithImportState = (*governorResource)(nil)
var _ resource.ResourceWithValidateConfig = (*governorResource)(nil)

func newGovernorResource() resource.Resource { return &governorResource{} }
func (r *governorResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ci_governor"
}

func governorAttributes() map[string]schema.Attribute {
	a := map[string]schema.Attribute{}
	for name, description := range map[string]string{
		"id": "Canonical import ID: organization/project/environment/governor_id.", "resource_id": "Opaque governor ID.", "owner_id": "Principal owning the retained management key.", "readiness": "Controller observation; an applied declaration can remain disabled with execution_unqualified.", "operation_id": "Original retained management operation ID.", "operation_status": "Latest retained request status; accepted does not mean applied.",
	} {
		a[name] = schema.StringAttribute{Computed: true, MarkdownDescription: description}
	}
	a["operation_fingerprint"] = schema.StringAttribute{Computed: true, MarkdownDescription: "Canonical scope, owner and request commitment frozen by the plan. This is not a credential or a durable acceptance receipt."}
	a["management_key"] = schema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.RegexMatches(governorKeyPattern, "must use 1–64 identifier characters, beginning with a letter or digit")}, MarkdownDescription: "Stable non-secret identity supplied in configuration before dispatch. Retire before changing it; recreation requires a new key."}
	for name, description := range map[string]string{"name": "Policy name.", "admission_policy_id": "Existing authorized admission entitlement. Immutable while this governor owns it.", "budget_policy_id": "Existing authorized budget policy reference; no spending is qualified.", "cache_policy_id": "Existing authorized managed-cache policy reference; no cache is provisioned."} {
		a[name] = schema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.RegexMatches(governorIDPattern, "must be a lowercase 1–63 character identifier beginning with a letter")}, MarkdownDescription: description}
	}
	a["execution_profile_ids"] = schema.ListAttribute{Required: true, ElementType: types.StringType, Validators: []validator.List{listvalidator.SizeBetween(1, 1), listvalidator.ValueStringsAre(stringvalidator.OneOf("linux-x86_64-v1"))}, MarkdownDescription: "Exactly [\"linux-x86_64-v1\"] in the provisional profile; this is not execution qualification."}
	for name, bounds := range map[string][2]int64{"max_active_jobs": {1, 256}, "max_allocated_vcpus": {1, 4096}, "max_warm_runners": {0, 0}, "max_job_seconds": {1, 86400}, "max_queue_seconds": {1, 3600}} {
		a[name] = schema.Int64Attribute{Required: true, Validators: []validator.Int64{int64validator.Between(bounds[0], bounds[1])}, MarkdownDescription: "Explicit policy bound from ci-governor-v1. Zero-demand changes launch no workers."}
	}
	a["retirement_policy"] = schema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.OneOf("drain")}, MarkdownDescription: "Must be drain. Uncertain leases, capacity and unsettled usage prevent completion."}
	a["deletion_protection"] = schema.BoolAttribute{Required: true, MarkdownDescription: "Apply false before retirement. Caches, artifacts and retained usage keep their separate owners."}
	for name, description := range map[string]string{"revision": "Desired policy revision used to fence mutations.", "active_revision": "Controller-selected active revision.", "active_jobs": "Observed active jobs, never a desired count.", "allocated_vcpus": "Observed allocated vCPUs, never a desired count.", "unsettled_usage": "Observed unsettled usage.", "operation_sequence": "Selected plan sequence or latest observed request sequence. Timeouts never allocate a successor."} {
		a[name] = schema.Int64Attribute{Computed: true, MarkdownDescription: description}
	}
	a["blockers"] = schema.ListAttribute{Computed: true, ElementType: types.StringType, MarkdownDescription: "Bounded runtime or drain blockers; reads do not resolve them."}
	return a
}
func (r *governorResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Version: 1, Attributes: governorAttributes(), MarkdownDescription: "Experimental operator CI supply policy through the separately pinned provisional ci-governor-v1 HTTPS contract. Requires existing organization, project and environment scopes. Synthetic installed tests do not qualify a native service, workers, warming, handoff or billing. Customers use workflow demand without capacity Terraform."}
}
func governorConfigured(value any) (*configured, *api.GovernorClient, error) {
	c, ok := value.(*configured)
	if !ok || c == nil {
		return nil, nil, errors.New("configured Spoke client required")
	}
	client, err := api.NewGovernor(c.httpConfig)
	return c, client, err
}
func (r *governorResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var err error
	r.config, r.client, err = governorConfigured(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Invalid governor scope", err.Error())
	}
}
func (m governorModel) policy(ctx context.Context) (*api.GovernorPolicy, bool, error) {
	for _, v := range []attr.Value{m.Name, m.Admission, m.Budget, m.Cache, m.Profiles, m.Jobs, m.CPUs, m.Warm, m.JobSeconds, m.QueueSeconds, m.Retirement, m.Protection} {
		if v.IsUnknown() {
			return nil, false, nil
		}
		if v.IsNull() {
			return nil, true, errors.New("all governor policy inputs are required")
		}
	}
	for _, v := range m.Profiles.Elements() {
		if v.IsUnknown() {
			return nil, false, nil
		}
		if v.IsNull() {
			return nil, true, errors.New("execution profile cannot be null")
		}
	}
	var profiles []string
	if d := m.Profiles.ElementsAs(ctx, &profiles, false); d.HasError() {
		return nil, true, errors.New("invalid execution profiles")
	}
	for _, v := range []types.Int64{m.Jobs, m.CPUs, m.Warm, m.JobSeconds, m.QueueSeconds} {
		if v.ValueInt64() < 0 || v.ValueInt64() > 4294967295 {
			return nil, true, errors.New("governor integer is outside the wire bounds")
		}
	}
	p := &api.GovernorPolicy{Name: m.Name.ValueString(), AdmissionPolicyID: m.Admission.ValueString(), BudgetPolicyID: m.Budget.ValueString(), CachePolicyID: m.Cache.ValueString(), ExecutionProfileIDs: profiles, MaxActiveJobs: uint64(m.Jobs.ValueInt64()), MaxAllocatedVCPUs: uint64(m.CPUs.ValueInt64()), MaxWarmRunners: uint64(m.Warm.ValueInt64()), MaxJobSeconds: uint64(m.JobSeconds.ValueInt64()), MaxQueueSeconds: uint64(m.QueueSeconds.ValueInt64()), RetirementPolicy: m.Retirement.ValueString(), DeletionProtection: m.Protection.ValueBool()}
	return p, true, api.ValidateGovernorPolicy(*p)
}
func (r *governorResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var m governorModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if _, _, err := m.policy(ctx); err != nil {
		resp.Diagnostics.AddError("Invalid governor policy", err.Error())
	}
}
func (m governorModel) anchor(scope api.Scope) (*governorAnchor, error) {
	id, err := governorParse(scope, m.ID.ValueString())
	if err != nil {
		return nil, err
	}
	if m.Owner.IsUnknown() || m.Owner.IsNull() || !governorIDPattern.MatchString(m.Owner.ValueString()) || !governorKeyPattern.MatchString(m.Key.ValueString()) || m.ResourceID.ValueString() != id || m.Revision.IsUnknown() || m.Revision.IsNull() || m.Revision.ValueInt64() < 1 || m.Revision.ValueInt64() > 4294967295 || m.Sequence.IsUnknown() || m.Sequence.IsNull() || m.Sequence.ValueInt64() < 0 || m.Sequence.ValueInt64() > 4294967295 {
		return nil, errors.New("incompatible governor state identity; import the owned scoped ID before mutation")
	}
	return &governorAnchor{Owner: m.Owner.ValueString(), Target: id, Revision: uint64(m.Revision.ValueInt64()), Sequence: uint64(m.Sequence.ValueInt64())}, nil
}
func governorObserved(ctx context.Context, scope api.Scope, g api.Governor, op api.GovernorOperation) governorModel {
	profiles, _ := types.ListValueFrom(ctx, types.StringType, g.Policy.ExecutionProfileIDs)
	blockers, _ := types.ListValueFrom(ctx, types.StringType, g.Blockers)
	return governorModel{ID: types.StringValue(governorCanonical(scope, g.ID)), ResourceID: types.StringValue(g.ID), Owner: types.StringValue(g.PrincipalID), Key: types.StringValue(g.ManagementKey), Name: types.StringValue(g.Policy.Name), Admission: types.StringValue(g.Policy.AdmissionPolicyID), Budget: types.StringValue(g.Policy.BudgetPolicyID), Cache: types.StringValue(g.Policy.CachePolicyID), Profiles: profiles, Jobs: types.Int64Value(int64(g.Policy.MaxActiveJobs)), CPUs: types.Int64Value(int64(g.Policy.MaxAllocatedVCPUs)), Warm: types.Int64Value(int64(g.Policy.MaxWarmRunners)), JobSeconds: types.Int64Value(int64(g.Policy.MaxJobSeconds)), QueueSeconds: types.Int64Value(int64(g.Policy.MaxQueueSeconds)), Retirement: types.StringValue(g.Policy.RetirementPolicy), Protection: types.BoolValue(g.Policy.DeletionProtection), Revision: types.Int64Value(int64(g.Revision)), ActiveRevision: types.Int64Value(int64(g.ActiveRevision)), Readiness: types.StringValue(g.State), Blockers: blockers, ActiveJobs: types.Int64Value(int64(g.ActiveJobs)), AllocatedCPUs: types.Int64Value(int64(g.AllocatedVCPUs)), Usage: types.Int64Value(int64(g.UnsettledUsage)), OperationID: types.StringValue(op.ID), Fingerprint: types.StringValue(op.Fingerprint), Sequence: types.Int64Value(int64(op.Sequence)), OperationStatus: types.StringValue(op.Status)}
}

func (r *governorResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if r.client == nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, r.config.timeout)
	defer cancel()
	var m, prior governorModel
	var anchor *governorAnchor
	var err error
	destroy := req.Plan.Raw.IsNull()
	if !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
		if resp.Diagnostics.HasError() {
			return
		}
		anchor, err = prior.anchor(r.config.httpConfig.Scope)
		if err != nil {
			resp.Diagnostics.AddError("Invalid governor state", err.Error())
			return
		}
	}
	kind := "create"
	var policy *api.GovernorPolicy
	if destroy {
		if anchor == nil {
			return
		}
		m = prior
		kind = "retire"
		if prior.Protection.ValueBool() {
			resp.Diagnostics.AddError("Governor deletion protected", "Apply deletion_protection = false before retirement.")
			return
		}
	} else {
		resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if m.Key.IsUnknown() {
			resp.Diagnostics.Append(resp.Private.SetKey(ctx, governorPrivateKey, nil)...)
			return
		}
		var known bool
		policy, known, err = m.policy(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Invalid governor policy", err.Error())
			return
		}
		if !known {
			resp.Diagnostics.Append(resp.Private.SetKey(ctx, governorPrivateKey, nil)...)
			return
		}
		if anchor != nil {
			if !prior.Key.Equal(m.Key) {
				resp.Diagnostics.AddError("Governor management key is immutable", "Retire the existing governor before configuring its successor with a new management_key.")
				return
			}
			kind = "update"
		}
	}
	v, err := governorInspect(ctx, r.client, m.Key.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Cannot inspect governor identity", err.Error())
		return
	}
	f, err := governorSelect(v, m.Key.ValueString(), kind, policy, anchor)
	if kind == "create" && v.Latest != nil && v.Latest.Intent.Kind == "retire" && v.Latest.Status == "applied" && v.Value == nil {
		// A destroy pre-pass can look like creation after refresh removed the
		// retired resource. Defer this refusal to Create's checked fence so
		// Terraform can finish its final null destroy plan. Never reuse the key.
		f = governorFence{Version: 1, Scope: v.Scope, Owner: v.Owner, Request: api.GovernorRequest{ManagementKey: m.Key.ValueString(), Intent: api.GovernorIntent{Kind: "create", Policy: policy}}}
		err = nil
	}
	if err != nil {
		resp.Diagnostics.AddError("Cannot plan governor mutation", err.Error())
		return
	}
	fingerprint, err := api.GovernorFingerprint(f.Scope, f.Owner, f.Request)
	if err != nil {
		resp.Diagnostics.AddError("Invalid governor intent", err.Error())
		return
	}
	b, _ := json.Marshal(f)
	resp.Diagnostics.Append(resp.Private.SetKey(ctx, governorPrivateKey, b)...)
	if destroy {
		return
	}
	if f.Observe && v.Value != nil {
		resp.Diagnostics.Append(resp.Plan.Set(ctx, governorObserved(ctx, r.config.httpConfig.Scope, *v.Value, *v.Latest))...)
	} else {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("operation_sequence"), int64(f.Request.Sequence))...)
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("operation_fingerprint"), fingerprint)...)
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("owner_id"), f.Owner)...)
	}
}

type governorPrivateReader interface {
	GetKey(context.Context, string) ([]byte, diag.Diagnostics)
}

func governorPlanned(ctx context.Context, private governorPrivateReader, diags *diag.Diagnostics) (governorFence, bool) {
	b, d := private.GetKey(ctx, governorPrivateKey)
	diags.Append(d...)
	if diags.HasError() {
		return governorFence{}, false
	}
	f, err := governorDecodeFence(b)
	if err != nil {
		diags.AddError("Invalid governor plan", err.Error())
		return f, false
	}
	return f, true
}
func (r *governorResource) apply(ctx context.Context, f governorFence, m governorModel, kind string) (*governorModel, error) {
	if f.Request.ManagementKey != m.Key.ValueString() {
		return nil, governorStale()
	}
	var policy *api.GovernorPolicy
	if kind != "retire" {
		var known bool
		var err error
		policy, known, err = m.policy(ctx)
		if err != nil {
			return nil, err
		}
		if !known {
			return nil, errors.New("governor mutation inputs must be known before dispatch")
		}
	}
	if f.Observe {
		if kind == "retire" || !governorPolicyEqual(f.ObservedPolicy, policy) {
			return nil, governorStale()
		}
	} else if f.Request.Intent.Kind != kind || !governorPolicyEqual(f.Request.Intent.Policy, policy) {
		return nil, governorStale()
	}
	if kind != "create" {
		a, err := m.anchor(r.config.httpConfig.Scope)
		if err != nil {
			return nil, err
		}
		if a.Owner != f.Owner || f.Request.Intent.TargetID != nil && *f.Request.Intent.TargetID != a.Target {
			return nil, governorStale()
		}
	}
	v, err := governorInspect(ctx, r.client, m.Key.ValueString())
	if err != nil {
		return nil, err
	}
	if err = governorCheckFence(v, f); err != nil {
		return nil, err
	}
	if !f.Observe {
		if _, err = r.client.Apply(ctx, f.Request); err != nil {
			return nil, err
		}
		v, err = governorInspect(ctx, r.client, m.Key.ValueString())
		if err != nil {
			return nil, err
		}
		if v.Latest == nil || v.Latest.Sequence != f.Request.Sequence || governorCheckFence(v, f) != nil {
			return nil, governorStale()
		}
	}
	if kind == "retire" {
		if v.Value != nil || v.Latest.Status != "applied" || v.Latest.Intent.Kind != "retire" {
			return nil, errors.New("governor retirement remains unresolved; retain state and original request")
		}
		return nil, nil
	}
	if v.Value == nil {
		return nil, errors.New("applied governor cannot be observed; retain the original request identity")
	}
	observed := governorObserved(ctx, r.config.httpConfig.Scope, *v.Value, *v.Latest)
	if !f.Observe && v.Value.Revision != f.Request.Intent.ExpectedRevision+1 {
		return &observed, errors.New("governor revision changed after the original operation; refresh and replan")
	}
	if !governorPolicyEqual(&v.Value.Policy, policy) {
		return &observed, errors.New("governor policy changed after the original operation; refresh and replan")
	}
	return &observed, nil
}
func (r *governorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m governorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Framework 1.19 drops PlannedPrivate in CreateResource. Freeze the create
	// tuple in computed plan fields instead; the fingerprint also fences scope.
	policy, known, err := m.policy(ctx)
	if err != nil || !known || m.Sequence.IsNull() || m.Sequence.IsUnknown() || m.Sequence.ValueInt64() < 0 || m.Sequence.ValueInt64() > 4294967295 || m.Owner.IsUnknown() || m.Owner.IsNull() || m.Fingerprint.IsUnknown() || m.Fingerprint.IsNull() {
		resp.Diagnostics.AddError("Invalid governor plan", "Creation identity and policy must be resolved by planning before dispatch.")
		return
	}
	f := governorFence{Version: 1, Scope: governorScope(r.config.httpConfig.Scope), Owner: m.Owner.ValueString(), Request: api.GovernorRequest{ManagementKey: m.Key.ValueString(), Sequence: uint64(m.Sequence.ValueInt64()), Intent: api.GovernorIntent{Kind: "create", Policy: policy}}}
	fingerprint, err := api.GovernorFingerprint(f.Scope, f.Owner, f.Request)
	if err != nil || fingerprint != m.Fingerprint.ValueString() {
		resp.Diagnostics.AddError("Invalid governor plan", governorStale().Error())
		return
	}
	ctx, cancel := context.WithTimeout(ctx, r.config.timeout)
	defer cancel()
	value, err := r.apply(ctx, f, m, "create")
	if value != nil {
		resp.Diagnostics.Append(resp.State.Set(ctx, value)...)
	}
	if err != nil {
		resp.Diagnostics.AddError("Cannot apply governor creation", err.Error())
	}
}
func (r *governorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.State = req.State
	var m, prior governorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Identity observations may be unknown in the plan; prior state supplies them.
	m.ID, m.ResourceID, m.Owner, m.Revision, m.Sequence = prior.ID, prior.ResourceID, prior.Owner, prior.Revision, prior.Sequence
	f, ok := governorPlanned(ctx, req.Private, &resp.Diagnostics)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, r.config.timeout)
	defer cancel()
	value, err := r.apply(ctx, f, m, "update")
	if value != nil {
		resp.Diagnostics.Append(resp.State.Set(ctx, value)...)
	}
	if err != nil {
		resp.Diagnostics.AddError("Cannot apply governor update", err.Error())
	}
}
func (r *governorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var m governorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if m.Protection.ValueBool() {
		resp.Diagnostics.AddError("Governor deletion protected", "Apply deletion_protection = false before retirement.")
		return
	}
	f, ok := governorPlanned(ctx, req.Private, &resp.Diagnostics)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, r.config.timeout)
	defer cancel()
	if _, err := r.apply(ctx, f, m, "retire"); err != nil {
		resp.Diagnostics.AddError("Cannot retire governor", err.Error())
	}
}
func (r *governorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m governorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	a, err := m.anchor(r.config.httpConfig.Scope)
	if err != nil {
		resp.Diagnostics.AddError("Invalid governor state", err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(ctx, r.config.timeout)
	defer cancel()
	v, err := governorInspect(ctx, r.client, m.Key.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Cannot read governor", err.Error())
		return
	}
	if v.Owner != a.Owner || v.Latest == nil || v.Latest.ResourceID == nil || *v.Latest.ResourceID != a.Target {
		resp.Diagnostics.AddError("Governor ownership conflict", governorStale().Error())
		return
	}
	if v.Value == nil {
		if governorPending(v.Latest) || v.Latest.Status != "applied" || v.Latest.Intent.Kind != "retire" {
			resp.Diagnostics.AddError("Governor absence is unresolved", "Retain state until the original operation and exact target absence are authoritative.")
			return
		}
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, governorObserved(ctx, r.config.httpConfig.Scope, *v.Value, *v.Latest))...)
}
func (r *governorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := governorParse(r.config.httpConfig.Scope, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid governor import ID", err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(ctx, r.config.timeout)
	defer cancel()
	g, err := r.client.Governor(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Cannot import governor", err.Error())
		return
	}
	v, err := governorInspect(ctx, r.client, g.ManagementKey)
	if err != nil {
		resp.Diagnostics.AddError("Cannot inspect imported governor", err.Error())
		return
	}
	if v.Value == nil || v.Value.ID != id || v.Latest == nil || governorPending(v.Latest) {
		resp.Diagnostics.AddError("Governor import requires settled ownership", "Recover the original pending operation before importing; import never commits it.")
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, governorObserved(ctx, r.config.httpConfig.Scope, *v.Value, *v.Latest))...)
}
