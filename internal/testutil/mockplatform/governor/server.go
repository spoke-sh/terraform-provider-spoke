// Package governor provides a synthetic persisted HTTP contract peer. It has
// no runner scheduler, native persistence, GitHub adapter or cloud effects.
package governor

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

const Profile = "spoke.platform.http.ci-governor.v1"
const Version = "ci-governor-v1"

type Identity struct {
	Principal string
	Operator  bool
}
type object = map[string]any
type slot struct {
	Principal string            `json:"principal"`
	Latest    uint64            `json:"latest"`
	Target    string            `json:"target"`
	Retired   bool              `json:"retired"`
	Requests  map[string]object `json:"requests"`
	Expired   map[string]bool   `json:"expired"`
}
type ledger struct {
	Profile      string            `json:"profile"`
	Schema       string            `json:"schema_sha256"`
	Scope        string            `json:"scope"`
	Next         uint64            `json:"next"`
	Snapshot     uint64            `json:"snapshot"`
	Keys         map[string]*slot  `json:"keys"`
	Governors    map[string]object `json:"governors"`
	Entitlements map[string]string `json:"entitlements"`
}
type Server struct {
	mu         sync.Mutex
	path, base string
	schemas    map[string]*jsonschema.Schema
	tokens     map[string]Identity
	state      ledger
	hold       bool
	faults     map[string]int
}
type offline struct{}

func (offline) Load(string) (any, error) { return nil, errors.New("external schema disabled") }

