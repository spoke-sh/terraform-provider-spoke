package platformclient

// Native v2 is an explicitly selected metadata profile. It never forwards
// bearer credentials, invents cloud effects, or translates native uncertainty
// into absence. Transport signing is independent of the logical intent digest.
import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/spoke-sh/transit/sdks/go/serving"
)

const NativeVersion = "spoke.management.native.v2"

// Set to the immutable shared caller by the Nix package. Explicit fixture paths
// may select a matching installed SDK during independent qualification.
var ServingExecutable string

const nativeMax = serving.MaxCounter

type NativeConfig struct {
	Executable       string `json:"client_executable"`
	CAFile           string `json:"-"`
	Realm            uint64 `json:"realm"`
	Principal        uint64 `json:"principal"`
	CredentialEpoch  uint64 `json:"credential_epoch"`
	Ingress          uint64 `json:"ingress"`
	Service          uint64 `json:"service"`
	InitialSequence  uint64 `json:"initial_sequence"`
	SeedFile         string `json:"seed_file"`
	JournalDirectory string `json:"journal_directory"`
}
type nativeClient struct {
	config NativeConfig
	sdk    *serving.Client
	calls  chan struct{}
	scope  nativeScope
	client *Client
}
type nativeScope struct {
	Organization uint32 `json:"organization"`
	Project      uint32 `json:"project"`
	Environment  uint32 `json:"environment"`
}
type nativeRead struct {
	Version    uint8       `json:"version"`
	Limit      uint8       `json:"limit"`
	Scope      nativeScope `json:"scope"`
	Kind       string      `json:"kind"`
	Key        [16]byte    `json:"key"`
	Generation uint32      `json:"generation"`
	Bucket     uint64      `json:"bucket"`
	After      uint64      `json:"after"`
	Revision   uint64      `json:"revision"`
	Cursor     [32]byte    `json:"cursor"`
}
type nativeRecord struct {
	Owner      uint64   `json:"owner"`
	Generation uint32   `json:"generation"`
	Revision   uint32   `json:"revision"`
	Bucket     uint64   `json:"bucket"`
	Operation  uint64   `json:"operation"`
	Pending    bool     `json:"pending"`
	Deleted    bool     `json:"deleted"`
	Protection bool     `json:"protection"`
	Name       [16]byte `json:"name"`
	Key        [16]byte `json:"key"`
	Capacity   uint64   `json:"capacity_bytes"`
	Retention  uint32   `json:"retention_days"`
	Outcome    string   `json:"outcome"`
}
type nativeReceipt struct {
	Owner       uint64       `json:"owner"`
	Operation   uint64       `json:"operation"`
	Correlation uint64       `json:"correlation"`
	Input       uint64       `json:"input"`
	Intent      [64]byte     `json:"intent"`
	Result      nativeRecord `json:"result"`
}
type nativeResponse struct {
	Version      string            `json:"version"`
	Scope        nativeScope       `json:"scope"`
	Revision     uint64            `json:"revision"`
	Record       *nativeRecord     `json:"record"`
	Receipt      *nativeReceipt    `json:"receipt"`
	Pending      [8]*nativeReceipt `json:"pending"`
	Buckets      [8]*nativeRecord  `json:"buckets"`
	Cursor       [32]byte          `json:"cursor"`
	Next         *uint64           `json:"next"`
	Complete     bool              `json:"complete"`
	Capabilities json.RawMessage   `json:"capabilities"`
}

