// Package mockplatform implements a synthetic, test-only management service.
package mockplatform

import (
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net"
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
type Principal struct {
	ID, OrganizationID string
	Admin, Disabled    bool
}
type ledger struct {
	Version    string
	Objects    map[string]api.Object
	Management map[string]api.Management
	Operations map[string]api.Operation
	Intents    map[string]api.Intent
	Principals map[string]Principal
	Expired    map[string]string
	Nonempty   map[string]bool
	NextID     int
	Epoch      int64
}
type fault struct {
	Method, Path, Code string
	Status, Remaining  int
}
type Barrier struct {
	Reached chan api.Operation
	Release chan struct{}
}
type Server struct {
	errorCanary              string
	mu                       sync.Mutex
	HTTP                     *httptest.Server
	Scope                    api.Scope
	Token, TokenFile, CAFile string
	state                    ledger
	ledgerFile               string
	requests, unexpected     []Request
	hold, dropReply, denied  bool
	readError                string
	accepted                 chan api.Operation
	barriers                 map[string]*Barrier
	faults                   []fault
	quota                    map[string]int
	cursors                  map[string]cursor
}

func New(t testing.TB) *Server {
	t.Helper()
	s := &Server{Scope: api.Scope{OrganizationID: "org-test", ProjectID: "project-test", EnvironmentID: "env-test"}, Token: "synthetic-test-token", accepted: make(chan api.Operation, 128), barriers: map[string]*Barrier{}, quota: map[string]int{}}
	s.cursors = map[string]cursor{}
	s.state = ledger{Version: api.APIVersion, Objects: map[string]api.Object{}, Management: map[string]api.Management{}, Operations: map[string]api.Operation{}, Intents: map[string]api.Intent{}, Principals: map[string]Principal{}, Expired: map[string]string{}, Nonempty: map[string]bool{}, Epoch: 1}
	dir := t.TempDir()
	s.ledgerFile = filepath.Join(dir, "ledger.json")
	s.TokenFile, s.CAFile = filepath.Join(dir, "token"), filepath.Join(dir, "ca.pem")
	s.seedTenant(s.Scope, s.Token, "principal-admin")
	s.persist()
	s.HTTP = httptest.NewTLSServer(http.HandlerFunc(s.serve))
	for path, contents := range map[string][]byte{s.TokenFile: []byte(s.Token), s.CAFile: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: s.HTTP.Certificate().Raw})} {
		if err := os.WriteFile(path, contents, 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		s.HTTP.Close()
		s.mu.Lock()
		defer s.mu.Unlock()
		if len(s.unexpected) > 0 {
			t.Errorf("unexpected mock requests: %v", s.unexpected)
		}
	})
	return s
}
func (s *Server) seedTenant(scope api.Scope, token, owner string) {
	s.state.Principals[token] = Principal{ID: owner, OrganizationID: scope.OrganizationID, Admin: true}
	parent := api.Scope{OrganizationID: scope.OrganizationID}
	s.state.Objects[parent.Path()+"/projects/"+scope.ProjectID] = api.Object{Bucket: api.Bucket{Scope: parent, BucketSpec: api.BucketSpec{Name: "Test project", DeletionProtection: true}, ID: scope.ProjectID, ManagementKey: "seed-project", OwnerID: owner, Revision: 1}, ResourceKind: "project"}
	parent.ProjectID = scope.ProjectID
	s.state.Objects[parent.Path()+"/environments/"+scope.EnvironmentID] = api.Object{Bucket: api.Bucket{Scope: parent, BucketSpec: api.BucketSpec{Name: "Test environment", DeletionProtection: true}, ID: scope.EnvironmentID, ManagementKey: "seed-environment", OwnerID: owner, Revision: 1}, ResourceKind: "environment"}
}
func (s *Server) SeedTenant(scope api.Scope, token, owner string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seedTenant(scope, token, owner)
	s.state.Epoch++
	s.persist()
}