// New reloads exact current-format synthetic state, rejecting incompatible files.
// Credentials are supplied externally and never saved in the ledger.
func New(path, base string, schema []byte, tokens map[string]Identity) (*Server, error) {
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schema))
	if err != nil {
		return nil, err
	}
	c := jsonschema.NewCompiler()
	c.UseLoader(offline{})
	const uri = "https://spoke.sh/contracts/http/ci-governor-v1.schema.json"
	if err = c.AddResource(uri, doc); err != nil {
		return nil, err
	}
	s := &Server{path: path, base: base, tokens: tokens, schemas: map[string]*jsonschema.Schema{}, faults: map[string]int{}}
	for name := range doc.(object)["$defs"].(object) {
		v, e := c.Compile(uri + "#/$defs/" + name)
		if e != nil {
			return nil, e
		}
		s.schemas[name] = v
	}
	if !s.valid("Scope", base) {
		return nil, errors.New("invalid fixture scope")
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(schema))
	s.state = ledger{Profile: Profile, Schema: digest, Scope: base, Next: 1, Snapshot: 1, Keys: map[string]*slot{}, Governors: map[string]object{}, Entitlements: map[string]string{}}
	raw, err := os.ReadFile(path)
	if err == nil {
		d := json.NewDecoder(bytes.NewReader(raw))
		d.DisallowUnknownFields()
		if d.Decode(&s.state) != nil || d.Decode(new(any)) != io.EOF || s.state.Profile != Profile || s.state.Schema != digest || s.state.Scope != base || s.state.Keys == nil || s.state.Governors == nil || s.state.Entitlements == nil || s.state.Next == 0 || s.state.Snapshot == 0 {
			return nil, errors.New("incompatible governor fixture ledger")
		}
		for _, k := range s.state.Keys {
			if k == nil || k.Requests == nil || k.Expired == nil {
				return nil, errors.New("incomplete fixture key")
			}
			for _, op := range k.Requests {
				if !s.valid("Operation", op) {
					return nil, errors.New("invalid retained operation")
				}
			}
		}
		for _, g := range s.state.Governors {
			if !s.valid("Governor", g) {
				return nil, errors.New("invalid retained governor")
			}
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	return s, nil
}
func (s *Server) valid(kind string, v any) bool {
	return s.schemas[kind] != nil && s.schemas[kind].Validate(v) == nil
}
func number(v any) uint64 {
	switch n := v.(type) {
	case float64:
		return uint64(n)
	case json.Number:
		u, _ := strconv.ParseUint(string(n), 10, 64)
		return u
	case uint64:
		return n
	case int:
		return uint64(n)
	}
	return 0
}
func clone[T any](v T) T {
	raw, _ := json.Marshal(v)
	var out T
	_ = json.Unmarshal(raw, &out)
	return out
}
func (s *Server) save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(s.path), "governor-*.tmp")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if err = f.Chmod(0600); err == nil {
		err = json.NewEncoder(f).Encode(s.state)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err = os.Rename(name, s.path); err != nil {
		return err
	}
	dir, err := os.Open(filepath.Dir(s.path))
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
func (s *Server) transact(fn func() error) error {
	prior := clone(s.state)
	if err := fn(); err != nil {
		s.state = prior
		return err
	}
	s.state.Snapshot++
	if err := s.save(); err != nil {
		s.state = prior
		return err
	}
	return nil
}
func (s *Server) newID(prefix string) string {
	id := fmt.Sprintf("%s-%08d", prefix, s.state.Next)
	s.state.Next++
	return id
}
func (s *Server) SetHold(hold bool) { s.mu.Lock(); defer s.mu.Unlock(); s.hold = hold }
func (s *Server) Drop(stage, when string, count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.faults[stage+"/"+when] = count
}
func (s *Server) Snapshot() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, _ := json.Marshal(s.state)
	return raw
}
func (s *Server) SeedEntitlement(id, owner string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.transact(func() error { s.state.Entitlements[id] = owner; return nil })
}
func (s *Server) SeedUsage(id string, jobs, cpus, usage uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.transact(func() error {
		g := s.state.Governors[id]
		if g == nil {
			return errors.New("unknown governor")
		}
		g["active_jobs"] = jobs
		g["allocated_vcpus"] = cpus
		g["unsettled_usage"] = usage
		return nil
	})
}
func (s *Server) Drift(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.transact(func() error {
		g := s.state.Governors[id]
		if g == nil {
			return errors.New("unknown governor")
		}
		g["revision"] = number(g["revision"]) + 1
		return nil
	})
}
func terminal(op object) bool {
	return op["status"] == "applied" || op["status"] == "refused" || op["status"] == "aborted"
}
func (s *Server) Expire(key string, seq uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.transact(func() error {
		k := s.state.Keys[key]
		n := strconv.FormatUint(seq, 10)
		if k == nil || !terminal(k.Requests[n]) {
			return errors.New("cannot expire pending or absent request")
		}
		k.Expired[n] = true
		return nil
	})
}
func (s *Server) Settle() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.transact(func() error { s.settle(); return nil })
}
func (s *Server) settle() {
	for key, k := range s.state.Keys {
		for _, op := range k.Requests {
			if op["status"] != "accepted" {
				continue
			}
			intent := op["intent"].(object)
			kind := intent["kind"].(string)
			if kind == "create" {
				id := s.newID("governor")
				k.Target = id
				op["resource_id"] = id
				s.state.Governors[id] = object{"scope": s.base, "principal_id": k.Principal, "id": id, "management_key": key, "revision": uint64(1), "active_revision": uint64(0), "policy": clone(intent["policy"]), "state": "disabled", "blockers": []any{"execution_unqualified"}, "active_jobs": uint64(0), "allocated_vcpus": uint64(0), "unsettled_usage": uint64(0)}
			} else {
				g := s.state.Governors[k.Target]
				if g == nil || number(g["revision"]) != number(intent["expected_revision"]) {
					op["status"] = "refused"
					op["reason"] = "revision_conflict"
					continue
				}
				if kind == "update" {
					g["policy"] = clone(intent["policy"])
					g["revision"] = number(g["revision"]) + 1
				} else {
					if g["policy"].(object)["deletion_protection"] == true {
						op["status"] = "refused"
						op["reason"] = "deletion_protected"
						continue
					}
					g["state"] = "draining"
					g["blockers"] = []any{}
					if number(g["active_jobs"]) > 0 || number(g["allocated_vcpus"]) > 0 {
						g["blockers"] = append(g["blockers"].([]any), "leases_outstanding")
					}
					if number(g["unsettled_usage"]) > 0 {
						g["blockers"] = append(g["blockers"].([]any), "usage_unsettled")
					}
					if len(g["blockers"].([]any)) > 0 {
						op["reason"] = "draining"
						continue
					}
					delete(s.state.Entitlements, g["policy"].(object)["admission_policy_id"].(string))
					delete(s.state.Governors, k.Target)
					k.Retired = true
				}
			}
			op["status"] = "applied"
			op["reason"] = "none"
		}
	}
}
func (s *Server) caps(principal string) object {
	return object{"scope": s.base, "principal_id": principal, "version": Version, "profile": Profile, "status": "provisional", "capabilities": object{"governors": true, "prepare_commit": true, "pending": true, "handoff": false, "execution": false, "billing": false, "warm_capacity": false}, "limits": object{"body_bytes": 16384, "response_bytes": 131072, "page_size": 8, "governors": 32, "management_keys": 64, "receipts": 256}}
}
func fingerprint(base, principal, key string, seq uint64, intent object) string {
	raw, _ := json.Marshal(object{"scope": base, "principal_id": principal, "management_key": key, "sequence": seq, "intent": intent})
	return fmt.Sprintf("%x", sha256.Sum256(append([]byte(Profile+"\n"), raw...)))
}
func (s *Server) fault(w http.ResponseWriter, key string) bool {
	if s.faults[key] <= 0 {
		return false
	}
	s.faults[key]--
	if h, ok := w.(http.Hijacker); ok {
		c, _, err := h.Hijack()
		if err == nil {
			_ = c.Close()
		}
	}
	return true
}
func reply(w http.ResponseWriter, status int, v any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, status int, code string) { reply(w, status, object{"code": code}) }