func nativeNumber(s string, max uint64) (uint64, error) {
	n, e := strconv.ParseUint(s, 10, 64)
	if e != nil || n == 0 || n > max || strconv.FormatUint(n, 10) != s {
		return 0, errors.New("native profile requires canonical positive decimal IDs")
	}
	return n, nil
}
func newNative(c *Client, cfg NativeConfig) (*nativeClient, error) {
	for _, n := range []uint64{cfg.Realm, cfg.Principal, cfg.CredentialEpoch, cfg.Ingress, cfg.Service} {
		if n == 0 || n > nativeMax {
			return nil, errors.New("invalid native identity")
		}
	}
	if cfg.Principal <= 1 || cfg.InitialSequence >= nativeMax || cfg.JournalDirectory == "" {
		return nil, errors.New("native profile requires a scoped principal and durable journal directory")
	}
	a, e := nativeNumber(c.scope.OrganizationID, 1<<32-1)
	if e != nil {
		return nil, e
	}
	b, e := nativeNumber(c.scope.ProjectID, 1<<32-1)
	if e != nil {
		return nil, e
	}
	d, e := nativeNumber(c.scope.EnvironmentID, 1<<32-1)
	if e != nil {
		return nil, e
	}
	if cfg.Executable == "" {
		cfg.Executable = ServingExecutable
	}
	var ca *string
	if cfg.CAFile != "" {
		ca = &cfg.CAFile
	}
	sdk, e := serving.Open(serving.Config{Executable: cfg.Executable, Origin: c.endpoint, Realm: cfg.Realm, Principal: cfg.Principal, CredentialEpoch: cfg.CredentialEpoch, Ingress: cfg.Ingress, Service: cfg.Service, SeedFile: cfg.SeedFile, CAFile: ca, TimeoutMS: 10000})
	if e != nil {
		return nil, errors.New("invalid shared native caller configuration")
	}
	return &nativeClient{config: cfg, sdk: sdk, calls: make(chan struct{}, 1), scope: nativeScope{uint32(a), uint32(b), uint32(d)}, client: c}, nil
}