// AddCredential is test-only external issuance, never a management API response.
func (s *Server) AddCredential(token string, p Principal) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Principals[token] = p
	s.persist()
}
func (s *Server) RevokePrincipal(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for token, p := range s.state.Principals {
		if p.ID == id {
			p.Disabled = true
			s.state.Principals[token] = p
		}
	}
	s.persist()
}
func (s *Server) Base() string                   { return s.Scope.Path() }
func (s *Server) Accepted() <-chan api.Operation { return s.accepted }
func (s *Server) Requests() []Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Request(nil), s.requests...)
}
func (s *Server) Count(kind string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, o := range s.state.Objects {
		if o.ResourceKind == kind {
			n++
		}
	}
	return n
}
func (s *Server) BucketCount() int { return s.Count("bucket") }
func (s *Server) OperationCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.state.Operations)
}
func (s *Server) HoldOperations(v bool) { s.mu.Lock(); defer s.mu.Unlock(); s.hold = v }
func (s *Server) LoseNextReply()        { s.mu.Lock(); defer s.mu.Unlock(); s.dropReply = true }
func (s *Server) DenyAccess(v bool)     { s.mu.Lock(); defer s.mu.Unlock(); s.denied = v }
func (s *Server) SetErrorCanary(value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errorCanary = value
}
func (s *Server) FailBucketReads(code string) { s.mu.Lock(); defer s.mu.Unlock(); s.readError = code }
func (s *Server) SetQuota(kind string, n int) { s.mu.Lock(); defer s.mu.Unlock(); s.quota[kind] = n }
func (s *Server) FailNext(method, path string, status int, code string, count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.faults = append(s.faults, fault{method, path, code, status, count})
}
func (s *Server) BlockNext(phase string) *Barrier {
	s.mu.Lock()
	defer s.mu.Unlock()
	b := &Barrier{make(chan api.Operation, 1), make(chan struct{})}
	s.barriers[phase] = b
	return b
}

// pause releases the store lock, so process death and independent operations can proceed.
func (s *Server) pause(r *http.Request, phase string, op api.Operation) bool {
	b := s.barriers[phase]
	if b == nil {
		return true
	}
	delete(s.barriers, phase)
	b.Reached <- op
	s.mu.Unlock()
	defer s.mu.Lock()
	select {
	case <-b.Release:
		return true
	case <-r.Context().Done():
		return false
	}
}
func (s *Server) SetNonempty(id string, v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Nonempty[id] = v
	s.persist()
}
func (s *Server) Drift(id string, spec api.BucketSpec) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, o := range s.state.Objects {
		if o.ID == id {
			o.BucketSpec = spec
			o.Revision++
			s.state.Objects[key] = o
			s.state.Epoch++
			s.persist()
			return
		}
	}
}
func (s *Server) Remove(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, o := range s.state.Objects {
		if o.ID == id {
			delete(s.state.Objects, key)
			mk := receiptKey(o.Scope, o.ResourceKind, o.ManagementKey)
			m := s.state.Management[mk]
			m.ResourceID = ""
			s.state.Management[mk] = m
			s.state.Epoch++
			s.persist()
			return
		}
	}
}
func (s *Server) CompleteAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, op := range s.state.Operations {
		if op.Status == "accepted" {
			s.complete(id)
		}
	}
}
func (s *Server) ExpireReceipt(scope api.Scope, kind, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	mk := receiptKey(scope, kind, key)
	m, ok := s.state.Management[mk]
	if !ok || m.Operation.Status == "accepted" {
		return fmt.Errorf("only terminal receipts may expire")
	}
	s.state.Expired[mk] = m.OwnerID
	delete(s.state.Management, mk)
	s.persist()
	return nil
}

