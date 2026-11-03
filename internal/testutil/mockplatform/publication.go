package mockplatform

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

// PublicationServer is a distinct, explicitly synthetic publication-v1 fixture.
// It never changes the full inventory mock-v5 or claims native/service evidence.
type PublicationServer struct {
	HTTP              *httptest.Server
	TokenFile, CAFile string
	Tokens            map[string]string
	releaseDenied     bool
	mu                sync.Mutex
	state             publicationLedger
	ledgerFile        string
	requests          []Request
	barriers          map[string]*Barrier
	denied            bool
}
type publicationIntent struct {
	Key          string `json:"operation_key"`
	Name         string `json:"name"`
	Protection   bool   `json:"deletion_protection"`
	Retention    int64  `json:"retention_days"`
	Revision     string `json:"expected_revision"`
	Environment  string `json:"environment_id"`
	Binding      string `json:"binding_id"`
	Method, Path string
}
type publicationLedger struct {
	Next, Revision            int
	Applications              int // Lifetime allocations, including deleted declarations.
	BindingCount, TargetCount int
	Bindings                  map[string]api.PublicationBinding
	Targets                   map[string]api.PublicationTarget
	Consumed                  map[string]bool
	Environments              map[string]*publicationLedger
	Apps                      map[string]api.PublicationApplication
	Ops                       map[string]api.PublicationOperation
	Intents                   map[string]publicationIntent
	Keys                      map[string]string
}

const PublicationBase = "/v1/publication/organizations/example/projects/saga"

