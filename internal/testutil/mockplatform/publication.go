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
	Method, Path string
}
type publicationLedger struct {
	Next, Revision int
	Applications   int // Lifetime allocations, including deleted declarations.
	Apps           map[string]api.PublicationApplication
	Ops            map[string]api.PublicationOperation
	Intents        map[string]publicationIntent
	Keys           map[string]string
}

const PublicationBase = "/v1/publication/organizations/example/projects/saga"

func NewPublication(t Runtime) *PublicationServer {
	t.Helper()
	dir := t.TempDir()
	s := &PublicationServer{TokenFile: filepath.Join(dir, "publication.token"), CAFile: filepath.Join(dir, "publication.pem"), ledgerFile: filepath.Join(dir, "publication-ledger.json"), barriers: map[string]*Barrier{}, state: publicationLedger{Revision: 1, Apps: map[string]api.PublicationApplication{}, Ops: map[string]api.PublicationOperation{}, Intents: map[string]publicationIntent{}, Keys: map[string]string{}}}
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
	if err := json.Unmarshal(data, &s.state); err != nil {
		panic(err)
	}
	for id, op := range s.state.Ops {
		if op.Status == "accepted" {
			s.completePublication(id)
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
func (s *PublicationServer) completePublication(id string) {
	op := s.state.Ops[id]
	if op.Status != "accepted" {
		return
	}
	i := s.state.Intents[id]
	resource := strings.TrimPrefix(i.Path, "/applications/")
	if i.Method == "POST" {
		resource = id
		if s.state.Applications >= 8 {
			op.Status = "capacity"
		} else {
			s.state.Applications++
			s.state.Apps[id] = api.PublicationApplication{ID: id, Revision: "1", PublicationPolicy: api.PublicationPolicy{Name: i.Name, DeletionProtection: i.Protection, RetentionDays: i.Retention}, Readiness: "pending", Reason: "dependency_unavailable"}
			op.ResourceID = &resource
			op.Status = "applied"
		}
	} else if a, ok := s.state.Apps[resource]; !ok {
		op.Status = "not_found"
	} else if a.Revision != i.Revision {
		op.Status = "conflict"
	} else if i.Method == "DELETE" && a.DeletionProtection {
		op.Status = "protected"
	} else {
		op.ResourceID = &resource
		op.Status = "applied"
		if i.Method == "DELETE" {
			delete(s.state.Apps, resource)
		} else {
			a.Name = i.Name
			a.DeletionProtection = i.Protection
			a.RetentionDays = i.Retention
			n, _ := strconv.Atoi(a.Revision)
			a.Revision = strconv.Itoa(n + 1)
			s.state.Apps[resource] = a
		}
	}
	s.state.Ops[id] = op
	s.state.Revision++
	s.persistPublication()
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
	if s.denied {
		fail(403, "forbidden")
		return
	}
	if r.Header.Get("Authorization") != "Bearer synthetic-publication-token" {
		fail(401, "unauthenticated")
		return
	}
	if r.Header.Get("X-Spoke-Organization") != "example" || !strings.HasPrefix(r.URL.Path, PublicationBase+"/") {
		fail(403, "forbidden")
		return
	}
	if r.Header.Get("X-Spoke-API-Version") != api.PublicationVersion {
		fail(400, "incompatible_api")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, PublicationBase)
	if r.Method == "GET" {
		switch {
		case path == "/capabilities":
			write(200, map[string]any{"profile": api.PublicationProfile, "version": api.PublicationVersion, "scope": PublicationBase, "principal_id": "201", "deployment_class": "operator_local", "receipt_policy": "retain-or-tombstone", "capabilities": map[string]bool{"applications": true, "target_bindings": true, "targets": false, "operations": true, "history": true, "uploads": false, "sealing": false, "publications": false, "previews": false, "domains": false, "promotion": false, "peer_selection": false, "peer_observation": false}, "limits": map[string]int{"applications": 8, "targets": 8, "bindings": 16, "operations": 64, "page_size": 8, "body_bytes": 65536, "operation_key_bytes": 32, "environment_id_bytes": 63, "native_frame_bytes": 190, "peer_selections": 4, "observations_per_selection": 4}})
		case strings.HasPrefix(path, "/operation-keys/"):
			id, ok := s.state.Keys[strings.TrimPrefix(path, "/operation-keys/")]
			if !ok {
				fail(404, "resource_not_found")
				return
			}
			write(200, s.state.Ops[id])
		case strings.HasPrefix(path, "/operations/"):
			op, ok := s.state.Ops[strings.TrimPrefix(path, "/operations/")]
			if !ok {
				fail(404, "resource_not_found")
				return
			}
			write(200, op)
		case strings.HasPrefix(path, "/applications/"):
			a, ok := s.state.Apps[strings.TrimPrefix(path, "/applications/")]
			if !ok {
				fail(404, "resource_not_found")
				return
			}
			write(200, a)
		case path == "/operations" || path == "/applications":
			ids := []int{}
			if path == "/operations" {
				for id := range s.state.Ops {
					n, _ := strconv.Atoi(id)
					ids = append(ids, n)
				}
			} else {
				for id := range s.state.Apps {
					n, _ := strconv.Atoi(id)
					ids = append(ids, n)
				}
			}
			sort.Ints(ids)
			items := []any{}
			for _, n := range ids {
				id := strconv.Itoa(n)
				if path == "/operations" {
					op := s.state.Ops[id]
					filter := r.URL.Query().Get("state")
					if filter == "pending" && op.Status != "accepted" || filter == "terminal" && op.Status == "accepted" {
						continue
					}
					items = append(items, op)
				} else {
					items = append(items, s.state.Apps[id])
				}
			}
			start := 0
			if cursor := r.URL.Query().Get("cursor"); cursor != "" {
				var revision int
				if _, err := fmt.Sscanf(cursor, "%d.%d", &revision, &start); err != nil || start < 0 || start >= len(items) {
					fail(422, "invalid_request")
					return
				}
				if revision != s.state.Revision {
					fail(409, "stale_snapshot")
					return
				}
			}
			end := min(start+8, len(items))
			var next *string
			if end < len(items) {
				v := fmt.Sprintf("%d.%d", s.state.Revision, end)
				next = &v
			}
			write(200, map[string]any{"snapshot": strconv.Itoa(s.state.Revision), "next_cursor": next, "complete": next == nil, "items": items[start:end]})
		default:
			fail(404, "unsupported_route")
		}
		return
	}
	if (r.Method != "POST" || path != "/applications") && ((r.Method != "PATCH" && r.Method != "DELETE") || !strings.HasPrefix(path, "/applications/")) {
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
	if id, ok := s.state.Keys[intent.Key]; ok {
		if s.state.Intents[id] != intent {
			fail(409, "operation_conflict")
			return
		}
		write(200, s.state.Ops[id])
		return
	}
	if len(s.state.Ops) >= 64 {
		fail(429, "capacity_exhausted")
		return
	}
	kind := map[string]string{"POST": "create", "PATCH": "update", "DELETE": "delete"}[r.Method]
	s.publicationBarrier(kind, "before_accept", api.PublicationOperation{})
	// A barrier releases the lock. Admission must still arbitrate concurrent
	// callers on the same retained slot when the paused request resumes.
	if id, ok := s.state.Keys[intent.Key]; ok {
		if s.state.Intents[id] != intent {
			fail(409, "operation_conflict")
			return
		}
		write(200, s.state.Ops[id])
		return
	}
	if len(s.state.Ops) >= 64 {
		fail(429, "capacity_exhausted")
		return
	}
	s.state.Next++
	id := strconv.Itoa(s.state.Next)
	encoded, _ := json.Marshal(intent)
	fingerprint := sha256.Sum256(encoded)
	op := api.PublicationOperation{ID: id, Key: intent.Key, Kind: "application_" + kind, PrincipalID: "201", Fingerprint: hex.EncodeToString(fingerprint[:]), Status: "accepted", Location: PublicationBase + "/operations/" + id, Generation: "0", EffectStatus: "none", ObservationRevision: "0", Readiness: "pending", Reason: "dependency_unavailable"}
	s.state.Keys[intent.Key] = id
	s.state.Ops[id] = op
	s.state.Intents[id] = intent
	s.state.Revision++
	s.persistPublication()
	s.publicationBarrier(kind, "after_accept", op)
	s.completePublication(id)
	op = s.state.Ops[id]
	s.publicationBarrier(kind, "after_complete", op)
	w.Header().Set("Location", op.Location)
	write(202, op)
}
