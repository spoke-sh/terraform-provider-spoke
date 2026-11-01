package nativecontract

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// schemaJSON is copied verbatim from PlatformRevision, not a provider-owned API.
//
//go:embed schema.json
var schemaJSON []byte

var observationSchema = sync.OnceValues(func() (*jsonschema.Schema, error) {
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaJSON))
	if err != nil {
		return nil, err
	}
	c := jsonschema.NewCompiler()
	const uri = "https://www.spoke.sh/contracts/management-native-v2.schema.json"
	if err := c.AddResource(uri, doc); err != nil {
		return nil, err
	}
	return c.Compile(uri + "#/$defs/response")
})

type Record struct {
	Owner         uint64   `json:"owner"`
	Generation    uint32   `json:"generation"`
	Revision      uint32   `json:"revision"`
	Bucket        uint64   `json:"bucket"`
	Operation     uint64   `json:"operation"`
	Pending       bool     `json:"pending"`
	Deleted       bool     `json:"deleted"`
	Protection    bool     `json:"protection"`
	Name          [16]byte `json:"name"`
	Key           [16]byte `json:"key"`
	CapacityBytes uint64   `json:"capacity_bytes"`
	RetentionDays uint32   `json:"retention_days"`
	Outcome       string   `json:"outcome"`
}
type Receipt struct {
	Owner       uint64   `json:"owner"`
	Operation   uint64   `json:"operation"`
	Correlation uint64   `json:"correlation"`
	Input       uint64   `json:"input"`
	Intent      [64]byte `json:"intent"`
	Result      Record   `json:"result"`
}
type Observation struct {
	Version      string          `json:"version"`
	Scope        Scope           `json:"scope"`
	Revision     uint64          `json:"revision"`
	Record       *Record         `json:"record"`
	Receipt      *Receipt        `json:"receipt"`
	Pending      [8]*Receipt     `json:"pending"`
	Buckets      [8]*Record      `json:"buckets"`
	Cursor       [32]byte        `json:"cursor"`
	Next         *uint64         `json:"next"`
	Complete     bool            `json:"complete"`
	Capabilities json.RawMessage `json:"capabilities"`
}

// uniqueJSON rejects ambiguous object members before schema validation.
func uniqueJSON(d *json.Decoder) error {
	t, err := d.Token()
	if err != nil {
		return err
	}
	delim, ok := t.(json.Delim)
	if !ok {
		return nil
	}
	if delim != '{' && delim != '[' {
		return invalid
	}
	seen := map[string]bool{}
	for d.More() {
		if delim == '{' {
			key, err := d.Token()
			if err != nil {
				return err
			}
			k, ok := key.(string)
			if !ok || seen[k] {
				return invalid
			}
			seen[k] = true
		}
		if err := uniqueJSON(d); err != nil {
			return err
		}
	}
	_, err = d.Token()
	return err
}
func validRecord(r Record, owner uint64) bool {
	if r.Owner != owner || r.Key == [16]byte{} || r.Pending != (r.Outcome == "pending") || r.Revision == 0 && r.Outcome == "succeeded" {
		return false
	}
	name := bytes.TrimRight(r.Name[:], "\x00")
	return len(name) == 0 && (r.Deleted || r.Revision == 0) || namePattern.Match(name)
}
func validReceipt(r Receipt, owner uint64, scope Scope) bool {
	if r.Owner != owner || r.Operation != r.Result.Operation || !validRecord(r.Result, owner) {
		return false
	}
	b := r.Intent
	c := Command{Scope: scope, Kind: b[1], Protection: b[2] == 1, ExpectedGeneration: binary.LittleEndian.Uint32(b[20:24]), ExpectedRevision: binary.LittleEndian.Uint32(b[24:28]), Bucket: binary.LittleEndian.Uint64(b[28:36]), CapacityBytes: binary.LittleEndian.Uint64(b[36:44]), RetentionDays: binary.LittleEndian.Uint32(b[44:48]), Name: string(bytes.TrimRight(b[48:], "\x00"))}
	copy(c.Key[:], b[4:20])
	encoded, err := c.Encode()
	if err != nil || encoded != b || c.Kind == 4 || c.Key != r.Result.Key || c.ExpectedGeneration+1 != r.Result.Generation || c.Kind != 1 && c.Bucket != r.Result.Bucket {
		return false
	}
	revision := c.ExpectedRevision
	if r.Result.Outcome == "succeeded" {
		revision++
	}
	return revision == r.Result.Revision
}