// Terraform may issue concurrent refresh/data-source requests. The shared SDK
// deliberately refuses concurrent native ownership; queue admission here under
// the Terraform deadline rather than treating ordinary refresh as SDK failure.
// This is scheduling only: no retry or replacement of a native operation.
func nativeCall[T any](ctx context.Context, n *nativeClient, call func(*serving.Client) (T, error)) (T, error) {
	select {
	case n.calls <- struct{}{}:
		defer func() { <-n.calls }()
		return call(n.sdk)
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	}
}
func nativeResponseBody(response serving.Response, output any) error {
	body := response.Body
	if response.HTTPStatus != 200 && response.HTTPStatus != 202 {
		var failure struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(body, &failure)
		code := "native_uncertain"
		switch {
		case response.HTTPStatus == 404 && failure.Error == "ManagementNotFound":
			code = "management_not_found"
		case response.HTTPStatus == 404 && failure.Error == "BucketNotFound":
			code = "resource_not_found"
		case response.HTTPStatus == 410 && failure.Error == "ReceiptExpired":
			code = "receipt_expired"
		case response.HTTPStatus == 403:
			code = "access_denied"
		case response.HTTPStatus == 409:
			code = "native_conflict"
		}
		return &APIError{Code: code, Status: response.HTTPStatus}
	}
	if output != nil {
		dec := json.NewDecoder(bytes.NewReader(body))
		dec.DisallowUnknownFields()
		if dec.Decode(output) != nil || dec.Decode(new(any)) != io.EOF {
			return errors.New("invalid native response")
		}
	}
	return nil
}
func (n *nativeClient) read(ctx context.Context, r nativeRead) (nativeResponse, error) {
	r.Version = 2
	r.Scope = n.scope
	if r.Limit == 0 {
		r.Limit = 8
	}
	b := new(bytes.Buffer)
	b.WriteString("SV1D")
	b.WriteByte(r.Version)
	b.WriteByte(r.Limit)
	_ = binary.Write(b, binary.LittleEndian, r.Scope)
	kinds := map[string]byte{"capabilities": 0, "management": 1, "receipt": 2, "bucket": 3, "pending": 4, "buckets": 5}
	kind, ok := kinds[r.Kind]
	if !ok {
		return nativeResponse{}, errors.New("unsupported native read")
	}
	b.WriteByte(kind)
	b.Write(r.Key[:])
	_ = binary.Write(b, binary.LittleEndian, r.Generation)
	_ = binary.Write(b, binary.LittleEndian, r.Bucket)
	_ = binary.Write(b, binary.LittleEndian, r.After)
	_ = binary.Write(b, binary.LittleEndian, r.Revision)
	b.Write(r.Cursor[:])
	digest := sha256.Sum256(append([]byte("spoke.management.read.v2\x00"), b.Bytes()...))
	input, e := json.Marshal(r)
	if e != nil {
		return nativeResponse{}, errors.New("invalid management read")
	}
	response, e := nativeCall(ctx, n, func(sdk *serving.Client) (serving.Response, error) {
		return sdk.ReadApplication(ctx, "/management/v2/read", digest, input)
	})
	var out nativeResponse
	if e == nil {
		e = nativeResponseBody(response, &out)
	}

	if e == nil && (out.Version != NativeVersion || out.Scope != n.scope || out.Revision == 0 || out.Revision > nativeMax || out.Complete != (out.Next == nil)) {
		e = errors.New("native response target or completeness mismatch")
	}
	return out, e
}
func (n *nativeClient) capabilities(ctx context.Context) (Capabilities, error) {
	out, e := n.read(ctx, nativeRead{Kind: "capabilities"})
	if e != nil {
		return Capabilities{}, e
	}
	var cap struct {
		Adapter   string `json:"adapter"`
		Effects   bool   `json:"cloud_effects"`
		Storage   bool   `json:"storage_settings"`
		Listing   bool   `json:"object_listing"`
		Expiry    bool   `json:"expiry"`
		Retention string `json:"retention"`
		Resources string `json:"resource_kinds"`
	}
	if json.Unmarshal(out.Capabilities, &cap) != nil || cap.Adapter != "bucket_metadata" || cap.Effects || !cap.Storage || !cap.Listing || !cap.Expiry || cap.Retention != "retain_or_tombstone" || cap.Resources != "bucket" {
		return Capabilities{}, errors.New("incompatible native capabilities")
	}
	return Capabilities{Scope: n.client.scope, Version: NativeVersion, PrincipalID: strconv.FormatUint(n.config.Principal, 10), Resources: []string{"bucket"}, ReceiptPolicy: "retain-or-tombstone"}, nil
}
func nativeKey(value string) ([16]byte, error) {
	var out [16]byte
	b, e := hex.DecodeString(value)
	if e != nil || len(b) != 16 || hex.EncodeToString(b) != value {
		return out, errors.New("native management_key must be 32 lowercase hex characters")
	}
	copy(out[:], b)
	if out == [16]byte{} {
		return out, errors.New("native management_key must be nonzero")
	}
	return out, nil
}
func nativeIntent(i Intent, generation uint32) ([64]byte, error) {
	var out [64]byte
	if i.ResourceKind != "" && i.ResourceKind != "bucket" {
		return out, errors.New("native profile supports bucket metadata only")
	}
	key, e := nativeKey(i.ManagementKey)
	if e != nil {
		return out, e
	}
	kind := map[string]byte{"create": 1, "update": 2, "delete": 3}[i.Kind]
	if kind == 0 {
		return out, errors.New("unsupported native mutation")
	}
	out[0] = 2
	out[1] = kind
	copy(out[4:20], key[:])
	binary.LittleEndian.PutUint32(out[20:24], generation)
	if i.Kind != "create" {
		id, e := nativeNumber(i.TargetID, nativeMax)
		if e != nil {
			return out, e
		}
		if i.ExpectedRevision < 1 || i.ExpectedRevision > 1<<32-1 {
			return out, errors.New("invalid native revision")
		}
		binary.LittleEndian.PutUint64(out[28:36], id)
		binary.LittleEndian.PutUint32(out[24:28], uint32(i.ExpectedRevision))
	} else if i.TargetID != "" || i.ExpectedRevision != 0 {
		return out, errors.New("create requires no target or revision")
	}
	if kind != 3 {
		if i.Spec == nil || i.Spec.Storage == nil || !i.Spec.Valid("bucket") || !regexp.MustCompile(`^[a-z][a-z0-9-]{1,14}[a-z0-9]$`).MatchString(i.Spec.Name) {
			return out, errors.New("native bucket requires valid capacity/retention and a 3–16 byte lowercase name")
		}
		if i.Spec.DeletionProtection {
			out[2] = 1
		}
		binary.LittleEndian.PutUint64(out[36:44], uint64(i.Spec.Storage.CapacityBytes))
		binary.LittleEndian.PutUint32(out[44:48], uint32(i.Spec.Storage.RetentionDays))
		copy(out[48:], i.Spec.Name)
	}
	return out, nil
}
func (n *nativeClient) fingerprint(intent [64]byte) string {
	copy(intent[20:24], make([]byte, 4))
	b := new(bytes.Buffer)
	b.WriteString("spoke.management.intent.v2\x00")
	_ = binary.Write(b, binary.LittleEndian, n.scope)
	b.Write(intent[:])
	h := sha256.Sum256(b.Bytes())
	return hex.EncodeToString(h[:])
}
func (n *nativeClient) object(r nativeRecord) (Object, error) {
	if r.Owner != n.config.Principal || r.Bucket == 0 || r.Bucket > nativeMax || r.Generation == 0 || r.Operation == 0 || r.Key == [16]byte{} {
		return Object{}, errors.New("invalid native record identity")
	}
	if r.Pending {
		return Object{}, &APIError{Code: "operation_pending", Status: 409}
	}
	if r.Deleted {
		return Object{}, &APIError{Code: "resource_not_found", Status: 404}
	}
	o := Object{Bucket: Bucket{Scope: n.client.scope, Spec: Spec{Name: strings.TrimRight(string(r.Name[:]), "\x00"), DeletionProtection: r.Protection, Storage: &StorageSpec{Profile: "object", CapacityBytes: int64(r.Capacity), RetentionDays: int64(r.Retention)}}, ID: strconv.FormatUint(r.Bucket, 10), ManagementKey: hex.EncodeToString(r.Key[:]), Revision: int64(r.Revision), OwnerID: strconv.FormatUint(r.Owner, 10)}, ResourceKind: "bucket", Status: &ServiceStatus{}}
	o.Status.Readiness = "pending"
	o.Status.Reason = "metadata_only"
	o.Status.DesiredRevision = int64(r.Revision)
	o.Status.OperationID = strconv.FormatUint(r.Operation, 10)
	if !o.Spec.Valid("bucket") || r.Revision == 0 || r.Capacity > nativeMax {
		return Object{}, errors.New("invalid native bucket attributes")
	}
	return o, nil
}
func (n *nativeClient) get(ctx context.Context, kind, id string) (Object, error) {
	if kind != "bucket" {
		return Object{}, errors.New("native profile supports bucket metadata only")
	}
	bucket, e := nativeNumber(id, nativeMax)
	if e != nil {
		return Object{}, e
	}
	out, e := n.read(ctx, nativeRead{Kind: "bucket", Bucket: bucket})
	if e != nil {
		return Object{}, e
	}
	if out.Record == nil || out.Record.Bucket != bucket {
		return Object{}, errors.New("incomplete native bucket response")
	}
	return n.object(*out.Record)
}
func (n *nativeClient) list(ctx context.Context, kind string) ([]Object, error) {
	if kind != "bucket" {
		return nil, errors.New("native profile supports bucket metadata only")
	}
	r := nativeRead{Kind: "buckets"}
	result := []Object{}
	ids := map[string]bool{}
	for page := 0; page < n.client.maxPages; page++ {
		out, e := n.read(ctx, r)
		if e != nil {
			return nil, e
		}
		if r.Revision != 0 && r.Revision != out.Revision {
			return nil, errors.New("native snapshot changed")
		}
		ended := false
		count := 0
		for _, record := range out.Buckets {
			if record == nil {
				ended = true
				continue
			}
			if ended {
				return nil, errors.New("noncanonical native page")
			}
			o, e := n.object(*record)
			if e != nil {
				return nil, e
			}
			if ids[o.ID] || record.Bucket <= r.After {
				return nil, errors.New("duplicate native bucket")
			}
			ids[o.ID] = true
			result = append(result, o)
			count++
			if len(result) > n.client.maxItems {
				return nil, errors.New("native item limit exceeded")
			}
		}
		if out.Complete {
			return result, nil
		}
		if count == 0 || *out.Next <= r.After || out.Cursor == [32]byte{} {
			return nil, errors.New("invalid native continuation")
		}
		r.After = *out.Next
		r.Revision = out.Revision
		r.Cursor = out.Cursor
	}
	return nil, errors.New("native page limit exceeded")
}
func (n *nativeClient) operation(receipt nativeReceipt) Operation {
	r := receipt.Result
	status := "succeeded"
	code := ""
	if r.Pending {
		status = "accepted"
	} else if r.Outcome != "succeeded" {
		status = "failed"
		code = "authorization_denied"
	}
	return Operation{Scope: n.client.scope, ID: strconv.FormatUint(receipt.Operation, 10), ManagementKey: hex.EncodeToString(r.Key[:]), Fingerprint: n.fingerprint(receipt.Intent), Kind: map[byte]string{1: "create", 2: "update", 3: "delete"}[receipt.Intent[1]], TargetID: strconv.FormatUint(r.Bucket, 10), Generation: int64(r.Generation), Status: status, ErrorCode: code, ResourceKind: "bucket", OwnerID: strconv.FormatUint(r.Owner, 10)}
}
func (n *nativeClient) outcome(ctx context.Context, key [16]byte, gen uint32, fingerprint string) (Operation, error) {
	for {
		out, e := n.read(ctx, nativeRead{Kind: "receipt", Key: key, Generation: gen})
		if e != nil {
			return Operation{}, e
		}
		if out.Receipt == nil || out.Receipt.Owner != n.config.Principal || out.Receipt.Result.Key != key || out.Receipt.Result.Generation != gen || n.fingerprint(out.Receipt.Intent) != fingerprint {
			return Operation{}, errors.New("native receipt identity mismatch")
		}
		op := n.operation(*out.Receipt)
		if op.Status == "failed" {
			return op, &OperationError{Operation: op, Code: op.ErrorCode}
		}
		if op.Status == "succeeded" {
			return op, nil
		}
		if e = n.client.wait(ctx, n.client.poll); e != nil {
			return op, &PendingError{Operation: op, Cause: e}
		}
	}
}
func (n *nativeClient) mutate(ctx context.Context, i Intent) (Operation, error) {
	payload, e := nativeIntent(i, 0)
	if e != nil {
		return Operation{}, e
	}
	if _, e = n.capabilities(ctx); e != nil {
		return Operation{}, e
	}
	key, _ := nativeKey(i.ManagementKey)
	fingerprint := n.fingerprint(payload)
	generation := uint32(0)
	current, e := n.read(ctx, nativeRead{Kind: "management", Key: key})
	if e != nil && !IsCode(e, "management_not_found") {
		return Operation{}, e
	}
	if e == nil {
		if current.Record == nil || current.Receipt == nil || current.Record.Owner != n.config.Principal || current.Record.Key != key || current.Receipt.Result.Generation != current.Record.Generation {
			return Operation{}, errors.New("incomplete native management identity")
		}
		generation = current.Record.Generation
		if n.fingerprint(current.Receipt.Intent) == fingerprint {
			return n.outcome(ctx, key, generation, fingerprint)
		}
		if current.Record.Pending {
			return Operation{}, &APIError{Code: "operation_pending", Status: 409}
		}
		if i.Kind == "create" && !current.Record.Deleted {
			return Operation{}, &APIError{Code: "ownership_conflict", Status: 409}
		}
	}
	if generation == 1<<32-1 {
		return Operation{}, errors.New("native generation exhausted")
	}
	binary.LittleEndian.PutUint32(payload[20:24], generation)
	operation, e := n.dispatch(ctx, payload)
	if e != nil {
		return Operation{}, e
	}
	// Observe the native Serving operation as well: pre-admission refusals have no
	// application receipt. Its absence alone never licenses a replacement send.
	for {
		out, e := n.read(ctx, nativeRead{Kind: "receipt", Key: key, Generation: generation + 1})
		if e == nil && out.Receipt != nil {
			return n.outcome(ctx, key, generation+1, fingerprint)
		}
		if e != nil && (IsCode(e, "access_denied") || IsCode(e, "receipt_expired")) {
			return Operation{}, e
		}
		stage, e := n.serving(ctx, "status", operation)
		if e == nil && stage.Stage == "completed" {
			raw, e := hex.DecodeString(stage.Operation.Result)
			if e != nil {
				return Operation{}, errors.New("invalid native operation result")
			}
			if !bytes.HasPrefix(raw, []byte("SV1R")) {
				code := string(raw)
				switch code {
				case "generation_conflict", "ownership_conflict", "revision_conflict", "operation_pending", "name_conflict", "quota_exceeded", "deletion_protected", "capacity_shrink", "receipt_expired", "authorization_denied", "unsupported_profile", "incompatible_api", "invalid_request", "generation_exhausted", "revision_exhausted", "incarnation_exhausted":
					return Operation{}, &APIError{Code: code, Status: 409}
				}
				return Operation{}, errors.New("unknown native refusal; preserve original intent")
			}
		}
		if e = n.client.wait(ctx, n.client.poll); e != nil {
			return Operation{}, fmt.Errorf("native operation %d remains uncertain; retry original intent", operation)
		}
	}
}

