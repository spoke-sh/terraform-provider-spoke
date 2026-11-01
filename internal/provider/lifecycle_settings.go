package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

type scheduleValidator struct{ zone bool }

func (v scheduleValidator) Description(context.Context) string {
	if v.zone {
		return "must be an explicit IANA time zone"
	}
	return "must be a standard five-field cron schedule"
}
func (v scheduleValidator) MarkdownDescription(ctx context.Context) string { return v.Description(ctx) }
func (v scheduleValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	var err error
	if v.zone {
		_, err = api.ParseSchedule("* * * * *", req.ConfigValue.ValueString())
	} else {
		_, err = api.ParseSchedule(req.ConfigValue.ValueString(), "UTC")
	}
	if err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid schedule", v.Description(ctx))
	}
}
func enabledAttribute() schema.BoolAttribute {
	return schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false), MarkdownDescription: "Explicitly enable future scheduler admission. Disabling preserves accepted executions and recovery points."}
}
func lifecycleAttributes(kind string) map[string]schema.Attribute {
	a := map[string]schema.Attribute{}
	id := func(n, d string, immutable bool) { a[n] = requiredString(d, immutable, idValidators()...) }
	switch kind {
	case "job":
		a = serviceAttributes("container_service")
		a["max_concurrent"] = defaultInt("Maximum concurrent accepted executions.", 1, 1, 64)
		a["retry_limit"] = defaultInt("Maximum retries of a known failed attempt. Unknown effects cannot be retried as new work.", 0, 0, 10)
		a["timeout_seconds"] = defaultInt("Execution deadline. Timeout does not prove rollback of external effects.", 3600, 1, 86400)
	case "schedule":
		id("job_id", "Exact job definition ID in this environment.", true)
		a["cron"] = requiredString("Standard five-field cron expression. No execution occurs during plan or refresh.", false, scheduleValidator{})
		a["time_zone"] = requiredString("Explicit IANA time zone. Nonexistent wall times are skipped; repeated wall times are distinct UTC occurrences.", false, scheduleValidator{zone: true})
		a["enabled"] = enabledAttribute()
		a["missed_run"] = enumString("Bounded missed-run policy.", false, "skip", "catch_up_once")
		a["overlap"] = enumString("Admission when this job already has active or uncertain work.", false, "allow", "forbid")
	case "stream", "queue":
		a["capacity_bytes"] = defaultInt("Messaging capacity; retained data cannot be purged by changing capacity.", api.DefaultCapacity, 1, api.MaxCapacity)
		a["retention_days"] = defaultInt("Retention policy. Shortening with retained data is refused.", 7, 0, 36500)
		if kind == "stream" {
			a["partitions"] = defaultInt("Immutable partition count; changes require empty, unprotected replacement.", 1, 1, 64)
			a["ordering"] = enumString("Ordering profile; no global ordering across partitions.", true, "partition")
		} else {
			a["ordering"] = enumString("FIFO queue profile. Atomic dead-letter transfer and mixed transactions are unsupported.", true, "fifo")
			a["visibility_seconds"] = defaultInt("Lease visibility window in seconds.", 30, 1, 86400)
			a["max_attempts"] = defaultInt("Delivery attempt limit. Exhaustion preserves inspectable work.", 5, 1, 100)
		}
	case "volume":
		a["profile"] = enumString("Persistent storage profile. Buckets are not volumes.", true, "filesystem", "block")
		a["capacity_bytes"] = defaultInt("Declared volume capacity. Growth is allowed; shrink is refused.", api.DefaultCapacity, 1, api.MaxCapacity)
		a["access_mode"] = enumString("Writer profile; an attachment never bypasses fencing.", true, "single_writer")
		id("region", "Exact supported storage region; moving a volume requires a separately qualified service.", true)
	case "volume_attachment":
		id("volume_id", "Exact retained volume ID.", true)
		id("service_id", "Exact workload ID.", true)
		a["mount_path"] = requiredString("Absolute path beneath /data/. Detach retains the volume and fences the old writer.", true)
	case "custom_domain":
		a["domain"] = requiredString("Lowercase ASCII DNS name. External DNS records remain with their DNS provider.", true)
		id("endpoint_id", "Exact endpoint ID; certificate keys stay outside Terraform.", true)
	case "private_connection":
		id("service_id", "Exact local service ID. Failure never enables public exposure.", true)
		id("peer_organization_id", "Explicit peer organization; peer consent is issued externally.", true)
		id("peer_network_id", "Exact peer network ID.", true)
		a["cidr"] = requiredString("Canonical private IPv4 prefix, /16 through /30. Overlapping routes on this service are refused.", true)
	case "backup_policy":
		id("database_id", "Exact source database ID.", true)
		a["cron"] = requiredString("Five-field backup schedule; reading a policy never starts backup.", false, scheduleValidator{})
		a["time_zone"] = requiredString("Explicit IANA time zone.", false, scheduleValidator{zone: true})
		a["retention_days"] = defaultInt("Retained recovery-point policy. Deletion and shortening cannot invalidate dependent points.", 7, 1, 36500)
		a["enabled"] = enabledAttribute()
	case "database_clone":
		id("source_database_id", "Exact source database in this environment; cross-scope cloning is unsupported in this draft.", true)
		id("recovery_point_id", "Exact completed immutable recovery point. No latest selector or retry retargeting.", true)
		a["capacity_bytes"] = defaultInt("Clone capacity; changing this draft's immutable clone declaration requires protected replacement.", api.DefaultCapacity, 1, api.MaxCapacity)
	case "scaling_policy":
		id("service_id", "Exact service ID. One policy owns capacity; deleting it freezes the last observed count.", true)
		a["min_instances"] = defaultInt("Minimum capacity.", 1, 0, 1000)
		a["max_instances"] = defaultInt("Maximum capacity; must be at least min_instances.", 3, 1, 1000)
		a["target_cpu_percent"] = defaultInt("CPU target percentage. Stale telemetry preserves current capacity.", 70, 1, 100)
		a["cooldown_seconds"] = defaultInt("Minimum time between capacity transitions.", 60, 1, 3600)
	case "placement_policy":
		id("service_id", "Exact service ID. Deleting policy retains current location and workload.", true)
		id("region", "Desired supported region. Incompatible volume placement is refused.", false)
		a["strategy"] = enumString("Qualified movement strategy. Live stateful migration is unsupported.", false, "stopped")
	}
	if kind == "stream" || kind == "database_clone" {
		name := "partitions"
		if kind == "database_clone" {
			name = "capacity_bytes"
		}
		value := a[name].(schema.Int64Attribute)
		value.PlanModifiers = []planmodifier.Int64{int64planmodifier.RequiresReplace()}
		a[name] = value
	}
	if !api.LifecycleKind(kind) {
		return a
	}
	if kind == "job" || kind == "schedule" {
		for _, n := range []string{"execution_count", "active_executions"} {
			a[n] = schema.Int64Attribute{Computed: true, MarkdownDescription: "Bounded scheduler observation; executions are not managed Terraform resources."}
		}
		a["last_execution_id"] = schema.StringAttribute{Computed: true, MarkdownDescription: "Original last admitted execution identity; no execution output or secret values."}
	}
	if kind == "volume_attachment" {
		a["writer_epoch"] = schema.Int64Attribute{Computed: true, MarkdownDescription: "Observed writer fence. A timeout cannot establish detachment."}
		a["attached"] = schema.BoolAttribute{Computed: true, MarkdownDescription: "Observed attachment completion."}
	}
	if kind == "custom_domain" {
		a["challenge"] = schema.StringAttribute{Computed: true, MarkdownDescription: "Public verification challenge for external DNS configuration."}
		a["challenge_expires"] = schema.Int64Attribute{Computed: true, MarkdownDescription: "Challenge expiry as Unix seconds."}
		a["verified"] = schema.BoolAttribute{Computed: true, MarkdownDescription: "Observed current domain verification."}
	}
	if kind == "private_connection" {
		a["peer_accepted"] = schema.BoolAttribute{Computed: true, MarkdownDescription: "Observed peer consent; the local declaration cannot grant it."}
	}
	if kind == "backup_policy" {
		a["latest_recovery_point_id"] = schema.StringAttribute{Computed: true, MarkdownDescription: "Observed completed recovery point; a configured policy is not a backup."}
	}
	if kind == "backup_policy" || kind == "database_clone" {
		a["source_revision"] = schema.Int64Attribute{Computed: true, MarkdownDescription: "Exact source revision attributed to the recovery point."}
	}
	if kind == "scaling_policy" {
		a["observed_instances"] = schema.Int64Attribute{Computed: true, MarkdownDescription: "Controller-owned count, independent of Terraform policy intent."}
	}
	if kind == "placement_policy" {
		a["observed_region"] = schema.StringAttribute{Computed: true, MarkdownDescription: "Controller-owned location, independent of desired region."}
	}
	return a
}
func readLifecycleSettings(ctx context.Context, source attributeReader, kind string) (api.Spec, diag.Diagnostics) {
	var d diag.Diagnostics
	s := api.Spec{}
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
	flag := func(n string) bool {
		var v types.Bool
		d.Append(source.GetAttribute(ctx, path.Root(n), &v)...)
		return v.ValueBool()
	}
	switch kind {
	case "job":
		base, more := readServiceSettings(ctx, source, "container_service")
		d.Append(more...)
		s.Job = &api.JobSpec{Workload: *base.Container, MaxConcurrent: num("max_concurrent"), RetryLimit: num("retry_limit"), TimeoutSeconds: num("timeout_seconds")}
	case "schedule":
		s.Schedule = &api.ScheduleSpec{JobID: str("job_id"), Cron: str("cron"), TimeZone: str("time_zone"), Enabled: flag("enabled"), MissedRun: str("missed_run"), Overlap: str("overlap")}
	case "stream":
		s.Stream = &api.StreamSpec{Partitions: num("partitions"), Ordering: str("ordering"), CapacityBytes: num("capacity_bytes"), RetentionDays: num("retention_days")}
	case "queue":
		s.Queue = &api.QueueSpec{Ordering: str("ordering"), CapacityBytes: num("capacity_bytes"), RetentionDays: num("retention_days"), VisibilitySeconds: num("visibility_seconds"), MaxAttempts: num("max_attempts")}
	case "volume":
		s.Volume = &api.VolumeSpec{Profile: str("profile"), CapacityBytes: num("capacity_bytes"), AccessMode: str("access_mode"), Region: str("region")}
	case "volume_attachment":
		s.Attachment = &api.AttachmentSpec{VolumeID: str("volume_id"), ServiceID: str("service_id"), MountPath: str("mount_path")}
	case "custom_domain":
		s.Domain = &api.DomainSpec{Domain: str("domain"), EndpointID: str("endpoint_id")}
	case "private_connection":
		s.Connection = &api.ConnectionSpec{ServiceID: str("service_id"), PeerOrganizationID: str("peer_organization_id"), PeerNetworkID: str("peer_network_id"), CIDR: str("cidr")}
	case "backup_policy":
		s.Backup = &api.BackupSpec{DatabaseID: str("database_id"), Cron: str("cron"), TimeZone: str("time_zone"), RetentionDays: num("retention_days"), Enabled: flag("enabled")}
	case "database_clone":
		s.Clone = &api.CloneSpec{SourceDatabaseID: str("source_database_id"), RecoveryPointID: str("recovery_point_id"), CapacityBytes: num("capacity_bytes")}
	case "scaling_policy":
		s.Scaling = &api.ScalingSpec{ServiceID: str("service_id"), MinInstances: num("min_instances"), MaxInstances: num("max_instances"), TargetCPUPercent: num("target_cpu_percent"), CooldownSeconds: num("cooldown_seconds")}
	case "placement_policy":
		s.Placement = &api.PlacementSpec{ServiceID: str("service_id"), Region: str("region"), Strategy: str("strategy")}
	}
	return s, d
}
func observeLifecycle(o api.Object, a map[string]any) {
	if v := o.Job; v != nil {
		copy := o
		copy.Job = nil
		copy.Container = &v.Workload
		copy.ResourceKind = "container_service"
		for k, val := range observedServiceSettings(copy) {
			a[k] = val
		}
		a["max_concurrent"] = v.MaxConcurrent
		a["retry_limit"] = v.RetryLimit
		a["timeout_seconds"] = v.TimeoutSeconds
	}
	if v := o.Schedule; v != nil {
		a["job_id"] = v.JobID
		a["cron"] = v.Cron
		a["time_zone"] = v.TimeZone
		a["enabled"] = v.Enabled
		a["missed_run"] = v.MissedRun
		a["overlap"] = v.Overlap
	}
	if v := o.Stream; v != nil {
		a["partitions"] = v.Partitions
		a["ordering"] = v.Ordering
		a["capacity_bytes"] = v.CapacityBytes
		a["retention_days"] = v.RetentionDays
	}
	if v := o.Queue; v != nil {
		a["ordering"] = v.Ordering
		a["capacity_bytes"] = v.CapacityBytes
		a["retention_days"] = v.RetentionDays
		a["visibility_seconds"] = v.VisibilitySeconds
		a["max_attempts"] = v.MaxAttempts
	}
	if v := o.Volume; v != nil {
		a["profile"] = v.Profile
		a["capacity_bytes"] = v.CapacityBytes
		a["access_mode"] = v.AccessMode
		a["region"] = v.Region
	}
	if v := o.Attachment; v != nil {
		a["volume_id"] = v.VolumeID
		a["service_id"] = v.ServiceID
		a["mount_path"] = v.MountPath
	}
	if v := o.Domain; v != nil {
		a["domain"] = v.Domain
		a["endpoint_id"] = v.EndpointID
	}
	if v := o.Connection; v != nil {
		a["service_id"] = v.ServiceID
		a["peer_organization_id"] = v.PeerOrganizationID
		a["peer_network_id"] = v.PeerNetworkID
		a["cidr"] = v.CIDR
	}
	if v := o.Backup; v != nil {
		a["database_id"] = v.DatabaseID
		a["cron"] = v.Cron
		a["time_zone"] = v.TimeZone
		a["retention_days"] = v.RetentionDays
		a["enabled"] = v.Enabled
	}
	if v := o.Clone; v != nil {
		a["source_database_id"] = v.SourceDatabaseID
		a["recovery_point_id"] = v.RecoveryPointID
		a["capacity_bytes"] = v.CapacityBytes
	}
	if v := o.Scaling; v != nil {
		a["service_id"] = v.ServiceID
		a["min_instances"] = v.MinInstances
		a["max_instances"] = v.MaxInstances
		a["target_cpu_percent"] = v.TargetCPUPercent
		a["cooldown_seconds"] = v.CooldownSeconds
	}
	if v := o.Placement; v != nil {
		a["service_id"] = v.ServiceID
		a["region"] = v.Region
		a["strategy"] = v.Strategy
	}
	if s := o.Status; s != nil {
		if o.Job != nil || o.Schedule != nil {
			a["execution_count"] = s.ExecutionCount
			a["active_executions"] = s.ActiveExecutions
			a["last_execution_id"] = s.LastExecutionID
		}
		if o.Attachment != nil {
			a["writer_epoch"] = s.WriterEpoch
			a["attached"] = s.Attached
		}
		if o.Domain != nil {
			a["challenge"] = s.Challenge
			a["challenge_expires"] = s.ChallengeExpires
			a["verified"] = s.Verified
		}
		if o.Connection != nil {
			a["peer_accepted"] = s.PeerAccepted
		}
		if o.Backup != nil {
			a["latest_recovery_point_id"] = s.RecoveryPointID
		}
		if o.Backup != nil || o.Clone != nil {
			a["source_revision"] = s.SourceRevision
		}
		if o.Scaling != nil {
			a["observed_instances"] = s.ObservedCapacity
		}
		if o.Placement != nil {
			a["observed_region"] = s.ObservedRegion
		}
	}
}
