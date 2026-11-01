package mockplatform

import (
	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
	"testing"
	"time"
)

// LifecycleDependencies adds independent owners required by the provisional
// later families. Creating declarations still goes through the real HTTP client.
func (s *Server) LifecycleDependencies(t testing.TB) ServiceDependencies {
	d := s.ServiceDependencies(t)
	d.Job = s.Provision(t, "job", "fixture-job", d.Spec("job"))
	d.Volume = s.Provision(t, "volume", "fixture-volume", d.Spec("volume"))
	d.Endpoint = s.Provision(t, "endpoint", "fixture-endpoint", d.Spec("endpoint"))
	backup := d.Spec("backup_policy")
	backup.Backup.Enabled = true
	d.Backup = s.Provision(t, "backup_policy", "fixture-backup", backup)
	if err := s.AdvanceTime(time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	d.Backup, _ = s.Object(s.Scope, "backup_policy", d.Backup.ID)
	return d
}
func (d ServiceDependencies) lifecycleSpec(kind string) api.Spec {
	s := api.Spec{Name: "example"}
	switch kind {
	case "job":
		s.Job = &api.JobSpec{Workload: *d.Spec("container_service").Container, MaxConcurrent: 1, RetryLimit: 1, TimeoutSeconds: 3600}
	case "schedule":
		s.Schedule = &api.ScheduleSpec{JobID: d.Job.ID, Cron: "* * * * *", TimeZone: "UTC", MissedRun: "catch_up_once", Overlap: "forbid"}
	case "stream":
		s.Stream = &api.StreamSpec{Partitions: 1, Ordering: "partition", CapacityBytes: api.DefaultCapacity, RetentionDays: 7}
	case "queue":
		s.Queue = &api.QueueSpec{Ordering: "fifo", CapacityBytes: api.DefaultCapacity, RetentionDays: 7, VisibilitySeconds: 30, MaxAttempts: 5}
	case "volume":
		s.Volume = &api.VolumeSpec{Profile: "filesystem", CapacityBytes: api.DefaultCapacity, AccessMode: "single_writer", Region: "region-a"}
	case "volume_attachment":
		s.Attachment = &api.AttachmentSpec{VolumeID: d.Volume.ID, ServiceID: d.Consumer.ID, MountPath: "/data/files"}
	case "custom_domain":
		s.Domain = &api.DomainSpec{Domain: "app.example.test", EndpointID: d.Endpoint.ID}
	case "private_connection":
		s.Connection = &api.ConnectionSpec{ServiceID: d.Consumer.ID, PeerOrganizationID: "peer-org", PeerNetworkID: "network", CIDR: "10.0.0.0/24"}
	case "backup_policy":
		s.Backup = &api.BackupSpec{DatabaseID: d.Database.ID, Cron: "* * * * *", TimeZone: "UTC", RetentionDays: 7}
	case "database_clone":
		s.Clone = &api.CloneSpec{SourceDatabaseID: d.Database.ID, RecoveryPointID: d.Backup.Status.RecoveryPointID, CapacityBytes: api.DefaultCapacity}
	case "scaling_policy":
		s.Scaling = &api.ScalingSpec{ServiceID: d.Consumer.ID, MinInstances: 1, MaxInstances: 3, TargetCPUPercent: 70, CooldownSeconds: 60}
	case "placement_policy":
		s.Placement = &api.PlacementSpec{ServiceID: d.Consumer.ID, Region: "region-a", Strategy: "stopped"}
	default:
		panic("unknown lifecycle kind")
	}
	return s
}
