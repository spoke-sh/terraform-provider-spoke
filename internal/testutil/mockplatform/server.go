// Package mockplatform is a test-only implementation of the provisional API.
// It models management intent, not object storage or a production service.
package mockplatform

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

type Request struct{ Method, Path string }

type Server struct {
	mu                       sync.Mutex
	HTTP                     *httptest.Server
	Scope                    api.Scope
	Token, TokenFile, CAFile string
	buckets                  map[string]api.Bucket
	management               map[string]api.Management
	operations               map[string]api.Operation
	intents                  map[string]api.Intent
	nonempty                 map[string]bool
	requests                 []Request
	unexpected               []Request
	nextID                   int
	hold, dropReply, denied  bool
	readError                string
	accepted                 chan api.Operation
}

func New(t testing.TB) *Server {
	t.Helper()
	s := &Server{
		Scope:   api.Scope{OrganizationID: "org-test", ProjectID: "project-test", EnvironmentID: "env-test"},
		Token:   "synthetic-test-token",
		buckets: map[string]api.Bucket{}, management: map[string]api.Management{}, operations: map[string]api.Operation{}, intents: map[string]api.Intent{}, nonempty: map[string]bool{},
		accepted: make(chan api.Operation, 32),
	}
	s.HTTP = httptest.NewTLSServer(http.HandlerFunc(s.serve))
	dir := t.TempDir()
	s.TokenFile, s.CAFile = filepath.Join(dir, "token"), filepath.Join(dir, "ca.pem")
	for path, contents := range map[string][]byte{
		s.TokenFile: []byte(s.Token),
		s.CAFile:    pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: s.HTTP.Certificate().Raw}),
	} {
		if err := os.WriteFile(path, contents, 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		s.HTTP.Close()
		s.mu.Lock()
		defer s.mu.Unlock()
		if len(s.unexpected) != 0 {
			t.Errorf("unexpected mock requests: %v", s.unexpected)
		}
	})
	return s
}

func (s *Server) Base() string {
	return "/v1/projects/" + s.Scope.ProjectID + "/environments/" + s.Scope.EnvironmentID
}

// Accepted signals durable fixture admission, allowing crash tests to avoid sleeps.
func (s *Server) Accepted() <-chan api.Operation { return s.accepted }
func (s *Server) Requests() []Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Request(nil), s.requests...)
}
func (s *Server) BucketCount() int            { s.mu.Lock(); defer s.mu.Unlock(); return len(s.buckets) }
func (s *Server) OperationCount() int         { s.mu.Lock(); defer s.mu.Unlock(); return len(s.operations) }
func (s *Server) HoldOperations(hold bool)    { s.mu.Lock(); defer s.mu.Unlock(); s.hold = hold }
func (s *Server) LoseNextReply()              { s.mu.Lock(); defer s.mu.Unlock(); s.dropReply = true }
func (s *Server) DenyAccess(deny bool)        { s.mu.Lock(); defer s.mu.Unlock(); s.denied = deny }
func (s *Server) FailBucketReads(code string) { s.mu.Lock(); defer s.mu.Unlock(); s.readError = code }
func (s *Server) SetNonempty(id string, nonempty bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nonempty[id] = nonempty
}
func (s *Server) Drift(id string, spec api.BucketSpec) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b := s.buckets[id]
	b.BucketSpec = spec
	b.Revision++
	s.buckets[id] = b
}
func (s *Server) Remove(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.buckets[id]
	if !ok {
		return
	}
	delete(s.buckets, id)
	m := s.management[b.ManagementKey]
	m.ResourceID = ""
	s.management[b.ManagementKey] = m
}
func (s *Server) CompleteAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, op := range s.operations {
		if op.Status == "accepted" {
			s.complete(id)
		}
	}
}

func send(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Spoke-API-Version", api.APIVersion)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func fail(w http.ResponseWriter, status int, code string) {
	send(w, status, map[string]string{"code": code})
}

