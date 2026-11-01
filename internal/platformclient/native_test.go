package platformclient

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
)

func TestNativeGoldenIntents(t *testing.T) {
	var fixture struct {
		Scope   nativeScope `json:"scope"`
		Vectors []struct {
			Kind        byte   `json:"kind"`
			Key         string `json:"key"`
			Generation  uint32 `json:"generation"`
			Revision    int64  `json:"revision"`
			Bucket      uint64 `json:"bucket"`
			Name        string `json:"name"`
			Protection  bool   `json:"protection"`
			Capacity    int64  `json:"capacity_bytes"`
			Retention   int64  `json:"retention_days"`
			Payload     string `json:"payload"`
			Fingerprint string `json:"fingerprint"`
		} `json:"vectors"`
	}
	raw, e := os.ReadFile("../../testdata/native-v2/vectors.json")
	if e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(raw, &fixture); e != nil {
		t.Fatal(e)
	}
	n := nativeClient{scope: fixture.Scope}
	for _, v := range fixture.Vectors {
		raw, e := hex.DecodeString(v.Payload)
		if e != nil {
			t.Fatal(e)
		}
		var want [64]byte
		copy(want[:], raw)
		if n.fingerprint(want) != v.Fingerprint {
			t.Fatal("fingerprint mismatch")
		}
		if v.Kind == 4 {
			continue
		}
		i := Intent{ResourceKind: "bucket", Kind: map[byte]string{1: "create", 2: "update", 3: "delete"}[v.Kind], ManagementKey: v.Key, ExpectedRevision: v.Revision, Spec: &Spec{Name: v.Name, DeletionProtection: v.Protection, Storage: &StorageSpec{Profile: "object", CapacityBytes: v.Capacity, RetentionDays: v.Retention}}}
		if v.Bucket != 0 {
			i.TargetID = "42"
		}
		got, e := nativeIntent(i, v.Generation)
		if e != nil {
			t.Fatal(e)
		}
		if got != want {
			t.Fatal("wire intent mismatch")
		}
		changed := got
		changed[20]++
		if n.fingerprint(changed) != v.Fingerprint {
			t.Fatal("generation entered logical digest")
		}
		changed = got
		changed[24]++
		if n.fingerprint(changed) == v.Fingerprint {
			t.Fatal("revision fence missing from digest")
		}
	}
}
func TestNativeRejectsLossySettings(t *testing.T) {
	i := Intent{ResourceKind: "bucket", Kind: "create", ManagementKey: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Spec: &Spec{Name: "test-data", DeletionProtection: true, Storage: &StorageSpec{Profile: "object", CapacityBytes: DefaultCapacity}}}
	if _, e := nativeIntent(i, 0); e != nil {
		t.Fatal(e)
	}
	for _, key := range []string{"recovery-key", "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", "00000000000000000000000000000000"} {
		bad := i
		bad.ManagementKey = key
		if _, e := nativeIntent(bad, 0); e == nil {
			t.Fatal("accepted noncanonical key")
		}
	}
	for _, name := range []string{"ab", "has spaces", "this-name-exceeds-native-width", "Name"} {
		spec := *i.Spec
		spec.Name = name
		bad := i
		bad.Spec = &spec
		if _, e := nativeIntent(bad, 0); e == nil {
			t.Fatal("accepted lossy name")
		}
	}
	bad := i
	bad.ResourceKind = "database"
	if _, e := nativeIntent(bad, 0); e == nil {
		t.Fatal("accepted unsupported kind")
	}
	for _, id := range []string{"01", "0", "-1", "1e2", "9007199254740992"} {
		if _, e := nativeNumber(id, nativeMax); e == nil {
			t.Fatal("accepted ambiguous/out-of-range ID")
		}
	}
}
func TestNativeRecordRequiresProtection(t *testing.T) {
	raw := []byte(`{"owner":2,"generation":1,"revision":1,"bucket":1,"operation":1,"pending":false,"deleted":false,"name":[],"key":[],"capacity_bytes":1,"retention_days":0,"outcome":"succeeded"}`)
	var r nativeRecord
	if json.Unmarshal(raw, &r) == nil {
		t.Fatal("missing protection accepted as false")
	}
}
