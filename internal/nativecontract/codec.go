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

const Profile = "spoke.management.native.v1"
const PlatformRevision = "230c2c4cee12c08aaeaaf65e3c252a6d50450a55"
const TransitRevision = "0e2fba44727ca858bd1e8e4a775b62fbcf1a665d"
const MaxCounter uint64 = 1<<53 - 1

type Scope struct{ Organization, Project, Environment uint32 }

func (s Scope) valid() bool { return s.Organization > 0 && s.Project > 0 && s.Environment > 0 }
func (s Scope) put(b []byte) {
	binary.LittleEndian.PutUint32(b, s.Organization)
	binary.LittleEndian.PutUint32(b[4:], s.Project)
	binary.LittleEndian.PutUint32(b[8:], s.Environment)
}

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,14}[a-z0-9]$`)
var invalid = errors.New("unsupported native metadata shape")

type Command struct {
	Kind                                 uint8 // create=1, update=2, delete=3
	Protection                           bool
	Scope                                Scope
	Key                                  [16]byte
	ExpectedGeneration, ExpectedRevision uint32
	Bucket                               uint64
	Name                                 string
}

// Encode includes all target and revision preconditions in the canonical intent.
// The caller must freeze/sign and durably journal the shared SDK's prepared work
// before dispatch; these bytes are not a signed Serving command.
func (c Command) Encode() ([64]byte, error) {
	var b [64]byte
	if !c.Scope.valid() || c.Key == [16]byte{} || c.Kind < 1 || c.Kind > 3 || c.Bucket > MaxCounter || c.ExpectedGeneration == ^uint32(0) {
		return b, invalid
	}
	if c.Kind == 1 && (c.ExpectedRevision != 0 || c.Bucket != 0) {
		return b, invalid
	}
	if c.Kind != 1 && (c.ExpectedRevision == 0 || c.Bucket == 0) {
		return b, invalid
	}
	if c.Kind == 3 {
		if c.Protection || c.Name != "" {
			return b, invalid
		}
	} else if !namePattern.MatchString(c.Name) {
		return b, invalid
	}
	b[0], b[1] = 1, c.Kind
	if c.Protection {
		b[2] = 1
	}
	c.Scope.put(b[4:16])
	copy(b[16:32], c.Key[:])
	binary.LittleEndian.PutUint32(b[32:36], c.ExpectedGeneration)
	binary.LittleEndian.PutUint32(b[36:40], c.ExpectedRevision)
	binary.LittleEndian.PutUint64(b[40:48], c.Bucket)
	copy(b[48:], c.Name)
	return b, nil
}
func (c Command) Fingerprint() ([32]byte, error) {
	b, err := c.Encode()
	if err != nil {
		return [32]byte{}, err
	}
	return sha256.Sum256(b[:]), nil
}

type Read struct {
	Limit                   uint8
	Scope                   Scope
	Kind                    uint8 // capabilities=0, management=1, receipt=2, bucket=3, pending=4
	Key                     [16]byte
	Generation              uint32
	Bucket, After, Revision uint64
}

// Context returns the application context for a native subject proof. The SDK
// must bind definition, realm, principal, epoch, role and ingress and sign it.
func (r Read) Context() ([32]byte, error) {
	if !r.Scope.valid() || r.Limit < 1 || r.Limit > 8 || r.Kind > 4 || r.Bucket > MaxCounter || r.After > MaxCounter || r.Revision > MaxCounter {
		return [32]byte{}, invalid
	}
	if (r.Kind != 4 && (r.After != 0 || r.Revision != 0)) || (r.Kind != 2 && r.Generation != 0) || (r.Kind != 3 && r.Bucket != 0) || ((r.Kind == 0 || r.Kind == 3 || r.Kind == 4) && r.Key != [16]byte{}) {
		return [32]byte{}, invalid
	}
	var b [63]byte
	copy(b[:4], "SV1D")
	b[4], b[5] = 1, r.Limit
	r.Scope.put(b[6:18])
	b[18] = r.Kind
	copy(b[19:35], r.Key[:])
	binary.LittleEndian.PutUint32(b[35:39], r.Generation)
	binary.LittleEndian.PutUint64(b[39:47], r.Bucket)
	binary.LittleEndian.PutUint64(b[47:55], r.After)
	binary.LittleEndian.PutUint64(b[55:63], r.Revision)
	return sha256.Sum256(append([]byte("spoke.management.read.v1\x00"), b[:]...)), nil
}

type Result struct {
	Generation, Revision uint32
	Bucket, Operation    uint64
	Pending, Deleted     bool
	Key                  [16]byte
}

// DecodeResult accepts only a canonical successful application result. Transport
// 202 and terminal Serving dispositions must be resolved separately; finite
// business refusal codes cannot be interpreted as a successful Result.
func DecodeResult(b []byte) (Result, error) {
	if len(b) != 46 || string(b[:4]) != "SV1R" || b[28] > 1 || b[29] > 1 {
		return Result{}, invalid
	}
	r := Result{Generation: binary.LittleEndian.Uint32(b[4:8]), Revision: binary.LittleEndian.Uint32(b[8:12]), Bucket: binary.LittleEndian.Uint64(b[12:20]), Operation: binary.LittleEndian.Uint64(b[20:28]), Pending: b[28] == 1, Deleted: b[29] == 1}
	copy(r.Key[:], b[30:])
	if r.Generation == 0 || r.Revision == 0 || r.Bucket == 0 || r.Bucket > MaxCounter || r.Operation == 0 || r.Operation > MaxCounter || r.Key == [16]byte{} {
		return Result{}, invalid
	}
	return r, nil
}
