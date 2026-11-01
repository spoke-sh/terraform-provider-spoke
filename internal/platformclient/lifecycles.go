package platformclient

import (
	"net/netip"
	"regexp"
	"slices"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/robfig/cron/v3"
)

var LifecycleCollections = map[string]string{
	"job": "jobs", "schedule": "schedules", "stream": "streams", "queue": "queues",
	"volume": "volumes", "volume_attachment": "volume_attachments", "custom_domain": "custom_domains",
	"private_connection": "private_connections", "backup_policy": "backup_policies", "database_clone": "database_clones",
	"scaling_policy": "scaling_policies", "placement_policy": "placement_policies",
}

func LifecycleKinds() []string {
	return []string{"job", "schedule", "stream", "queue", "volume", "volume_attachment", "custom_domain", "private_connection", "backup_policy", "database_clone", "scaling_policy", "placement_policy"}
}
func LifecycleKind(kind string) bool { _, ok := LifecycleCollections[kind]; return ok }
func ManagedServiceKinds() []string {
	out := []string{"artifact_registry", "database", "secret_reference", "service_binding", "container_service", "endpoint"}
	return append(out, LifecycleKinds()...)
}

type JobSpec struct {
	Workload       ContainerSpec `json:"workload"`
	MaxConcurrent  int64         `json:"max_concurrent"`
	RetryLimit     int64         `json:"retry_limit"`
	TimeoutSeconds int64         `json:"timeout_seconds"`
}
type ScheduleSpec struct {
	JobID     string `json:"job_id"`
	Cron      string `json:"cron"`
	TimeZone  string `json:"time_zone"`
	Enabled   bool   `json:"enabled"`
	MissedRun string `json:"missed_run"`
	Overlap   string `json:"overlap"`
}
type StreamSpec struct {
	Partitions    int64  `json:"partitions"`
	Ordering      string `json:"ordering"`
	CapacityBytes int64  `json:"capacity_bytes"`
	RetentionDays int64  `json:"retention_days"`
}
type QueueSpec struct {
	Ordering          string `json:"ordering"`
	CapacityBytes     int64  `json:"capacity_bytes"`
	RetentionDays     int64  `json:"retention_days"`
	VisibilitySeconds int64  `json:"visibility_seconds"`
	MaxAttempts       int64  `json:"max_attempts"`
}
type VolumeSpec struct {
	Profile       string `json:"profile"`
	CapacityBytes int64  `json:"capacity_bytes"`
	AccessMode    string `json:"access_mode"`
	Region        string `json:"region"`
}
type AttachmentSpec struct {
	VolumeID  string `json:"volume_id"`
	ServiceID string `json:"service_id"`
	MountPath string `json:"mount_path"`
}
type DomainSpec struct {
	Domain     string `json:"domain"`
	EndpointID string `json:"endpoint_id"`
}
type ConnectionSpec struct {
	ServiceID          string `json:"service_id"`
	PeerOrganizationID string `json:"peer_organization_id"`
	PeerNetworkID      string `json:"peer_network_id"`
	CIDR               string `json:"cidr"`
}
type BackupSpec struct {
	DatabaseID    string `json:"database_id"`
	Cron          string `json:"cron"`
	TimeZone      string `json:"time_zone"`
	RetentionDays int64  `json:"retention_days"`
	Enabled       bool   `json:"enabled"`
}
type CloneSpec struct {
	SourceDatabaseID string `json:"source_database_id"`
	RecoveryPointID  string `json:"recovery_point_id"`
	CapacityBytes    int64  `json:"capacity_bytes"`
}
type ScalingSpec struct {
	ServiceID        string `json:"service_id"`
	MinInstances     int64  `json:"min_instances"`
	MaxInstances     int64  `json:"max_instances"`
	TargetCPUPercent int64  `json:"target_cpu_percent"`
	CooldownSeconds  int64  `json:"cooldown_seconds"`
}
type PlacementSpec struct {
	ServiceID string `json:"service_id"`
	Region    string `json:"region"`
	Strategy  string `json:"strategy"`
}