// jsonValue independently rejects duplicate object keys before schema checking.
func jsonValue(d *json.Decoder, depth int) (any, error) {
	if depth > 24 {
		return nil, errors.New("JSON depth")
	}
	t, err := d.Token()
	if err != nil {
		return nil, err
	}
	switch t {
	case json.Delim('{'):
		o := object{}
		for d.More() {
			k, e := d.Token()
			if e != nil {
				return nil, e
			}
			key, ok := k.(string)
			if !ok {
				return nil, errors.New("key")
			}
			if _, ok = o[key]; ok {
				return nil, errors.New("duplicate")
			}
			v, e := jsonValue(d, depth+1)
			if e != nil {
				return nil, e
			}
			o[key] = v
		}
		end, e := d.Token()
		if e != nil || end != json.Delim('}') {
			return nil, errors.New("object")
		}
		return o, nil
	case json.Delim('['):
		a := []any{}
		for d.More() {
			v, e := jsonValue(d, depth+1)
			if e != nil {
				return nil, e
			}
			a = append(a, v)
		}
		end, e := d.Token()
		if e != nil || end != json.Delim(']') {
			return nil, errors.New("array")
		}
		return a, nil
	default:
		if _, ok := t.(json.Delim); ok {
			return nil, errors.New("delimiter")
		}
		return t, nil
	}
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Spoke-API-Version", Version)
	w.Header().Set("X-Spoke-Management-Profile", Profile)
	w.Header().Set("Cache-Control", "no-store")
	identity, ok := s.tokens[strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")]
	if !ok || !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
		fail(w, 401, "unauthenticated")
		return
	}
	if r.Header.Get("Cookie") != "" || r.Header.Get("Origin") != "" || r.Header.Get("X-Forwarded-User") != "" {
		fail(w, 403, "forbidden")
		return
	}
	org := strings.Split(s.base, "/")[4]
	if r.Header.Get("X-Spoke-Organization") != org || !strings.HasPrefix(r.URL.Path, s.base+"/") {
		fail(w, 403, "forbidden")
		return
	}
	if r.Header.Get("X-Spoke-API-Version") != Version {
		fail(w, 400, "incompatible_api")
		return
	}
	if r.Header.Get("Accept") != "application/json" {
		fail(w, 422, "invalid_request")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, s.base)
	if r.Method == "GET" {
		s.read(w, r, path, identity)
		return
	}
	if !identity.Operator {
		fail(w, 403, "forbidden")
		return
	}
	if r.URL.RawQuery != "" || r.Header.Get("Content-Type") != "application/json" {
		fail(w, 422, "invalid_request")
		return
	}
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) < 3 || len(parts) > 4 || parts[0] != "requests" || !s.valid("Key", parts[1]) {
		fail(w, 404, "unsupported_route")
		return
	}
	seq, err := strconv.ParseUint(parts[2], 10, 32)
	if err != nil || strconv.FormatUint(seq, 10) != parts[2] {
		fail(w, 422, "invalid_request")
		return
	}
	stage := "prepare"
	kind := "Intent"
	if len(parts) == 4 {
		stage = parts[3]
		kind = "Commit"
	}
	if (stage == "prepare" && (r.Method != "PUT" || len(parts) != 3)) || ((stage == "commit" || stage == "abort") && r.Method != "POST") || (stage != "prepare" && stage != "commit" && stage != "abort") {
		fail(w, 404, "unsupported_route")
		return
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, 16385))
	if err != nil || len(raw) > 16384 {
		fail(w, 413, "body_too_large")
		return
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	v, err := jsonValue(d, 0)
	if err != nil {
		fail(w, 422, "invalid_request")
		return
	}
	if _, e := d.Token(); e != io.EOF || !s.valid(kind, v) {
		fail(w, 422, "invalid_request")
		return
	}
	if s.fault(w, stage+"/before") {
		return
	}
	status, op, code := s.mutate(identity, parts[1], seq, stage, v.(object))
	if code != "" {
		fail(w, status, code)
		return
	}
	if s.fault(w, stage+"/after") {
		return
	}
	w.Header().Set("Location", s.base+"/requests/"+parts[1]+"/"+parts[2])
	reply(w, status, op)
}