func (r *nativeRecord) UnmarshalJSON(raw []byte) error {
	if e := nativeLengths(raw, map[string]int{"name": 16, "key": 16}); e != nil {
		return e
	}

	if e := requireMembers(raw, reflect.TypeOf(*r)); e != nil {
		return e
	}
	type plain nativeRecord
	var p plain
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if e := d.Decode(&p); e != nil {
		return e
	}
	if p.Outcome != "pending" && p.Outcome != "succeeded" && p.Outcome != "authorization_denied" && p.Outcome != "expired" {
		return errors.New("invalid native outcome")
	}
	if p.Pending != (p.Outcome == "pending") {
		return errors.New("inconsistent native outcome")
	}
	*r = nativeRecord(p)
	return nil
}

func nativeLengths(raw []byte, fields map[string]int) error {
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil {
		return errors.New("invalid native envelope")
	}
	for field, count := range fields {
		var a []json.RawMessage
		if json.Unmarshal(m[field], &a) != nil || len(a) != count {
			return errors.New("invalid native array width")
		}
	}
	return nil
}
func (r *nativeReceipt) UnmarshalJSON(raw []byte) error {
	if e := requireMembers(raw, reflect.TypeOf(*r)); e != nil {
		return e
	}
	if e := nativeLengths(raw, map[string]int{"intent": 64}); e != nil {
		return e
	}
	type plain nativeReceipt
	var value plain
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if e := d.Decode(&value); e != nil {
		return e
	}
	if value.Owner != value.Result.Owner || value.Operation != value.Result.Operation || value.Correlation == 0 || value.Input == 0 {
		return errors.New("native receipt owner/operation mismatch")
	}
	*r = nativeReceipt(value)
	return nil
}
func (r *nativeResponse) UnmarshalJSON(raw []byte) error {
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		return errors.New("invalid native envelope")
	}
	for _, name := range []string{"version", "scope", "revision", "record", "receipt", "pending", "buckets", "cursor", "next", "complete", "capabilities"} {
		if _, ok := fields[name]; !ok {
			return errors.New("incomplete native envelope")
		}
	}
	for _, name := range []string{"version", "scope", "revision", "complete"} {
		if bytes.Equal(fields[name], []byte("null")) {
			return errors.New("null native envelope field")
		}
	}
	if e := nativeLengths(raw, map[string]int{"pending": 8, "buckets": 8, "cursor": 32}); e != nil {
		return e
	}
	type plain nativeResponse
	var value plain
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if e := d.Decode(&value); e != nil {
		return e
	}
	*r = nativeResponse(value)
	return nil
}

// ObjectPath is an identity label, not a fabricated REST route for native reads.
func (c *Client) ObjectPath(o Object) string {
	if c.native != nil {
		return fmt.Sprintf("%s/%d/%d/%s/%s/%s/bucket/%s", NativeVersion, c.native.config.Realm, c.native.config.Ingress, o.OrganizationID, o.ProjectID, o.EnvironmentID, o.ID)
	}
	return o.Scope.Path() + "/" + Collection(o.ResourceKind) + "/" + o.ID
}

func (c *Client) CollectionPath(kind string) string {
	return strings.TrimSuffix(c.ObjectPath(Object{Bucket: Bucket{Scope: c.scope}, ResourceKind: kind}), "/")
}
