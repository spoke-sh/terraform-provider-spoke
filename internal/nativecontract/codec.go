// Package nativecontract encodes the platform-owned bucket metadata profile.
// It has no transport or signing implementation and is not registered by the
// provider. Shared SDK Serving/proof support is required before live adoption.
package nativecontract

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"regexp"
)

const Profile = "spoke.management.native.v2"
const PlatformRevision = "73b43609a1b493bafaee836bdc349c9e55db68ab"
const TransitRevision = "a183850c000bca7651a89d1e2f429ad9350e8691"
const MaxCounter uint64 = 1<<53 - 1

type Scope struct {
	Organization uint32 `json:"organization"`
	Project      uint32 `json:"project"`
	Environment  uint32 `json:"environment"`
}

func (s Scope) valid() bool { return s.Organization > 0 && s.Project > 0 && s.Environment > 0 }
func (s Scope) put(b []byte) {
	binary.LittleEndian.PutUint32(b, s.Organization)
	binary.LittleEndian.PutUint32(b[4:], s.Project)
	binary.LittleEndian.PutUint32(b[8:], s.Environment)
}

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,14}[a-z0-9]$`)
var invalid = errors.New("unsupported native metadata shape")

type Command struct {
	Kind                                 uint8 // create=1, update=2, delete=3, expire=4
	Protection                           bool
	Scope                                Scope
	Key                                  [16]byte
	ExpectedGeneration, ExpectedRevision uint32
	Bucket                               uint64
	Name                                 string
	CapacityBytes                        uint64
	RetentionDays                        uint32
}

// Encode includes all target and revision preconditions in the canonical intent.
// The caller must freeze/sign and durably journal the shared SDK's prepared work
// before dispatch; these bytes are not a signed Serving command.
func (c Command) Encode() ([64]byte, error) {
	var b [64]byte
	if !c.Scope.valid() || c.Key == [16]byte{} || c.Kind < 1 || c.Kind > 4 || c.Bucket > MaxCounter || c.ExpectedGeneration == ^uint32(0) {
		return b, invalid
	}
	if c.Kind == 1 && (c.ExpectedRevision != 0 || c.Bucket != 0) {
		return b, invalid
	}
	if c.Kind != 1 && (c.ExpectedRevision == 0 || c.Bucket == 0) {
		return b, invalid
	}
	if c.Kind >= 3 {
		if c.Protection || c.Name != "" || c.CapacityBytes != 0 || c.RetentionDays != 0 {
			return b, invalid
		}
	} else if !namePattern.MatchString(c.Name) || c.CapacityBytes == 0 || c.CapacityBytes > MaxCounter || c.RetentionDays > 36500 {
		return b, invalid
	}
	b[0], b[1] = 2, c.Kind
	if c.Protection {
		b[2] = 1
	}
	copy(b[4:20], c.Key[:])
	binary.LittleEndian.PutUint32(b[20:24], c.ExpectedGeneration)
	binary.LittleEndian.PutUint32(b[24:28], c.ExpectedRevision)
	binary.LittleEndian.PutUint64(b[28:36], c.Bucket)
	binary.LittleEndian.PutUint64(b[36:44], c.CapacityBytes)
	binary.LittleEndian.PutUint32(b[44:48], c.RetentionDays)
	copy(b[48:], c.Name)
	return b, nil
}
func (c Command) Fingerprint() ([32]byte, error) {
	b, err := c.Encode()
	if err != nil {
		return [32]byte{}, err
	}
	clear(b[20:24]) // Logical intent is independent of management generation.
	var scope [12]byte
	c.Scope.put(scope[:])
	data := append([]byte("spoke.management.intent.v2\x00"), scope[:]...)
	return sha256.Sum256(append(data, b[:]...)), nil
}

type Read struct {
	Limit                   uint8
	Scope                   Scope
	Kind                    uint8 // capabilities=0, management=1, receipt=2, bucket=3, pending=4, buckets=5
	Key                     [16]byte
	Generation              uint32
	Bucket, After, Revision uint64
	Cursor                  [32]byte
}

// Context returns the application context for a native subject proof. The SDK
// must bind definition, realm, principal, epoch, role and ingress and sign it.
func (r Read) Context() ([32]byte, error) {
	if !r.Scope.valid() || r.Limit < 1 || r.Limit > 8 || r.Kind > 5 || r.Bucket > MaxCounter || r.After > MaxCounter || r.Revision > MaxCounter {
		return [32]byte{}, invalid
	}
	if (r.Kind != 4 && r.Kind != 5 && (r.After != 0 || r.Revision != 0)) || (r.Kind != 2 && r.Generation != 0) || (r.Kind != 3 && r.Bucket != 0) || ((r.Kind == 0 || r.Kind == 3 || r.Kind == 4 || r.Kind == 5) && r.Key != [16]byte{}) {
		return [32]byte{}, invalid
	}
	if r.After == 0 && (r.Revision != 0 || r.Cursor != [32]byte{}) || r.After != 0 && (r.Revision == 0 || r.Cursor == [32]byte{}) {
		return [32]byte{}, invalid
	}
	if (r.Kind == 1 || r.Kind == 2) && r.Key == [16]byte{} || r.Kind == 2 && r.Generation == 0 || r.Kind == 3 && r.Bucket == 0 {
		return [32]byte{}, invalid
	}
	var b [95]byte
	copy(b[:4], "SV1D")
	b[4], b[5] = 2, r.Limit
	r.Scope.put(b[6:18])
	b[18] = r.Kind
	copy(b[19:35], r.Key[:])
	binary.LittleEndian.PutUint32(b[35:39], r.Generation)
	binary.LittleEndian.PutUint64(b[39:47], r.Bucket)
	binary.LittleEndian.PutUint64(b[47:55], r.After)
	binary.LittleEndian.PutUint64(b[55:63], r.Revision)
	copy(b[63:], r.Cursor[:])
	return sha256.Sum256(append([]byte("spoke.management.read.v2\x00"), b[:]...)), nil
}

type Result struct {
	Generation, Revision uint32
	Bucket, Operation    uint64
	Pending, Deleted     bool
	Key                  [16]byte
	Outcome              uint8 // pending=0, succeeded=1, authorization_denied=2, expired=3
}

// DecodeResult accepts a canonical application receipt, including pending replay. Transport
// 202 and terminal Serving dispositions must be resolved separately; finite
// business refusal codes cannot be interpreted as a successful Result.
func DecodeResult(b []byte) (Result, error) {
	if len(b) != 47 || string(b[:4]) != "SV1R" || b[28] > 1 || b[29] > 1 || b[46] > 3 {
		return Result{}, invalid
	}
	r := Result{Generation: binary.LittleEndian.Uint32(b[4:8]), Revision: binary.LittleEndian.Uint32(b[8:12]), Bucket: binary.LittleEndian.Uint64(b[12:20]), Operation: binary.LittleEndian.Uint64(b[20:28]), Pending: b[28] == 1, Deleted: b[29] == 1}
	copy(r.Key[:], b[30:46])
	r.Outcome = b[46]
	if r.Generation == 0 || (r.Revision == 0 && r.Outcome == 1) || (r.Pending != (r.Outcome == 0)) || r.Bucket == 0 || r.Bucket > MaxCounter || r.Operation == 0 || r.Operation > MaxCounter || r.Key == [16]byte{} {
		return Result{}, invalid
	}
	return r, nil
}