func NewPublication(t Runtime) *PublicationServer {
	t.Helper()
	dir := t.TempDir()
	s := &PublicationServer{TokenFile: filepath.Join(dir, "publication.token"), CAFile: filepath.Join(dir, "publication.pem"), ledgerFile: filepath.Join(dir, "publication-ledger.json"), barriers: map[string]*Barrier{}, state: publicationLedger{Revision: 1, Apps: map[string]api.PublicationApplication{}, Ops: map[string]api.PublicationOperation{}, Intents: map[string]publicationIntent{}, Keys: map[string]string{}}}
	s.state.Bindings = map[string]api.PublicationBinding{}
	s.state.Environments = map[string]*publicationLedger{}
	for _, env := range []string{"nightly", "production"} {
		s.state.Environments[env] = &publicationLedger{Revision: 1, Targets: map[string]api.PublicationTarget{}, Consumed: map[string]bool{}, Ops: map[string]api.PublicationOperation{}, Intents: map[string]publicationIntent{}, Keys: map[string]string{}}
	}
	s.Tokens = map[string]string{"project": s.TokenFile}
	for role, token := range map[string]string{"nightly": "synthetic-nightly-token", "production": "synthetic-production-token", "reader": "synthetic-reader-token", "foreign": "synthetic-foreign-token"} {
		p := filepath.Join(dir, role+".token")
		if err := os.WriteFile(p, []byte(token), 0600); err != nil {
			t.Fatal(err)
		}
		s.Tokens[role] = p
	}
	s.persistPublication()
	s.HTTP = httptest.NewTLSServer(http.HandlerFunc(s.servePublication))
	for path, data := range map[string][]byte{s.TokenFile: []byte("synthetic-publication-token"), s.CAFile: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: s.HTTP.Certificate().Raw})} {
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(s.HTTP.Close)
	return s
}
func (s *PublicationServer) persistPublication() {
	data, err := json.Marshal(s.state)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(s.ledgerFile+".tmp", data, 0600); err != nil {
		panic(err)
	}
	if err := os.Rename(s.ledgerFile+".tmp", s.ledgerFile); err != nil {
		panic(err)
	}
}
func (s *PublicationServer) Restart() {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.ledgerFile)
	if err != nil {
		panic(err)
	}
	var retained publicationLedger
	if err := json.Unmarshal(data, &retained); err != nil {
		panic(err)
	}
	s.state = retained
	for id, op := range s.state.Ops {
		if op.Status == "accepted" {
			s.completePublication("", id)
		}
	}
	for environment, owner := range s.state.Environments {
		for id, op := range owner.Ops {
			if op.Status == "accepted" {
				s.completePublication(environment, id)
			}
		}
	}
}
func (s *PublicationServer) Requests() []Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Request(nil), s.requests...)
}
func (s *PublicationServer) Count() int { s.mu.Lock(); defer s.mu.Unlock(); return len(s.state.Apps) }
func (s *PublicationServer) OperationCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.state.Ops)
}
func (s *PublicationServer) Deny(value bool) { s.mu.Lock(); defer s.mu.Unlock(); s.denied = value }
func (s *PublicationServer) Drift(id, name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a := s.state.Apps[id]
	a.Name = name
	n, _ := strconv.Atoi(a.Revision)
	a.Revision = strconv.Itoa(n + 1)
	s.state.Apps[id] = a
	s.state.Revision++
	s.persistPublication()
}
func (s *PublicationServer) Arm(kind, phase string) *Barrier {
	s.mu.Lock()
	defer s.mu.Unlock()
	b := &Barrier{Reached: make(chan api.Operation, 1), Release: make(chan struct{})}
	s.barriers[kind+"/"+phase] = b
	return b
}
func (s *PublicationServer) publicationBarrier(kind, phase string, op api.PublicationOperation) {
	if b := s.barriers[kind+"/"+phase]; b != nil {
		delete(s.barriers, kind+"/"+phase)
		b.Reached <- api.Operation{ID: op.ID}
		s.mu.Unlock()
		<-b.Release
		s.mu.Lock()
	}
}
func (s *PublicationServer) owner(environment string) *publicationLedger {
	if environment == "" {
		return &s.state
	}
	return s.state.Environments[environment]
}
func (s *PublicationServer) OperationCountIn(environment string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.owner(environment).Ops)
}
func (s *PublicationServer) DenyRelease(denied bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.releaseDenied = denied
}
func publicationIncrement(value string) string {
	n, _ := strconv.Atoi(value)
	return strconv.Itoa(n + 1)
}
func (s *PublicationServer) completePublication(environment, id string) {
	owner := s.owner(environment)
	op := owner.Ops[id]
	if op.Status != "accepted" {
		return
	}
	i := owner.Intents[id]
	policy := api.PublicationPolicy{Name: i.Name, DeletionProtection: i.Protection, RetentionDays: i.Retention}
	applied := func(resource string) { op.ResourceID = &resource; op.Status = "applied" }
	op.Status = "applied"
	switch op.Kind {
	case "application_create":
		if owner.Applications >= 8 {
			op.Status = "capacity"
			break
		}
		owner.Applications++
		owner.Apps[id] = api.PublicationApplication{ID: id, Revision: "1", PublicationPolicy: policy, Readiness: "pending", Reason: "dependency_unavailable"}
		applied(id)
	case "application_update", "application_delete":
		resource := strings.TrimPrefix(i.Path, "/applications/")
		v, ok := owner.Apps[resource]
		if !ok {
			op.Status = "not_found"
			break
		}
		if v.Revision != i.Revision {
			op.Status = "conflict"
			break
		}
		held := v.DeletionProtection
		for _, b := range owner.Bindings {
			if b.ApplicationID == resource {
				held = true
			}
		}
		if i.Method == "DELETE" && held {
			op.Status = "protected"
			break
		}
		if i.Method == "DELETE" {
			delete(owner.Apps, resource)
		} else {
			v.PublicationPolicy = policy
			v.Revision = publicationIncrement(v.Revision)
			owner.Apps[resource] = v
		}
		applied(resource)
	case "binding_create":
		application := strings.TrimSuffix(strings.TrimPrefix(i.Path, "/applications/"), "/target-bindings")
		app, ok := owner.Apps[application]
		if !ok {
			op.Status = "not_found"
			break
		}
		if app.Revision != i.Revision {
			op.Status = "conflict"
			break
		}
		if s.state.Environments[i.Environment] == nil {
			op.Status = "denied"
			break
		}
		if owner.BindingCount >= 16 {
			op.Status = "capacity"
			break
		}
		owner.BindingCount++
		owner.Bindings[id] = api.PublicationBinding{ID: id, Revision: "1", ApplicationID: application, EnvironmentID: i.Environment, State: "reserved"}
		applied(id)
	case "binding_delete":
		resource := strings.TrimPrefix(i.Path, "/target-bindings/")
		v, ok := owner.Bindings[resource]
		if !ok {
			op.Status = "not_found"
			break
		}
		if v.Revision != i.Revision {
			op.Status = "conflict"
			break
		}
		v.Revision = publicationIncrement(publicationIncrement(v.Revision))
		owner.Bindings[resource] = v
		op.ResourceID = &resource
		if v.TargetID != nil {
			if _, live := s.owner(v.EnvironmentID).Targets[*v.TargetID]; live {
				op.Status = "protected"
				break
			}
		}
		if s.releaseDenied {
			op.Status = "denied"
			break
		}
		// A cancellation fences late consumption even before a target existed.
		s.owner(v.EnvironmentID).Consumed[resource] = true
		delete(owner.Bindings, resource)
		applied(resource)
	case "target_create":
		if owner.Consumed[i.Binding] {
			op.Status = "conflict"
			break
		}
		if owner.TargetCount >= 8 {
			op.Status = "capacity"
			break
		}
		owner.TargetCount++
		owner.Consumed[i.Binding] = true
		op.ResourceID = &id
		binding, ok := s.state.Bindings[i.Binding]
		if !ok || binding.EnvironmentID != environment || binding.TargetID != nil {
			op.Status = "conflict"
			break
		}
		binding.TargetID = &id
		binding.State = "consumed"
		binding.Revision = publicationIncrement(binding.Revision)
		s.state.Bindings[i.Binding] = binding
		s.state.Revision++
		app := binding.ApplicationID
		owner.Targets[id] = api.PublicationTarget{PublicationApplication: api.PublicationApplication{ID: id, Revision: "1", PublicationPolicy: policy, Readiness: "pending", Reason: "dependency_unavailable"}, ApplicationID: &app, BindingID: i.Binding, State: "declared", DesiredGeneration: "0", ConfigurationDigest: op.Fingerprint}
		applied(id)
	case "target_update", "target_delete":
		resource := strings.TrimPrefix(i.Path, "/targets/")
		v, ok := owner.Targets[resource]
		if !ok {
			op.Status = "not_found"
			break
		}
		if v.Revision != i.Revision {
			op.Status = "conflict"
			break
		}
		if i.Method == "DELETE" && (v.DeletionProtection || v.BundleDigest != nil || v.ObservedBundleDigest != nil) {
			op.Status = "protected"
			break
		}
		applied(resource)
		if i.Method == "DELETE" {
			delete(owner.Targets, resource)
			if s.releaseDenied {
				op.Status = "denied"
			} else {
				delete(s.state.Bindings, v.BindingID)
				s.state.Revision++
			}
		} else {
			v.PublicationPolicy = policy
			v.Revision = publicationIncrement(v.Revision)
			v.ConfigurationDigest = op.Fingerprint
			owner.Targets[resource] = v
		}
	default:
		panic("unknown synthetic publication operation")
	}
	owner.Ops[id] = op
	owner.Revision++
	s.persistPublication()
}
func publicationCapabilities(base, principal string, environment bool) map[string]any {
	return map[string]any{"profile": api.PublicationProfile, "version": api.PublicationVersion, "scope": base, "principal_id": principal, "deployment_class": "operator_local", "receipt_policy": "retain-or-tombstone", "capabilities": map[string]bool{"applications": !environment, "target_bindings": !environment, "targets": environment, "operations": true, "history": true, "uploads": false, "sealing": false, "publications": false, "previews": false, "domains": false, "promotion": false, "peer_selection": false, "peer_observation": false}, "limits": map[string]int{"applications": 8, "targets": 8, "bindings": 16, "operations": 64, "page_size": 8, "body_bytes": 65536, "operation_key_bytes": 32, "environment_id_bytes": 63, "native_frame_bytes": 190, "peer_selections": 4, "observations_per_selection": 4}}
}
func (s *PublicationServer) servePublication(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, Request{r.Method, r.URL.RequestURI()})
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Spoke-API-Version", api.PublicationVersion)
	w.Header().Set("X-Spoke-Management-Profile", api.PublicationProfile)
	w.Header().Set("Cache-Control", "no-store")
	write := func(status int, value any) { w.WriteHeader(status); _ = json.NewEncoder(w).Encode(value) }
	fail := func(status int, code string) { write(status, map[string]string{"code": code}) }
	principal := map[string]string{"Bearer synthetic-publication-token": "201", "Bearer synthetic-nightly-token": "202", "Bearer synthetic-production-token": "203", "Bearer synthetic-reader-token": "205", "Bearer synthetic-foreign-token": "206"}[r.Header.Get("Authorization")]
	if principal == "" {
		fail(401, "unauthenticated")
		return
	}
	if s.denied || principal == "206" || r.Header.Get("X-Spoke-Organization") != "example" || !strings.HasPrefix(r.URL.Path, PublicationBase+"/") {
		fail(403, "forbidden")
		return
	}
	if r.Header.Get("X-Spoke-API-Version") != api.PublicationVersion {
		fail(400, "incompatible_api")
		return
	}
	environment, base, path := "", PublicationBase, strings.TrimPrefix(r.URL.Path, PublicationBase)
	if strings.HasPrefix(path, "/environments/") {
		rest := strings.TrimPrefix(path, "/environments/")
		var ok bool
		environment, path, ok = strings.Cut(rest, "/")
		path = "/" + path
		if !ok || s.owner(environment) == nil || (environment == "nightly" && principal != "202" && principal != "205") || (environment == "production" && principal != "203") {
			fail(403, "forbidden")
			return
		}
		base += "/environments/" + environment
	}
	owner := s.owner(environment)
	if r.Method != "GET" && ((environment == "" && principal != "201") || principal == "205") {
		fail(403, "forbidden")
		return
	}
	if r.Method == "GET" {
		switch {
		case path == "/capabilities":
			write(200, publicationCapabilities(base, principal, environment != ""))
			return
		case strings.HasPrefix(path, "/operation-keys/") || strings.HasPrefix(path, "/operations/"):
			id := strings.TrimPrefix(path, "/operations/")
			if strings.HasPrefix(path, "/operation-keys/") {
				id = owner.Keys[strings.TrimPrefix(path, "/operation-keys/")]
			}
			op, ok := owner.Ops[id]
			if !ok {
				fail(404, "resource_not_found")
				return
			}
			if op.PrincipalID != principal {
				fail(403, "forbidden")
				return
			}
			write(200, op)
			return
		case strings.HasPrefix(path, "/target-bindings/") && environment == "":
			v, ok := owner.Bindings[strings.TrimPrefix(path, "/target-bindings/")]
			if !ok {
				fail(404, "resource_not_found")
				return
			}
			write(200, v)
			return
		case strings.HasPrefix(path, "/targets/") && environment != "":
			v, ok := owner.Targets[strings.TrimPrefix(path, "/targets/")]
			if !ok {
				fail(404, "resource_not_found")
				return
			}
			write(200, v)
			return
		case strings.HasPrefix(path, "/applications/") && !strings.HasSuffix(path, "/target-bindings") && environment == "":
			v, ok := owner.Apps[strings.TrimPrefix(path, "/applications/")]
			if !ok {
				fail(404, "resource_not_found")
				return
			}
			write(200, v)
			return
		}
		entries := map[string]any{}
		switch {
		case path == "/operations":
			for id, op := range owner.Ops {
				filter := r.URL.Query().Get("state")
				pending := op.Status == "accepted" || op.Status == "unknown"
				if op.PrincipalID != principal || filter == "pending" && !pending || filter == "terminal" && pending {
					continue
				}
				entries[id] = op
			}
		case path == "/applications" && environment == "":
			for id, v := range owner.Apps {
				entries[id] = v
			}
		case path == "/targets" && environment != "":
			for id, v := range owner.Targets {
				entries[id] = v
			}
		case strings.HasPrefix(path, "/applications/") && strings.HasSuffix(path, "/target-bindings") && environment == "":
			app := strings.TrimSuffix(strings.TrimPrefix(path, "/applications/"), "/target-bindings")
			if _, ok := owner.Apps[app]; !ok {
				fail(404, "resource_not_found")
				return
			}
			for id, v := range owner.Bindings {
				if v.ApplicationID == app {
					entries[id] = v
				}
			}
		default:
			fail(404, "unsupported_route")
			return
		}
		ids := []int{}
		for id := range entries {
			n, _ := strconv.Atoi(id)
			ids = append(ids, n)
		}
		sort.Ints(ids)
		items := []any{}
		for _, id := range ids {
			items = append(items, entries[strconv.Itoa(id)])
		}
		start := 0
		if cursor := r.URL.Query().Get("cursor"); cursor != "" {
			var revision int
			if _, err := fmt.Sscanf(cursor, "%d.%d", &revision, &start); err != nil || start < 0 || start >= len(items) {
				fail(422, "invalid_request")
				return
			}
			if revision != owner.Revision {
				fail(409, "stale_snapshot")
				return
			}
		}
		end := min(start+8, len(items))
		var next *string
		if end < len(items) {
			v := fmt.Sprintf("%d.%d", owner.Revision, end)
			next = &v
		}
		write(200, map[string]any{"snapshot": strconv.Itoa(owner.Revision), "next_cursor": next, "complete": next == nil, "items": items[start:end]})
		return
	}
	kind := ""
	if environment == "" {
		if r.Method == "POST" && strings.HasSuffix(path, "/target-bindings") || r.Method == "DELETE" && strings.HasPrefix(path, "/target-bindings/") {
			kind = "binding"
		} else if r.Method == "POST" && path == "/applications" || (r.Method == "PATCH" || r.Method == "DELETE") && strings.HasPrefix(path, "/applications/") {
			kind = "application"
		}
	} else if r.Method == "POST" && path == "/targets" || (r.Method == "PATCH" || r.Method == "DELETE") && strings.HasPrefix(path, "/targets/") {
		kind = "target"
	}
	if kind == "" {
		fail(404, "unsupported_route")
		return
	}
	var intent publicationIntent
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 65536))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&intent) != nil || intent.Key == "" {
		fail(422, "invalid_request")
		return
	}
	intent.Method, intent.Path = r.Method, path
	action := map[string]string{"POST": "create", "PATCH": "update", "DELETE": "delete"}[r.Method]
	for pass := 0; pass < 2; pass++ {
		if id, ok := owner.Keys[intent.Key]; ok {
			if owner.Intents[id] != intent {
				fail(409, "operation_conflict")
				return
			}
			if owner.Ops[id].PrincipalID != principal {
				fail(403, "forbidden")
				return
			}
			write(200, owner.Ops[id])
			return
		}
		if len(owner.Ops) >= 64 {
			fail(429, "capacity_exhausted")
			return
		}
		if pass == 0 {
			s.publicationBarrier(action, "before_accept", api.PublicationOperation{})
			owner = s.owner(environment)
		}
	}
	owner.Next++
	id := strconv.Itoa(owner.Next)
	encoded, _ := json.Marshal(intent)
	fingerprint := sha256.Sum256(encoded)
	op := api.PublicationOperation{ID: id, Key: intent.Key, Kind: kind + "_" + action, PrincipalID: principal, Fingerprint: hex.EncodeToString(fingerprint[:]), Status: "accepted", Location: base + "/operations/" + id, Generation: "0", EffectStatus: "none", ObservationRevision: "0", Readiness: "pending", Reason: "dependency_unavailable"}
	owner.Keys[intent.Key] = id
	owner.Ops[id] = op
	owner.Intents[id] = intent
	owner.Revision++
	s.persistPublication()
	s.publicationBarrier(action, "after_accept", op)
	s.completePublication(environment, id)
	owner = s.owner(environment)
	op = owner.Ops[id]
	s.publicationBarrier(action, "after_complete", op)
	w.Header().Set("Location", op.Location)
	write(202, op)
}

// ObserveTargetHead is test-only controller instrumentation, not an exposed
// publication API route or evidence of native/hosted selection support.
func (s *PublicationServer) ObserveTargetHead(environment, id, bundle string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	owner := s.owner(environment)
	v := owner.Targets[id]
	if bundle == "" {
		v.BundleDigest = nil
		v.ObservedBundleDigest = nil
		v.DesiredConfigurationDigest = nil
		v.ObservedConfigurationDigest = nil
		v.DesiredGeneration = "0"
		v.ObservedGeneration = nil
	} else {
		v.BundleDigest = &bundle
		v.ObservedBundleDigest = &bundle
		configuration := v.ConfigurationDigest
		v.DesiredConfigurationDigest = &configuration
		v.ObservedConfigurationDigest = &configuration
		v.DesiredGeneration = "1"
		generation := "1"
		v.ObservedGeneration = &generation
	}
	owner.Targets[id] = v
	owner.Revision++
	s.persistPublication()
}
