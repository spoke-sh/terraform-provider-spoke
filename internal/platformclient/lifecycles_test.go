package platformclient

import (
	"testing"
	"time"
)

func TestScheduleBoundsAndDST(t *testing.T) {
	for _, v := range [][2]string{{"@every 1s", "UTC"}, {"0 0 0 * * *", "UTC"}, {"* * * * *", "Local"}, {"0 0 31 2 *", "UTC"}, {"* * * * *", ""}, {"* * * * *", "not/a-zone"}} {
		if _, err := ParseSchedule(v[0], v[1]); err == nil {
			t.Fatal("accepted unsupported schedule", v)
		}
	}
	// Spring gap is skipped; the fall fold has two distinct UTC occurrences.
	spring, _ := ParseSchedule("30 2 * * *", "America/New_York")
	from := time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC)
	if next := spring.Next(from); next != time.Date(2026, 3, 9, 6, 30, 0, 0, time.UTC) {
		t.Fatal(next)
	}
	fall, _ := ParseSchedule("30 1 * * *", "America/New_York")
	first := fall.Next(time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC))
	second := fall.Next(first)
	if first != time.Date(2026, 11, 1, 5, 30, 0, 0, time.UTC) || second != time.Date(2026, 11, 1, 6, 30, 0, 0, time.UTC) {
		t.Fatal(first, second)
	}
}
func TestLifecycleSpecBoundaries(t *testing.T) {
	cases := []struct {
		kind string
		spec Spec
	}{
		{"job", Spec{Job: &JobSpec{MaxConcurrent: 1}}},
		{"stream", Spec{Stream: &StreamSpec{Partitions: 65, Ordering: "partition", CapacityBytes: 1}}},
		{"queue", Spec{Queue: &QueueSpec{Ordering: "fifo", CapacityBytes: 1, VisibilitySeconds: 0, MaxAttempts: 1}}},
		{"volume_attachment", Spec{Attachment: &AttachmentSpec{VolumeID: "volume", ServiceID: "service", MountPath: "/data/../secret"}}},
		{"custom_domain", Spec{Domain: &DomainSpec{Domain: "UPPER.example.com", EndpointID: "endpoint"}}},
		{"private_connection", Spec{Connection: &ConnectionSpec{ServiceID: "service", PeerOrganizationID: "org", PeerNetworkID: "peer", CIDR: "10.0.0.1/24"}}},
		{"scaling_policy", Spec{Scaling: &ScalingSpec{ServiceID: "service", MinInstances: 5, MaxInstances: 3, TargetCPUPercent: 70, CooldownSeconds: 60}}},
		{"placement_policy", Spec{Placement: &PlacementSpec{ServiceID: "service", Region: "region-a", Strategy: "live"}}},
	}
	for _, v := range cases {
		if v.spec.Valid(v.kind) {
			t.Fatal("invalid settings accepted", v.kind)
		}
	}
}