func (s *Server) mutate(who Identity, key string, seq uint64, stage string, body object) (int, object, string) {
	k := s.state.Keys[key]
	n := strconv.FormatUint(seq, 10)
	if k != nil && k.Principal != who.Principal {
		return 403, nil, "forbidden"
	}
	if k != nil && k.Expired[n] {
		return 410, nil, "receipt_expired"
	}
	var prior object
	if k != nil {
		prior = k.Requests[n]
	}
	if stage == "prepare" {
		fp := fingerprint(s.base, who.Principal, key, seq, body)
		if prior != nil {
			if prior["fingerprint"] != fp {
				return 409, nil, "intent_conflict"
			}
			return 200, clone(prior), ""
		}
		if k == nil && seq != 0 || k != nil && (k.Retired || k.Latest == 4294967295 || seq != k.Latest+1) {
			return 409, nil, "sequence_conflict"
		}
		if k != nil && !terminal(k.Requests[strconv.FormatUint(k.Latest, 10)]) {
			return 409, nil, "operation_pending"
		}
		kind := body["kind"].(string)
		if kind == "create" && k != nil && k.Target != "" {
			return 409, nil, "ownership_conflict"
		}
		if kind != "create" && (k == nil || k.Target == "" || body["target_id"] != k.Target) {
			return 403, nil, "forbidden"
		}
		if kind != "retire" {
			admission := body["policy"].(object)["admission_policy_id"].(string)
			if owner := s.state.Entitlements[admission]; owner != "" && owner != key {
				return 409, nil, "ownership_conflict"
			}
			if kind == "update" && s.state.Governors[k.Target]["policy"].(object)["admission_policy_id"] != admission {
				return 422, nil, "unsupported_capability"
			}
		}
		total := 0
		for _, v := range s.state.Keys {
			total += len(v.Requests)
		}
		if total >= 256 || k == nil && len(s.state.Keys) >= 64 || kind == "create" && len(s.state.Entitlements) >= 32 {
			return 429, nil, "capacity_exhausted"
		}
		var op object
		err := s.transact(func() error {
			if k == nil {
				k = &slot{Principal: who.Principal, Requests: map[string]object{}, Expired: map[string]bool{}}
				s.state.Keys[key] = k
			}
			var target any
			if k.Target != "" {
				target = k.Target
			}
			op = object{"scope": s.base, "principal_id": who.Principal, "id": s.newID("operation"), "management_key": key, "sequence": seq, "fingerprint": fp, "intent": clone(body), "status": "prepared", "resource_id": target, "reason": "none"}
			k.Latest = seq
			k.Requests[n] = op
			if kind == "create" {
				s.state.Entitlements[body["policy"].(object)["admission_policy_id"].(string)] = key
			}
			return nil
		})
		if err != nil {
			return 503, nil, "unavailable"
		}
		return 201, clone(op), ""
	}
	if prior == nil {
		return 404, nil, "request_not_found"
	}
	if body["fingerprint"] != prior["fingerprint"] {
		return 409, nil, "intent_conflict"
	}
	if stage == "abort" && prior["status"] != "prepared" && prior["status"] != "aborted" {
		return 409, nil, "operation_pending"
	}
	if terminal(prior) {
		return 200, clone(prior), ""
	}
	err := s.transact(func() error {
		if stage == "abort" {
			prior["status"] = "aborted"
			prior["reason"] = "aborted"
			intent := prior["intent"].(object)
			if intent["kind"] == "create" {
				delete(s.state.Entitlements, intent["policy"].(object)["admission_policy_id"].(string))
			}
		} else {
			prior["status"] = "accepted"
			if !s.hold {
				s.settle()
			}
		}
		return nil
	})
	if err != nil {
		return 503, nil, "unavailable"
	}
	status := 200
	if prior["status"] == "accepted" {
		status = 202
	}
	return status, clone(prior), ""
}

