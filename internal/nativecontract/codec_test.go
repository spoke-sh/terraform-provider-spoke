package nativecontract

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
)

func sample() Command {
	c := Command{Kind: 2, Protection: true, Scope: Scope{1, 2, 3}, ExpectedGeneration: 7, ExpectedRevision: 11, Bucket: 19, Name: "assets-2", CapacityBytes: 1024, RetentionDays: 7}
	for i := range c.Key {
		c.Key[i] = byte(i + 1)
	}
	return c
}
func TestPlatformV2Vectors(t *testing.T) {
	b, err := os.ReadFile("../../testdata/contracts/native-v2-vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Scope   Scope
		Vectors []struct {
			Kind                 uint8
			Key                  string
			Generation, Revision uint32
			Bucket               uint64
			Name                 string
			Protection           bool
			Capacity             uint64 `json:"capacity_bytes"`
			Retention            uint32 `json:"retention_days"`
			Payload, Fingerprint string
		}
	}
	if err = json.Unmarshal(b, &fixture); err != nil {
		t.Fatal(err)
	}
	for _, v := range fixture.Vectors {
		c := Command{Kind: v.Kind, Scope: fixture.Scope, ExpectedGeneration: v.Generation, ExpectedRevision: v.Revision, Bucket: v.Bucket, Name: v.Name, Protection: v.Protection, CapacityBytes: v.Capacity, RetentionDays: v.Retention}
		key, _ := hex.DecodeString(v.Key)
		copy(c.Key[:], key)
		payload, err := c.Encode()
		if err != nil || hex.EncodeToString(payload[:]) != v.Payload {
			t.Fatalf("kind %d: command mismatch: %v", v.Kind, err)
		}
		fp, err := c.Fingerprint()
		if err != nil || hex.EncodeToString(fp[:]) != v.Fingerprint {
			t.Fatalf("kind %d: fingerprint mismatch: %v", v.Kind, err)
		}
	}
}
func TestNativeIdentityAndBounds(t *testing.T) {
	c := sample()
	fp, _ := c.Fingerprint()
	c.ExpectedGeneration++
	same, _ := c.Fingerprint()
	if fp != same {
		t.Fatal("generation changed logical intent")
	}
	for _, change := range []func(*Command){func(c *Command) { c.Bucket++ }, func(c *Command) { c.ExpectedRevision++ }, func(c *Command) { c.Scope.Environment++ }, func(c *Command) { c.CapacityBytes++ }, func(c *Command) { c.RetentionDays++ }} {
		c := sample()
		change(&c)
		got, err := c.Fingerprint()
		if err != nil || got == fp {
			t.Fatal("lost identity field")
		}
	}
	for _, change := range []func(*Command){func(c *Command) { c.Name = "Name" }, func(c *Command) { c.Name = "abcdefghijklmnopq" }, func(c *Command) { c.Key = [16]byte{} }, func(c *Command) { c.Bucket = MaxCounter + 1 }, func(c *Command) { c.Scope.Project = 0 }, func(c *Command) { c.Kind = 1 }, func(c *Command) { c.Kind = 3 }, func(c *Command) { c.CapacityBytes = 0 }, func(c *Command) { c.RetentionDays = 36501 }} {
		c := sample()
		change(&c)
		if _, err := c.Encode(); err == nil {
			t.Fatal("accepted unsupported shape")
		}
	}
}
func TestResultOutcomes(t *testing.T) {
	b := make([]byte, 47)
	copy(b, "SV1R")
	binary.LittleEndian.PutUint32(b[4:], 8)
	binary.LittleEndian.PutUint64(b[12:], 19)
	binary.LittleEndian.PutUint64(b[20:], 41)
	key := sample().Key
	copy(b[30:], key[:])
	b[28] = 1
	if r, err := DecodeResult(b); err != nil || !r.Pending {
		t.Fatal("lost pending create", err)
	}
	b[28] = 0
	b[46] = 2
	if r, err := DecodeResult(b); err != nil || r.Outcome != 2 {
		t.Fatal("lost failed create", err)
	}
	b[46] = 1
	if _, err := DecodeResult(b); err == nil {
		t.Fatal("zero revision terminal success")
	}
	binary.LittleEndian.PutUint32(b[8:], 12)
	if _, err := DecodeResult(b); err != nil {
		t.Fatal(err)
	}
	for _, bad := range [][]byte{b[:46], append(append([]byte{}, b...), 0), []byte("revision_conflict"), []byte("202")} {
		if _, err := DecodeResult(bad); err == nil {
			t.Fatal("accepted non-result")
		}
	}
	b[28] = 1
	if _, err := DecodeResult(b); err == nil {
		t.Fatal("pending success")
	}
}
