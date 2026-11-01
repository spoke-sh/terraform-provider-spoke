package mockplatform

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

func TestCutoverRejectsOldLedgerAndFingerprint(t *testing.T) {
	s := New(t)
	s.Provision(t, "bucket", "current", api.Spec{Name: "current", Storage: &api.StorageSpec{Profile: "object", CapacityBytes: api.DefaultCapacity}})
	before := s.SnapshotLedger()
	for _, version := range []string{"mock-v1", "mock-v2", "mock-v3", "mock-v4", ""} {
		old := bytes.Replace(before, []byte(api.APIVersion), []byte(version), 1)
		if _, err := decodeLedger(old); err == nil {
			t.Fatalf("accepted ledger %q", version)
		}
	}
	var changed ledger
	if err := json.Unmarshal(before, &changed); err != nil {
		t.Fatal(err)
	}
	for id, op := range changed.Operations {
		op.Fingerprint = strings.Repeat("0", 64)
		changed.Operations[id] = op
	}
	data, _ := json.Marshal(changed)
	if _, err := decodeLedger(data); err == nil {
		t.Fatal("accepted incompatible receipt fingerprint")
	}
	if !bytes.Equal(before, s.SnapshotLedger()) {
		t.Fatal("rejection changed current work")
	}
	s.Restart(t)
}

func TestCutoverRejectsOldWireShapes(t *testing.T) {
	s := New(t)
	for _, body := range []string{
		`{"kind":"create","management_key":"old","target_id":"","expected_revision":0,"expected_generation":0,"spec":{"name":"old","deletion_protection":false}}`,
		`{"kind":"create","resource_kind":"bucket","management_key":"old","target_id":"","expected_revision":0,"expected_generation":0,"spec":{"name":"old","deletion_protection":false}}`,
	} {
		if status, _ := raw(t, s, "POST", s.Base()+"/buckets", body); status != 400 {
			t.Fatalf("accepted old wire shape: %d", status)
		}
	}
	for _, version := range []string{"v1", "v2", "v3", "v4"} {
		if _, _, ok := route(strings.Replace(s.Base(), "v5", version, 1) + "/buckets"); ok {
			t.Fatal("accepted old route")
		}
	}
	if s.OperationCount() != 0 || s.BucketCount() != 0 {
		t.Fatal("old input mutated service")
	}
}