// ParseSchedule parses only standard five-field cron with an explicit IANA zone.
// No scheduler, worker, I/O to the service or execution is started.
func ParseSchedule(expression, zone string) (cron.Schedule, error) {
	if len(expression) > 128 || len(strings.Fields(expression)) != 5 || zone == "Local" || zone == "" || len(zone) > 128 {
		return nil, errSchedule
	}
	loc, err := time.LoadLocation(zone)
	if err != nil {
		return nil, errSchedule
	}
	parsed, err := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow).Parse(expression)
	if err != nil {
		return nil, errSchedule
	}
	spec := parsed.(*cron.SpecSchedule)
	spec.Location = loc
	if spec.Next(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)).IsZero() {
		return nil, errSchedule
	}
	return spec, nil
}

var errSchedule = &APIError{Code: "invalid_schedule", Status: 400}
var domainPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)+$`)

func ValidDomain(v string) bool { return len(v) <= 253 && domainPattern.MatchString(v) }
func (s Spec) validLifecycle(kind string) bool {
	switch kind {
	case "job":
		j := s.Job
		return j != nil && (Spec{Container: &j.Workload}).Valid("container_service") && j.MaxConcurrent >= 1 && j.MaxConcurrent <= 64 && j.RetryLimit >= 0 && j.RetryLimit <= 10 && j.TimeoutSeconds >= 1 && j.TimeoutSeconds <= 86400
	case "schedule":
		v := s.Schedule
		if v == nil {
			return false
		}
		_, err := ParseSchedule(v.Cron, v.TimeZone)
		return err == nil && ValidID(v.JobID) && slices.Contains([]string{"skip", "catch_up_once"}, v.MissedRun) && slices.Contains([]string{"allow", "forbid"}, v.Overlap)
	case "stream":
		v := s.Stream
		return v != nil && v.Partitions >= 1 && v.Partitions <= 64 && v.Ordering == "partition" && capacityValid(v.CapacityBytes, v.RetentionDays)
	case "queue":
		v := s.Queue
		return v != nil && v.Ordering == "fifo" && capacityValid(v.CapacityBytes, v.RetentionDays) && v.VisibilitySeconds >= 1 && v.VisibilitySeconds <= 86400 && v.MaxAttempts >= 1 && v.MaxAttempts <= 100
	case "volume":
		v := s.Volume
		return v != nil && slices.Contains([]string{"filesystem", "block"}, v.Profile) && capacityValid(v.CapacityBytes, 0) && v.AccessMode == "single_writer" && ValidID(v.Region)
	case "volume_attachment":
		v := s.Attachment
		return v != nil && ValidID(v.VolumeID) && ValidID(v.ServiceID) && len(v.MountPath) <= 256 && strings.HasPrefix(v.MountPath, "/data/") && !strings.Contains(v.MountPath, "..") && !strings.ContainsAny(v.MountPath, "\x00\r\n\t ")
	case "custom_domain":
		v := s.Domain
		return v != nil && ValidDomain(v.Domain) && ValidID(v.EndpointID)
	case "private_connection":
		v := s.Connection
		if v == nil {
			return false
		}
		p, err := netip.ParsePrefix(v.CIDR)
		return err == nil && p == p.Masked() && p.Addr().IsPrivate() && p.Bits() >= 16 && p.Bits() <= 30 && p.Addr().Is4() && ValidID(v.ServiceID) && ValidID(v.PeerOrganizationID) && ValidID(v.PeerNetworkID)
	case "backup_policy":
		v := s.Backup
		if v == nil {
			return false
		}
		_, err := ParseSchedule(v.Cron, v.TimeZone)
		return err == nil && ValidID(v.DatabaseID) && v.RetentionDays >= 1 && v.RetentionDays <= 36500
	case "database_clone":
		v := s.Clone
		return v != nil && ValidID(v.SourceDatabaseID) && ValidID(v.RecoveryPointID) && capacityValid(v.CapacityBytes, 0)
	case "scaling_policy":
		v := s.Scaling
		return v != nil && ValidID(v.ServiceID) && v.MinInstances >= 0 && v.MaxInstances >= 1 && v.MaxInstances <= 1000 && v.MinInstances <= v.MaxInstances && v.TargetCPUPercent >= 1 && v.TargetCPUPercent <= 100 && v.CooldownSeconds >= 1 && v.CooldownSeconds <= 3600
	case "placement_policy":
		v := s.Placement
		return v != nil && ValidID(v.ServiceID) && ValidID(v.Region) && v.Strategy == "stopped"
	}
	return false
}