func (s *Server) serve(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, Request{r.Method, r.URL.Path})
	if r.Header.Get("Authorization") != "Bearer "+s.Token {
		fail(w, 401, "unauthenticated")
		return
	}
	if s.denied || r.Header.Get("X-Spoke-Organization") != s.Scope.OrganizationID {
		fail(w, 403, "forbidden")
		return
	}
	if r.Header.Get("X-Spoke-API-Version") != api.APIVersion {
		fail(w, 400, "incompatible_api")
		return
	}
	if r.URL.RawQuery != "" {
		fail(w, 400, "invalid_request")
		return
	}
	if r.Method == "GET" && r.URL.Path == "/v1/projects/"+s.Scope.ProjectID {
		send(w, 200, api.Project{ID: s.Scope.ProjectID, OrganizationID: s.Scope.OrganizationID, Name: "Test project"})
		return
	}
	if r.Method == "GET" && r.URL.Path == s.Base() {
		send(w, 200, api.Environment{ID: s.Scope.EnvironmentID, ProjectID: s.Scope.ProjectID, OrganizationID: s.Scope.OrganizationID, Name: "Test environment"})
		return
	}
	if strings.HasPrefix(r.URL.Path, "/v1/projects/") && !strings.HasPrefix(r.URL.Path, s.Base()+"/") {
		fail(w, 403, "forbidden")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, s.Base()+"/")
	parts := strings.Split(path, "/")
	if r.Method == "GET" && len(parts) == 2 {
		switch parts[0] {
		case "management":
			m, ok := s.management[parts[1]]
			if !ok {
				fail(w, 404, "management_not_found")
				return
			}
			send(w, 200, m)
			return
		case "operations":
			op, ok := s.operations[parts[1]]
			if !ok {
				fail(w, 404, "resource_not_found")
				return
			}
			send(w, 200, op)
			return
		case "buckets":
			if s.readError != "" {
				fail(w, 503, s.readError)
				return
			}
			for _, op := range s.operations {
				if op.TargetID == parts[1] && op.Kind == "create" && op.Status == "accepted" {
					fail(w, 409, "operation_pending")
					return
				}
			}
			b, ok := s.buckets[parts[1]]
			if !ok {
				fail(w, 404, "resource_not_found")
				return
			}
			send(w, 200, b)
			return
		}
	}
	if (r.Method == "POST" && path == "buckets") || ((r.Method == "PATCH" || r.Method == "DELETE") && len(parts) == 2 && parts[0] == "buckets") {
		s.mutate(w, r, parts)
		return
	}
	s.unexpected = append(s.unexpected, Request{r.Method, r.URL.Path})
	fail(w, 404, "unsupported_route")
}

