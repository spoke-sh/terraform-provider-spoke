package nativecontract

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func observation(t *testing.T, body Observation) []byte {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func record(id uint64) *Record {
	c := sample()
	r := &Record{Owner: 9, Generation: 1, Revision: 1, Bucket: id, Operation: 41, Key: c.Key, CapacityBytes: 1024, RetentionDays: 7, Outcome: "succeeded"}
	copy(r.Name[:], "assets")
	return r
}
func TestBoundedNativeDiscovery(t *testing.T) {
	read := Read{Limit: 1, Scope: Scope{1, 2, 3}, Kind: 5}
	page := Observation{Version: Profile, Scope: read.Scope, Revision: 7, Complete: false}
	page.Buckets[0] = record(1)
	next := uint64(1)
	page.Next = &next
	page.Cursor[0] = 3
	calls := 0
	fetch := func(ctx context.Context, r Read) ([]byte, error) {
		calls++
		if calls == 1 {
			return observation(t, page), nil
		}
		if r.After != 1 || r.Revision != 7 || r.Cursor != page.Cursor {
			t.Fatal("lost continuation binding")
		}
		last := page
		last.Complete = true
		last.Next = nil
		last.Cursor = [32]byte{}
		last.Buckets[0] = record(2)
		return observation(t, last), nil
	}
	got, err := Collect(context.Background(), read, 9, 2, 2, fetch)
	if err != nil || len(got) != 2 {
		t.Fatal(got, err)
	}
	for _, limit := range []struct{ pages, items int }{{1, 2}, {2, 1}} {
		calls = 0
		if got, err := Collect(context.Background(), read, 9, limit.pages, limit.items, fetch); err == nil || got != nil {
			t.Fatal("partial traversal succeeded")
		}
	}
	for _, change := range []func(*Observation){func(p *Observation) { p.Scope.Environment++ }, func(p *Observation) { p.Buckets[0].Owner = 8 }, func(p *Observation) { p.Complete = true }, func(p *Observation) { p.Next = nil }, func(p *Observation) { p.Buckets[1] = record(2) }, func(p *Observation) { p.Version = "spoke.management.native.v1" }, func(p *Observation) { p.Revision = 0 }, func(p *Observation) { p.Cursor = [32]byte{} }} {
		p := page
		p.Buckets[0] = record(1)
		change(&p)
		if _, err := DecodeObservation(observation(t, p), read, 9); err == nil {
			t.Fatal("accepted invalid observation")
		}
	}
	raw := string(observation(t, page))
	for _, bad := range []string{strings.Replace(raw, `"complete":false`, `"complete":false,"complete":false`, 1), strings.Replace(raw, `"complete":false`, `"complete":null`, 1), strings.Replace(raw, `"retention_days":7,`, "", 1), raw + "{}", strings.Repeat("x", 65537)} {
		if _, err := DecodeObservation([]byte(bad), read, 9); err == nil {
			t.Fatal("accepted ambiguous/incomplete response")
		}
	}
}
func TestProofContextBindsEverySelector(t *testing.T) {
	c := sample()
	read := Read{Limit: 8, Scope: c.Scope, Kind: 2, Key: c.Key, Generation: 8}
	original, err := read.Context()
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*Read){func(r *Read) { r.Limit-- }, func(r *Read) { r.Scope.Environment++ }, func(r *Read) { r.Key[0]++ }, func(r *Read) { r.Generation++ }} {
		r := read
		change(&r)
		now, err := r.Context()
		if err != nil || now == original {
			t.Fatal("unbound selector")
		}
	}
	for _, r := range []Read{{Limit: 9, Scope: c.Scope}, {Limit: 1, Scope: c.Scope, Kind: 3, Key: c.Key}, {Limit: 1, Scope: c.Scope, Kind: 5, Revision: 1}, {Limit: 1, Scope: c.Scope, Kind: 4, Generation: 1}, {Limit: 1, Scope: c.Scope, Kind: 4, After: 1, Revision: 1}} {
		if _, err := r.Context(); err == nil {
			t.Fatal("accepted invalid selectors")
		}
	}
}
