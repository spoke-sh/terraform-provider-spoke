package provider

import (
	"context"
	"regexp"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

func requiredString(description string, immutable bool, validators ...validator.String) schema.StringAttribute {
	a := schema.StringAttribute{Required: true, MarkdownDescription: description, Validators: validators}
	if immutable {
		a.PlanModifiers = []planmodifier.String{stringplanmodifier.RequiresReplace()}
	}
	return a
}
func enumString(description string, immutable bool, values ...string) schema.StringAttribute {
	return requiredString(description, immutable, stringvalidator.OneOf(values...))
}
func defaultInt(description string, value, min, max int64) schema.Int64Attribute {
	return schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(value), Validators: []validator.Int64{int64validator.Between(min, max)}, MarkdownDescription: description}
}
func serviceAttributes(kind string) map[string]schema.Attribute {
	a := map[string]schema.Attribute{}
	switch kind {
	case "artifact_registry":
		a["format"] = schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("oci"), Validators: []validator.String{stringvalidator.OneOf("oci")}, MarkdownDescription: "Artifact format. The draft supports OCI metadata only."}
		a["capacity_bytes"] = defaultInt("Registry quota in bytes; retained content prevents reduction below use.", api.DefaultCapacity, 1, api.MaxCapacity)
		a["retention_days"] = defaultInt("Retention policy in days; retained artifacts prevent shortening the policy.", 0, 0, 36500)
	case "database", "bucket":
		if kind == "database" {
			a["profile"] = enumString("Database tenancy profile. Changing profile requires replacement; nonempty/protected databases cannot be destroyed.", true, "shared", "dedicated")
		}
		a["capacity_bytes"] = defaultInt("Declared storage capacity in bytes. Capacity can grow; shrink is rejected.", api.DefaultCapacity, 1, api.MaxCapacity)
		a["retention_days"] = defaultInt("Retention policy in days. Stored data prevents shortening the policy.", 0, 0, 36500)
	case "secret_reference":
		a["external_owner"] = requiredString("Opaque identifier of the external secret owner; never a credential.", true, idValidators()...)
		a["external_id"] = requiredString("Opaque external secret identifier. No secret payload is fetched or stored.", true, idValidators()...)
		a["version"] = requiredString("Explicit pinned secret version. Rotation is an intentional reference update.", false, idValidators()...)
	case "service_binding":
		a["consumer_kind"] = schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("container_service"), Validators: []validator.String{stringvalidator.OneOf("container_service", "job")}, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, MarkdownDescription: "Exact consumer type; a binding never starts a job or grants runtime access."}
		a["consumer_id"] = requiredString("Exact container service ID in this environment.", true, idValidators()...)
		a["target_kind"] = enumString("Type of the exact target service.", true, "artifact_registry", "database", "bucket", "secret_reference", "stream", "queue")
		a["target_id"] = requiredString("Exact target ID in this environment. Detach retains that target.", true, idValidators()...)
		a["slot"] = requiredString("Unique connection slot on this consumer. A slot has one binding owner.", true, idValidators()...)
		a["intent"] = enumString("Runtime action required for this connection; an independent explicit grant must authorize it.", false, "artifact_pull", "artifact_push", "database_connect", "bucket_read", "bucket_write", "secret_resolve", "stream_produce", "stream_consume", "queue_produce", "queue_consume")
	case "container_service":
		a["registry_id"] = requiredString("Exact registry ID in this environment.", false, idValidators()...)
		a["repository"] = requiredString("Exact OCI repository path.", false, stringvalidator.LengthBetween(1, 255), stringvalidator.RegexMatches(regexp.MustCompile(`^[a-z0-9]+([._-][a-z0-9]+)*(/[a-z0-9]+([._-][a-z0-9]+)*)*$`), "must be a repository path"))
		a["digest"] = requiredString("Immutable image identity. Build and upload occur outside Terraform.", false, stringvalidator.RegexMatches(regexp.MustCompile(`^sha256:[0-9a-f]{64}$`), "must be a sha256 image digest"))
		a["service_account_id"] = requiredString("Explicit runtime identity in this organization; no implicit grants.", false, idValidators()...)
		a["cpu_millis"] = defaultInt("Declared CPU limit in millicores.", 500, 100, 64000)
		a["memory_mib"] = defaultInt("Declared memory limit in MiB.", 256, 64, 262144)
		a["configuration"] = schema.MapAttribute{Optional: true, Computed: true, ElementType: types.StringType, Default: mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})), Validators: []validator.Map{mapvalidator.SizeAtMost(64), mapvalidator.KeysAre(idValidators()...), mapvalidator.ValueStringsAre(stringvalidator.LengthAtMost(4096))}, MarkdownDescription: "Non-secret configuration only. These strings are stored in Terraform state; use secret_references for secrets."}
		a["secret_references"] = schema.MapAttribute{Optional: true, Computed: true, ElementType: types.StringType, Default: mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})), Validators: []validator.Map{mapvalidator.SizeAtMost(64), mapvalidator.KeysAre(idValidators()...), mapvalidator.ValueStringsAre(idValidators()...)}, MarkdownDescription: "Configuration slot to exact secret-reference ID. Values are identifiers, never secret payloads."}
		a["required_bindings"] = schema.SetAttribute{Optional: true, Computed: true, ElementType: types.StringType, Default: setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})), Validators: []validator.Set{setvalidator.SizeAtMost(64), setvalidator.ValueStringsAre(idValidators()...)}, MarkdownDescription: "Required connection slot names. Bindings refer to the returned consumer ID, avoiding a dependency cycle."}
	case "endpoint":
		a["service_id"] = requiredString("Exact container service ID in this environment.", true, idValidators()...)
		a["port"] = schema.Int64Attribute{Required: true, Validators: []validator.Int64{int64validator.Between(1, 65535)}, MarkdownDescription: "Target service port."}
		a["exposure"] = enumString("Explicit routing exposure. There is no default public route.", false, "private", "public")
		a["tls_policy"] = enumString("Platform-managed TLS policy. Private keys are never Terraform attributes.", false, "managed")
	default:
		if !api.LifecycleKind(kind) {
			return a
		}
		a = lifecycleAttributes(kind)
	}
	for name, description := range map[string]string{"endpoint": "Observed service URL without credentials.", "protocol": "Observed service protocol; SQL wire support does not imply PostgreSQL/MySQL compatibility.", "readiness": "Controller observation: pending, ready or failed. Settled management does not imply readiness.", "reason": "Safe controller reason code; never raw service error text.", "resolved_version": "Observed external secret version; no secret value.", "tls_state": "Observed managed TLS state.", "operation_id": "Original management operation identity."} {
		a[name] = schema.StringAttribute{Computed: true, MarkdownDescription: description}
	}
	for name, description := range map[string]string{"desired_revision": "Observed desired declaration revision.", "effective_revision": "Observed effective configuration revision, including independent bindings and secret references.", "active_revision": "Last active effective revision; failure does not imply rollback."} {
		a[name] = schema.Int64Attribute{Computed: true, MarkdownDescription: description}
	}
	return a
}
func readServiceSettings(ctx context.Context, source attributeReader, kind string) (api.Spec, diag.Diagnostics) {
	if api.LifecycleKind(kind) {
		return readLifecycleSettings(ctx, source, kind)
	}
	var d diag.Diagnostics
	spec := api.Spec{}
	str := func(n string) string {
		var v types.String
		d.Append(source.GetAttribute(ctx, path.Root(n), &v)...)
		return v.ValueString()
	}
	num := func(n string) int64 {
		var v types.Int64
		d.Append(source.GetAttribute(ctx, path.Root(n), &v)...)
		return v.ValueInt64()
	}
	dictionary := func(n string) map[string]string {
		var v types.Map
		d.Append(source.GetAttribute(ctx, path.Root(n), &v)...)
		out := map[string]string{}
		if !v.IsNull() {
			d.Append(v.ElementsAs(ctx, &out, false)...)
		}
		return out
	}
	switch kind {
	case "artifact_registry":
		spec.Registry = &api.RegistrySpec{Format: str("format"), CapacityBytes: num("capacity_bytes"), RetentionDays: num("retention_days")}
	case "database", "bucket":
		profile := "object"
		if kind == "database" {
			profile = str("profile")
		}
		spec.Storage = &api.StorageSpec{Profile: profile, CapacityBytes: num("capacity_bytes"), RetentionDays: num("retention_days")}
	case "secret_reference":
		spec.Secret = &api.SecretSpec{ExternalOwner: str("external_owner"), ExternalID: str("external_id"), Version: str("version")}
	case "service_binding":
		spec.Binding = &api.BindingSpec{ConsumerKind: str("consumer_kind"), ConsumerID: str("consumer_id"), TargetKind: str("target_kind"), TargetID: str("target_id"), Slot: str("slot"), Intent: str("intent")}
	case "container_service":
		var v types.Set
		d.Append(source.GetAttribute(ctx, path.Root("required_bindings"), &v)...)
		slots := []string{}
		if !v.IsNull() {
			d.Append(v.ElementsAs(ctx, &slots, false)...)
		}
		sort.Strings(slots)
		spec.Container = &api.ContainerSpec{RegistryID: str("registry_id"), Repository: str("repository"), Digest: str("digest"), ServiceAccountID: str("service_account_id"), CPU: num("cpu_millis"), Memory: num("memory_mib"), Configuration: dictionary("configuration"), SecretReferences: dictionary("secret_references"), RequiredBindings: slots}
	case "endpoint":
		spec.Endpoint = &api.EndpointSpec{ServiceID: str("service_id"), Port: num("port"), Exposure: str("exposure"), TLS: str("tls_policy")}
	}
	return spec, d
}
func observedServiceSettings(o api.Object) map[string]any {
	a := map[string]any{}
	if !api.ServiceKind(o.ResourceKind) && o.ResourceKind != "bucket" {
		return a
	}
	if s := o.Registry; s != nil {
		a["format"] = s.Format
		a["capacity_bytes"] = s.CapacityBytes
		a["retention_days"] = s.RetentionDays
	}
	if s := o.Storage; s != nil {
		a["capacity_bytes"] = s.CapacityBytes
		a["retention_days"] = s.RetentionDays
		if o.ResourceKind == "database" {
			a["profile"] = s.Profile
		}
	}
	if s := o.Secret; s != nil {
		a["external_owner"] = s.ExternalOwner
		a["external_id"] = s.ExternalID
		a["version"] = s.Version
	}
	if s := o.Binding; s != nil {
		a["consumer_kind"] = s.ConsumerKind
		a["consumer_id"] = s.ConsumerID
		a["target_kind"] = s.TargetKind
		a["target_id"] = s.TargetID
		a["slot"] = s.Slot
		a["intent"] = s.Intent
	}
	if s := o.Container; s != nil {
		a["registry_id"] = s.RegistryID
		a["repository"] = s.Repository
		a["digest"] = s.Digest
		a["service_account_id"] = s.ServiceAccountID
		a["cpu_millis"] = s.CPU
		a["memory_mib"] = s.Memory
		a["configuration"] = s.Configuration
		a["secret_references"] = s.SecretReferences
		a["required_bindings"] = s.RequiredBindings
	}
	if s := o.Endpoint; s != nil {
		a["service_id"] = s.ServiceID
		a["port"] = s.Port
		a["exposure"] = s.Exposure
		a["tls_policy"] = s.TLS
	}
	observeLifecycle(o, a)
	if s := o.Status; s != nil {
		a["endpoint"] = s.Endpoint
		a["protocol"] = s.Protocol
		a["readiness"] = s.Readiness
		a["reason"] = s.Reason
		a["resolved_version"] = s.ResolvedVersion
		a["tls_state"] = s.TLSState
		a["operation_id"] = s.OperationID
		a["desired_revision"] = s.DesiredRevision
		a["effective_revision"] = s.EffectiveRevision
		a["active_revision"] = s.ActiveRevision
	} else {
		for _, key := range []string{"endpoint", "protocol", "readiness", "reason", "resolved_version", "tls_state", "operation_id"} {
			a[key] = types.StringNull()
		}
		for _, key := range []string{"desired_revision", "effective_revision", "active_revision"} {
			a[key] = types.Int64Null()
		}
	}
	return a
}