func (s *Server) mutate(w http.ResponseWriter, r *http.Request, parts []string) {
	if r.Header.Get("Content-Type") != "application/json" {
		fail(w, 400, "invalid_request")
		return
	}
	var req api.Mutation
	dec := json.NewDecoder(io.LimitReader(r.Body, 65537))
	dec.DisallowUnknownFields()
	if dec.Decode(&req) != nil || dec.Decode(new(any)) != io.EOF || !api.ValidID(req.ManagementKey) || req.ExpectedGeneration < 0 {
		fail(w, 400, "invalid_request")
		return
	}
	kind := map[string]string{"POST": "create", "PATCH": "update", "DELETE": "delete"}[r.Method]
	if req.Kind != kind || (kind == "create" && (req.TargetID != "" || req.ExpectedRevision != 0)) || (kind != "create" && (len(parts) != 2 || req.TargetID != parts[1] || req.ExpectedRevision < 1)) {
		fail(w, 400, "invalid_request")
		return
	}
	if kind != "delete" && (req.Spec == nil || len(req.Spec.Name) < 1 || len(req.Spec.Name) > 63 || strings.TrimSpace(req.Spec.Name) != req.Spec.Name || strings.ContainsAny(req.Spec.Name, "\r\n\t")) {
		fail(w, 400, "invalid_request")
		return
	}
	if kind == "delete" && req.Spec != nil {
		fail(w, 400, "invalid_request")
		return
	}
	// The fixture implements its own admission/CAS rules. It does not call the
	// client's mutation, retry, lookup, or settlement code.
	encoded, _ := json.Marshal(req.Intent)
	hash := sha256.Sum256(encoded)
	fingerprint := hex.EncodeToString(hash[:])
	m := s.management[req.ManagementKey]
	if m.Operation != nil && req.ExpectedGeneration == m.Generation-1 && m.Operation.Fingerprint == fingerprint {
		send(w, 202, m.Operation)
		return
	}
	if m.Generation != req.ExpectedGeneration {
		fail(w, 409, "generation_conflict")
		return
	}
	if m.Operation != nil && m.Operation.Status == "accepted" {
		fail(w, 409, "operation_pending")
		return
	}
	if kind == "create" {
		if m.ResourceID != "" {
			fail(w, 409, "ownership_conflict")
			return
		}
	} else {
		b, ok := s.buckets[req.TargetID]
		if !ok {
			fail(w, 404, "resource_not_found")
			return
		}
		if m.ResourceID != b.ID || b.ManagementKey != req.ManagementKey {
			fail(w, 409, "ownership_conflict")
			return
		}
		if b.Revision != req.ExpectedRevision {
			fail(w, 409, "revision_conflict")
			return
		}
		if kind == "delete" && b.DeletionProtection {
			fail(w, 409, "deletion_protected")
			return
		}
		if kind == "delete" && s.nonempty[b.ID] {
			fail(w, 409, "not_empty")
			return
		}
	}
	s.nextID++
	target := req.TargetID
	if kind == "create" {
		target = fmt.Sprintf("bucket-%d", s.nextID)
	}
	op := api.Operation{Scope: s.Scope, ID: fmt.Sprintf("op-%d", s.nextID), ManagementKey: req.ManagementKey, Fingerprint: fingerprint, Kind: kind, TargetID: target, Generation: m.Generation + 1, Status: "accepted"}
	s.operations[op.ID] = op
	s.intents[op.ID] = req.Intent
	m = api.Management{Scope: s.Scope, ManagementKey: req.ManagementKey, Generation: op.Generation, ResourceID: m.ResourceID, Operation: &op}
	s.management[req.ManagementKey] = m
	select {
	case s.accepted <- op:
	default:
	}
	// Completion is an explicit controller action, never a side effect of GET.
	if !s.hold {
		s.complete(op.ID)
	}
	if s.dropReply {
		s.dropReply = false
		conn, _, err := w.(http.Hijacker).Hijack()
		if err == nil {
			_ = conn.Close()
		}
		return
	}
	send(w, 202, op)
}

func (s *Server) complete(id string) {
	op := s.operations[id]
	intent := s.intents[id]
	m := s.management[op.ManagementKey]
	if op.Status != "accepted" {
		return
	}
	if op.Kind != "create" {
		b, ok := s.buckets[op.TargetID]
		switch {
		case !ok || b.Revision != intent.ExpectedRevision:
			op.Status, op.ErrorCode = "failed", "revision_conflict"
		case op.Kind == "delete" && b.DeletionProtection:
			op.Status, op.ErrorCode = "failed", "deletion_protected"
		case op.Kind == "delete" && s.nonempty[op.TargetID]:
			op.Status, op.ErrorCode = "failed", "not_empty"
		}
		if op.Status == "failed" {
			s.operations[id] = op
			m.Operation = &op
			s.management[op.ManagementKey] = m
			return
		}
	}
	switch op.Kind {
	case "create":
		s.buckets[op.TargetID] = api.Bucket{Scope: s.Scope, BucketSpec: *intent.Spec, ID: op.TargetID, ManagementKey: op.ManagementKey, Revision: 1}
		m.ResourceID = op.TargetID
	case "update":
		b := s.buckets[op.TargetID]
		b.BucketSpec = *intent.Spec
		b.Revision++
		s.buckets[b.ID] = b
	case "delete":
		delete(s.buckets, op.TargetID)
		delete(s.nonempty, op.TargetID)
		m.ResourceID = ""
	}
	op.Status = "succeeded"
	s.operations[id] = op
	m.Operation = &op
	s.management[op.ManagementKey] = m
}