// DecodeObservation binds a strict platform envelope to the exact signed read and
// caller. It neither authenticates transport nor creates a subject proof.
func DecodeObservation(body []byte, read Read, principal uint64) (Observation, error) {
	var out Observation
	if _, err := read.Context(); err != nil || principal == 0 || principal > MaxCounter || len(body) > 65536 {
		return out, invalid
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	if uniqueJSON(dec) != nil {
		return out, invalid
	}
	if _, err := dec.Token(); err != io.EOF {
		return out, invalid
	}
	schema, err := observationSchema()
	if err != nil {
		return out, errors.New("invalid embedded native schema")
	}
	value, err := jsonschema.UnmarshalJSON(bytes.NewReader(body))
	if err != nil || schema.Validate(value) != nil || json.Unmarshal(body, &out) != nil {
		return Observation{}, invalid
	}
	if out.Scope != read.Scope || (read.Revision != 0 && out.Revision != read.Revision) || out.Complete != (out.Next == nil) {
		return Observation{}, invalid
	}
	if out.Next == nil && out.Cursor != [32]byte{} || out.Next != nil && (*out.Next <= read.After || out.Cursor == [32]byte{}) {
		return Observation{}, invalid
	}
	if out.Record != nil && !validRecord(*out.Record, principal) || out.Receipt != nil && !validReceipt(*out.Receipt, principal, read.Scope) {
		return Observation{}, invalid
	}
	for _, r := range out.Pending {
		if r != nil && (!validReceipt(*r, principal, read.Scope) || !r.Result.Pending) {
			return Observation{}, invalid
		}
	}
	for _, r := range out.Buckets {
		if r != nil && (!validRecord(*r, principal) || r.Deleted && !r.Pending) {
			return Observation{}, invalid
		}
	}
	if read.Kind < 4 && (!out.Complete || out.Next != nil) {
		return Observation{}, invalid
	}
	hasCaps := len(out.Capabilities) > 0 && string(out.Capabilities) != "null"
	if hasCaps != (read.Kind == 0) {
		return Observation{}, invalid
	}
	if (out.Record != nil) != (read.Kind == 1 || read.Kind == 3) || (out.Receipt != nil) != (read.Kind >= 1 && read.Kind <= 3) {
		return Observation{}, invalid
	}
	if out.Record != nil {
		expected := out.Receipt.Result
		// Expiry changes the management tombstone, not the historical receipt.
		// Bucket inspection remains available under the owner's published v2 contract.
		if read.Kind == 3 && out.Record.Outcome == "expired" {
			expected.Outcome = "expired"
		}
		if read.Kind == 1 && out.Record.Key != read.Key || read.Kind == 3 && out.Record.Bucket != read.Bucket || *out.Record != expected {
			return Observation{}, invalid
		}
	}
	if read.Kind == 2 && (out.Receipt.Result.Key != read.Key || out.Receipt.Result.Generation != read.Generation) {
		return Observation{}, invalid
	}
	count, last := 0, read.After
	ended := false
	for i := 0; i < 8; i++ {
		if read.Kind != 4 && out.Pending[i] != nil || read.Kind != 5 && out.Buckets[i] != nil {
			return Observation{}, invalid
		}
		var id uint64
		if read.Kind == 4 && out.Pending[i] != nil {
			id = out.Pending[i].Input
		}
		if read.Kind == 5 && out.Buckets[i] != nil {
			id = out.Buckets[i].Bucket
		}
		if id == 0 {
			ended = true
			continue
		}
		if ended || id <= last {
			return Observation{}, invalid
		}
		last = id
		count++
	}
	if count > int(read.Limit) || out.Next != nil && (count == 0 || *out.Next != last) {
		return Observation{}, invalid
	}
	return out, nil
}

// Collect observes one complete cut. A stale cut fails so the caller can perform
// an explicitly bounded restart; no partial result can establish absence.
func Collect(ctx context.Context, read Read, principal uint64, maxPages, maxItems int, fetch func(context.Context, Read) ([]byte, error)) ([]Record, error) {
	if read.Kind != 4 && read.Kind != 5 || read.After != 0 || read.Revision != 0 || read.Cursor != [32]byte{} || maxPages < 1 || maxPages > 1024 || maxItems < 1 || maxItems > 8192 || fetch == nil {
		return nil, invalid
	}
	if ctx == nil || principal == 0 || principal > MaxCounter {
		return nil, invalid
	}
	if _, err := read.Context(); err != nil {
		return nil, err
	}
	var records []Record
	seen := map[[32]byte]bool{}
	var revision uint64
	for n := 0; n < maxPages; n++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		body, err := fetch(ctx, read)
		if err != nil {
			return nil, err
		}
		out, err := DecodeObservation(body, read, principal)
		if err != nil {
			return nil, err
		}
		if revision != 0 && out.Revision != revision {
			return nil, invalid
		}
		revision = out.Revision
		for i := 0; i < 8; i++ {
			if out.Buckets[i] != nil {
				records = append(records, *out.Buckets[i])
			}
			if out.Pending[i] != nil {
				records = append(records, out.Pending[i].Result)
			}
		}
		if len(records) > maxItems {
			return nil, errors.New("native collection item limit exceeded")
		}
		if out.Complete {
			return records, nil
		}
		if seen[out.Cursor] {
			return nil, invalid
		}
		seen[out.Cursor] = true
		read.After, read.Revision, read.Cursor = *out.Next, out.Revision, out.Cursor
	}
	return nil, errors.New("native collection page limit exceeded")
}