func (s *Server) read(w http.ResponseWriter, r *http.Request, path string, who Identity) {
	q := r.URL.Query()
	for key, v := range q {
		if (key != "cursor" && key != "state") || len(v) != 1 {
			fail(w, 422, "invalid_request")
			return
		}
	}
	if path == "/capabilities" && len(q) == 0 {
		reply(w, 200, s.caps(who.Principal))
		return
	}
	if path == "/governors" || path == "/management" {
		pending := path == "/management"
		if pending && q.Get("state") != "pending" || !pending && q.Has("state") {
			fail(w, 422, "invalid_request")
			return
		}
		var items []object
		items = []object{}
		if pending {
			for _, k := range s.state.Keys {
				if k.Principal != who.Principal {
					continue
				}
				op := k.Requests[strconv.FormatUint(k.Latest, 10)]
				if !terminal(op) {
					items = append(items, clone(op))
				}
			}
		} else {
			for _, g := range s.state.Governors {
				if g["principal_id"] == who.Principal {
					items = append(items, clone(g))
				}
			}
		}
		slices.SortFunc(items, func(a, b object) int { return strings.Compare(a["id"].(string), b["id"].(string)) })
		offset := 0
		if q.Has("cursor") {
			encoded := q.Get("cursor")
			if encoded == "" || !s.valid("Cursor", encoded) {
				fail(w, 503, "incomplete_result")
				return
			}
			raw, err := base64.RawURLEncoding.DecodeString(encoded)
			parts := strings.Split(string(raw), "|")
			if err != nil || len(parts) != 4 || parts[0] != who.Principal || parts[1] != path {
				fail(w, 503, "incomplete_result")
				return
			}
			if parts[2] != strconv.FormatUint(s.state.Snapshot, 10) {
				fail(w, 409, "stale_snapshot")
				return
			}
			offset, err = strconv.Atoi(parts[3])
			if err != nil || offset <= 0 || offset >= len(items) || offset%8 != 0 {
				fail(w, 503, "incomplete_result")
				return
			}
		}
		end := min(offset+8, len(items))
		next := ""
		if end < len(items) {
			next = base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("%s|%s|%d|%d", who.Principal, path, s.state.Snapshot, end)))
		}
		reply(w, 200, object{"scope": s.base, "principal_id": who.Principal, "snapshot": s.state.Snapshot, "complete": end == len(items), "next_cursor": next, "items": items[offset:end]})
		return
	}
	if len(q) != 0 {
		fail(w, 422, "invalid_request")
		return
	}
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) == 2 && parts[0] == "governors" && s.valid("ID", parts[1]) {
		g := s.state.Governors[parts[1]]
		if g == nil {
			fail(w, 404, "resource_not_found")
		} else if g["principal_id"] != who.Principal {
			fail(w, 403, "forbidden")
		} else {
			reply(w, 200, g)
		}
		return
	}
	if len(parts) >= 2 && (parts[0] == "management" || parts[0] == "requests") && s.valid("Key", parts[1]) {
		if parts[0] == "management" && len(parts) != 2 || parts[0] == "requests" && len(parts) != 3 {
			fail(w, 404, "unsupported_route")
			return
		}
		k := s.state.Keys[parts[1]]
		if k == nil {
			fail(w, 404, "request_not_found")
			return
		}
		if k.Principal != who.Principal {
			fail(w, 403, "forbidden")
			return
		}
		n := strconv.FormatUint(k.Latest, 10)
		if parts[0] == "requests" {
			n = parts[2]
			v, err := strconv.ParseUint(n, 10, 32)
			if err != nil || strconv.FormatUint(v, 10) != n {
				fail(w, 422, "invalid_request")
				return
			}
		}
		if k.Expired[n] {
			fail(w, 410, "receipt_expired")
			return
		}
		op := k.Requests[n]
		if op == nil {
			fail(w, 404, "request_not_found")
		} else {
			reply(w, 200, op)
		}
		return
	}
	fail(w, 404, "unsupported_route")
}