func (s *Server) persist() {
	// Atomic replacement and fsync before any acceptance notification or response.
	b, err := json.Marshal(s.state)
	if err != nil {
		panic(err)
	}
	f, err := os.OpenFile(s.ledgerFile+".tmp", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		panic(err)
	}
	if _, err = f.Write(b); err != nil {
		_ = f.Close()
		panic(err)
	}
	if err = f.Sync(); err != nil {
		_ = f.Close()
		panic(err)
	}
	if err = f.Close(); err != nil {
		panic(err)
	}
	if err = os.Rename(s.ledgerFile+".tmp", s.ledgerFile); err != nil {
		panic(err)
	}
	d, err := os.Open(filepath.Dir(s.ledgerFile))
	if err != nil {
		panic(err)
	}
	defer d.Close()
	if err = d.Sync(); err != nil {
		panic(err)
	}
}

// Restart discards the in-memory ledger and reloads the committed synthetic file.
// It retains the endpoint and TLS identity, just as an independently restarted service would.
func (s *Server) Restart(t testing.TB) {
	t.Helper()
	addr := s.HTTP.Listener.Addr().String()
	tlsConfig := s.HTTP.TLS.Clone()
	s.HTTP.Close()
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := os.ReadFile(s.ledgerFile)
	if err != nil {
		t.Fatal(err)
	}
	var restored ledger
	if err = json.Unmarshal(b, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.Version != api.APIVersion {
		t.Fatal("wrong fixture ledger version")
	}
	s.state = restored
	s.cursors = map[string]cursor{}
	h := httptest.NewUnstartedServer(http.HandlerFunc(s.serve))
	h.Listener.Close()
	h.Listener, err = net.Listen("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	h.TLS = tlsConfig
	h.StartTLS()
	s.HTTP = h
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
func receiptKey(scope api.Scope, kind, key string) string {
	return scope.Path() + "/management/" + kind + "/" + key
}
func objectPath(scope api.Scope, kind, id string) string {
	return scope.Path() + "/" + api.Collection(kind) + "/" + id
}
func kindFor(collection string) string {
	for _, k := range []string{"project", "environment", "service_account", "grant", "bucket"} {
		if api.Collection(k) == collection {
			return k
		}
	}
	return ""
}

func route(path string) (api.Scope, []string, bool) {
	p := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(p) < 3 || p[0] != "v2" || p[1] != "organizations" || !api.ValidID(p[2]) {
		return api.Scope{}, nil, false
	}
	scope := api.Scope{OrganizationID: p[2]}
	p = p[3:]
	if len(p) >= 2 && p[0] == "projects" {
		if !api.ValidID(p[1]) {
			return scope, nil, false
		}
		scope.ProjectID = p[1]
		p = p[2:]
	}
	if len(p) >= 2 && p[0] == "environments" {
		if scope.ProjectID == "" || !api.ValidID(p[1]) {
			return scope, nil, false
		}
		scope.EnvironmentID = p[1]
		p = p[2:]
	}
	if len(p) == 0 {
		if scope.EnvironmentID != "" {
			p = []string{"environments", scope.EnvironmentID}
			scope.EnvironmentID = ""
		} else if scope.ProjectID != "" {
			p = []string{"projects", scope.ProjectID}
			scope.ProjectID = ""
		}
	}
	return scope, p, true
}
func (s *Server) currentPrincipal(id, organization string) Principal {
	for _, p := range s.state.Principals {
		if p.ID == id && p.OrganizationID == organization && !p.Disabled {
			return p
		}
	}
	return Principal{Disabled: true}
}

func (s *Server) can(p Principal, path, permission string) bool {
	if p.Disabled {
		return false
	}
	if p.Admin {
		return path == "/v2/organizations/"+p.OrganizationID || strings.HasPrefix(path, "/v2/organizations/"+p.OrganizationID+"/")
	}
	// Issued service-account credentials stop working when their subject is retired.
	account := s.state.Objects["/v2/organizations/"+p.OrganizationID+"/service_accounts/"+p.ID]
	if account.ID == "" {
		return false
	}
	for _, o := range s.state.Objects {
		if o.ResourceKind == "grant" && o.OrganizationID == p.OrganizationID && o.SubjectID == p.ID && (o.Permission == permission || o.Permission == "manage") && (path == o.Target || strings.HasPrefix(path, o.Target+"/")) {
			return true
		}
	}
	return false
}

// Capability metadata describes the API, not the scope's object inventory.
// A caller authorized for one descendant may negotiate without listing siblings.
func (s *Server) canInspectScope(p Principal, scope api.Scope) bool {
	if s.can(p, scope.Path(), "read") {
		return true
	}
	if p.Disabled || s.state.Objects["/v2/organizations/"+p.OrganizationID+"/service_accounts/"+p.ID].ID == "" {
		return false
	}
	for _, o := range s.state.Objects {
		if o.ResourceKind == "grant" && o.OrganizationID == p.OrganizationID && o.SubjectID == p.ID && strings.HasPrefix(o.Target, scope.Path()+"/") {
			return true
		}
	}
	return false
}

func (s *Server) receiptPath(scope api.Scope, kind, key string) string {
	m := s.state.Management[receiptKey(scope, kind, key)]
	if m.Operation != nil {
		return objectPath(scope, kind, m.Operation.TargetID)
	}
	return scope.Path()
}
func (s *Server) parentExists(scope api.Scope) bool {
	if scope.ProjectID != "" {
		parent := api.Scope{OrganizationID: scope.OrganizationID}
		if s.state.Objects[objectPath(parent, "project", scope.ProjectID)].ID == "" {
			return false
		}
	}
	if scope.EnvironmentID != "" {
		parent := scope
		parent.EnvironmentID = ""
		if s.state.Objects[objectPath(parent, "environment", scope.EnvironmentID)].ID == "" {
			return false
		}
	}
	return true
}
func (s *Server) serve(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, Request{r.Method, r.URL.Path})
	p, ok := s.state.Principals[strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")]
	if !ok || !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
		fail(w, 401, "unauthenticated")
		return
	}
	if s.denied || p.Disabled || p.OrganizationID != r.Header.Get("X-Spoke-Organization") {
		fail(w, 403, "forbidden")
		return
	}
	if r.Header.Get("X-Spoke-API-Version") != api.APIVersion {
		fail(w, 400, "incompatible_api")
		return
	}
	scope, parts, valid := route(r.URL.Path)
	if !valid {
		s.unexpected = append(s.unexpected, Request{r.Method, r.URL.Path})
		fail(w, 404, "unsupported_route")
		return
	}
	if scope.OrganizationID != p.OrganizationID {
		fail(w, 403, "forbidden")
		return
	}
	if r.URL.RawQuery != "" && !(r.Method == "GET" && len(parts) == 1 && kindFor(parts[0]) != "") {
		fail(w, 400, "invalid_request")
		return
	}
	permission := "read"
	if r.Method != "GET" {
		permission = "manage"
	}
	authPath := scope.Path()
	if len(parts) == 2 && kindFor(parts[0]) != "" {
		authPath = r.URL.Path
	}
	if r.Method == "GET" && len(parts) == 3 && parts[0] == "management" {
		authPath = s.receiptPath(scope, parts[1], parts[2])
	}
	if r.Method == "GET" && len(parts) == 2 && parts[0] == "operations" {
		if op, ok := s.state.Operations[parts[1]]; ok && op.Scope == scope {
			authPath = objectPath(scope, op.ResourceKind, op.TargetID)
		}
	}
	allowed := s.can(p, authPath, permission)
	if r.Method == "GET" && len(parts) == 1 && parts[0] == "capabilities" {
		allowed = s.canInspectScope(p, scope)
	}
	if !allowed {
		fail(w, 403, "forbidden")
		return
	}
	if !s.parentExists(scope) {
		fail(w, 409, "dependency_conflict")
		return
	}
	for i := range s.faults {
		f := &s.faults[i]
		if f.Remaining > 0 && f.Method == r.Method && f.Path == r.URL.Path {
			f.Remaining--
			if f.Status == 429 {
				w.Header().Set("Retry-After", "1")
			}
			fail(w, f.Status, f.Code)
			return
		}
	}
	if r.Method == "GET" && len(parts) == 1 && parts[0] == "capabilities" {
		if r.URL.RawQuery != "" {
			fail(w, 400, "invalid_request")
			return
		}
		kinds := []string{}
		for _, kind := range []string{"project", "environment", "service_account", "grant", "bucket"} {
			if scope.AllowsKind(kind) {
				kinds = append(kinds, kind)
			}
		}
		send(w, 200, api.Capabilities{Scope: scope, Version: api.APIVersion, PrincipalID: p.ID, Resources: kinds, ReceiptPolicy: "retain-or-tombstone"})
		return
	}
	if r.Method == "GET" && len(parts) == 3 && parts[0] == "management" && api.Collection(parts[1]) != "" && api.ValidID(parts[2]) {
		if !s.pause(r, "before_dispatch", api.Operation{}) {
			return
		}
		p = s.currentPrincipal(p.ID, p.OrganizationID)
		if !s.can(p, s.receiptPath(scope, parts[1], parts[2]), "read") {
			fail(w, 403, "forbidden")
			return
		}
		mk := receiptKey(scope, parts[1], parts[2])
		if owner, ok := s.state.Expired[mk]; ok {
			if owner != p.ID {
				fail(w, 409, "ownership_conflict")
			} else {
				fail(w, 410, "receipt_expired")
			}
			return
		}
		m, ok := s.state.Management[mk]
		if !ok {
			fail(w, 404, "management_not_found")
			return
		}
		if m.OwnerID != p.ID {
			fail(w, 409, "ownership_conflict")
			return
		}
		send(w, 200, m)
		return
	}
	if r.Method == "GET" && len(parts) == 2 && parts[0] == "operations" {
		op, ok := s.state.Operations[parts[1]]
		if !ok || op.Scope != scope {
			fail(w, 404, "resource_not_found")
			return
		}
		if op.OwnerID != p.ID {
			fail(w, 403, "forbidden")
			return
		}
		send(w, 200, op)
		return
	}
	if len(parts) == 1 || len(parts) == 2 {
		kind := kindFor(parts[0])
		if kind != "" && scope.AllowsKind(kind) {
			if len(parts) == 1 && r.Method == "GET" {
				s.list(w, r, p, scope, kind)
				return
			}
			if r.URL.RawQuery != "" {
				fail(w, 400, "invalid_request")
				return
			}
			if len(parts) == 2 && r.Method == "GET" {
				if !api.ValidID(parts[1]) {
					fail(w, 400, "invalid_request")
					return
				}
				if kind == "bucket" && s.readError != "" {
					send(w, 503, map[string]string{"code": s.readError, "message": s.errorCanary})
					return
				}
				for _, op := range s.state.Operations {
					if op.Scope == scope && op.ResourceKind == kind && op.TargetID == parts[1] && op.Status == "accepted" {
						fail(w, 409, "operation_pending")
						return
					}
				}
				o, ok := s.state.Objects[objectPath(scope, kind, parts[1])]
				if !ok {
					fail(w, 404, "resource_not_found")
					return
				}
				send(w, 200, o)
				return
			}
			if (len(parts) == 1 && r.Method == "POST") || (len(parts) == 2 && (r.Method == "PATCH" || r.Method == "DELETE")) {
				s.mutate(w, r, p, scope, kind, parts)
				return
			}
		}
	}
	s.unexpected = append(s.unexpected, Request{r.Method, r.URL.Path})
	fail(w, 404, "unsupported_route")
}
