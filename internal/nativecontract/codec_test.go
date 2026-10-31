package nativecontract

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
)

func sample() Command {
	c := Command{Kind: 2, Protection: true, Scope: Scope{1, 2, 3}, ExpectedGeneration: 7, ExpectedRevision: 11, Bucket: 19, Name: "assets-2"}
	for i := range c.Key {
		c.Key[i] = byte(i + 1)
	}
	return c
}
func TestIndependentPlatformVectors(t *testing.T) {
	data, err := os.ReadFile("../../testdata/contracts/native-v1-vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var v map[string]string
	if err = json.Unmarshal(data, &v); err != nil {
		t.Fatal(err)
	}
	c := sample()
	b, err := c.Encode()
	if err != nil || hex.EncodeToString(b[:]) != v["command"] {
		t.Fatal("canonical command mismatch", err)
	}
	hash, err := c.Fingerprint()
	if err != nil || hex.EncodeToString(hash[:]) != v["fingerprint"] {
		t.Fatal("fingerprint mismatch", err)
	}
	context, err := (Read{Limit: 8, Scope: c.Scope, Kind: 2, Key: c.Key, Generation: 8}).Context()
	if err != nil || hex.EncodeToString(context[:]) != v["read_context"] {
		t.Fatal("proof context mismatch", err)
	}
	result, _ := hex.DecodeString(v["result"])
	r, err := DecodeResult(result)
	if err != nil || r.Bucket != 19 || r.Operation != 41 || r.Generation != 8 || r.Revision != 12 || r.Pending || r.Deleted || r.Key != c.Key {
		t.Fatal("business result mismatch", err)
	}
	// The platform can replay a pending create receipt before its first revision.
	// This is still pending work, never completed CRUD or authoritative absence.
	pending := append([]byte{}, result...)
	clear(pending[8:12])
	pending[28] = 1
	if r, err := DecodeResult(pending); err != nil || !r.Pending || r.Revision != 0 || r.Deleted {
		t.Fatal("lost pending create receipt", err)
	}
	pending[28] = 0
	if _, err := DecodeResult(pending); err == nil {
		t.Fatal("zero revision became terminal success")
	}
	for _, bad := range [][]byte{result[:45], append(append([]byte{}, result...), 0), []byte("revision_conflict"), []byte("202")} {
		if _, err := DecodeResult(bad); err == nil {
			t.Fatal("accepted malformed/non-result")
		}
	}
	result[28] = 2
	if _, err := DecodeResult(result); err == nil {
		t.Fatal("accepted noncanonical boolean")
	}
}
func TestNativePreconditionsAndShape(t *testing.T) {
	original := sample()
	old, _ := original.Fingerprint()
	for _, change := range []func(*Command){func(c *Command) { c.Bucket++ }, func(c *Command) { c.ExpectedRevision++ }, func(c *Command) { c.ExpectedGeneration++ }, func(c *Command) { c.Scope.Environment++ }} {
		c := original
		change(&c)
		now, err := c.Fingerprint()
		if err != nil || now == old {
			t.Fatal("precondition absent from fingerprint")
		}
	}
	for _, change := range []func(*Command){func(c *Command) { c.Name = "Name" }, func(c *Command) { c.Name = "abcdefghijklmnopq" }, func(c *Command) { c.Key = [16]byte{} }, func(c *Command) { c.Bucket = MaxCounter + 1 }, func(c *Command) { c.Scope.Project = 0 }, func(c *Command) { c.Kind = 1 }, func(c *Command) { c.Kind = 3 }} {
		c := original
		change(&c)
		if _, err := c.Encode(); err == nil {
			t.Fatal("accepted unsupported native shape")
		}
	}
	for _, r := range []Read{{Limit: 9, Scope: original.Scope}, {Limit: 1, Scope: original.Scope, Kind: 3, Key: original.Key}, {Limit: 1, Scope: original.Scope, Kind: 1, Revision: 1}, {Limit: 1, Scope: original.Scope, Kind: 4, Generation: 1}} {
		if _, err := r.Context(); err == nil {
			t.Fatal("accepted invalid read selectors")
		}
	}
}
