package platformclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

var publicationTestScope = Scope{OrganizationID: "example", ProjectID: "saga"}

const publicationTestBase = "/v1/publication/organizations/example/projects/saga"

func publicationTestCaps(scope Scope) map[string]any {
	base := publicationTestBase
	if scope.EnvironmentID != "" {
		base += "/environments/" + scope.EnvironmentID
	}
	return map[string]any{"profile": PublicationProfile, "version": PublicationVersion, "scope": base, "principal_id": "201", "deployment_class": "operator_local", "receipt_policy": "retain-or-tombstone",
		"capabilities": map[string]bool{"applications": scope.EnvironmentID == "", "target_bindings": scope.EnvironmentID == "", "targets": scope.EnvironmentID != "", "operations": true, "history": true, "uploads": false, "sealing": false, "publications": false, "previews": false, "domains": false, "promotion": false, "peer_selection": false, "peer_observation": false},
		"limits":       map[string]int{"applications": 8, "targets": 8, "bindings": 16, "operations": 64, "page_size": 8, "body_bytes": 65536, "operation_key_bytes": 32, "environment_id_bytes": 63, "native_frame_bytes": 190, "peer_selections": 4, "observations_per_selection": 4}}
}

func publicationTestApplication(id string) map[string]any {
	return map[string]any{"id": id, "revision": "1", "name": "application", "deletion_protection": true, "retention_days": 30, "readiness": "pending", "reason": "dependency_unavailable"}
}

func publicationTestOperation(base, id, key, kind, status string) map[string]any {
	return map[string]any{"id": id, "operation_key": key, "kind": kind, "principal_id": "201", "fingerprint": strings.Repeat("a", 64), "status": status, "resource_id": "1", "location": base + "/operations/" + id, "generation": "0", "bundle_digest": nil, "configuration_digest": nil, "effect_status": "none", "observation_revision": "0", "last_observation_ms": nil, "readiness": "pending", "reason": "dependency_unavailable", "expires_at_ms": nil}
}

func publicationTestWrite(w http.ResponseWriter, status int, value any) {
	if status == 202 {
		if object, ok := value.(map[string]any); ok {
			w.Header().Set("Location", object["location"].(string))
		}
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func publicationTestClient(t *testing.T, scope Scope, handler http.HandlerFunc) *PublicationClient {
	t.Helper()
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Spoke-API-Version", PublicationVersion)
		w.Header().Set("X-Spoke-Management-Profile", PublicationProfile)
		w.Header().Set("Cache-Control", "no-store")
		if r.Header.Get("Authorization") != "Bearer publication-canary" || r.Header.Get("X-Spoke-Organization") != scope.OrganizationID || r.Header.Get("X-Spoke-API-Version") != PublicationVersion {
			t.Error("incorrect publication authentication or version")
		}
		if strings.HasSuffix(r.URL.Path, "/capabilities") {
			publicationTestWrite(w, 200, publicationTestCaps(scope))
			return
		}
		handler(w, r)
	}))
	t.Cleanup(s.Close)
	c, err := NewPublication(Config{Endpoint: s.URL, Scope: scope, Token: "publication-canary", HTTPClient: s.Client(), PollInterval: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestPublicationSchemaRejectsIncompleteAmbiguousAndUnboundedJSON(t *testing.T) {
	valid, _ := json.Marshal(publicationTestApplication("9007199254740991"))
	if err := decodePublication(valid, "Application", new(PublicationApplication)); err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(map[string]any){
		"missing-false-protection": func(v map[string]any) { delete(v, "deletion_protection") },
		"null-protection":          func(v map[string]any) { v["deletion_protection"] = nil },
		"too-large-id":             func(v map[string]any) { v["id"] = "9007199254740992" },
		"numeric-id":               func(v map[string]any) { v["id"] = 1 },
		"leading-zero":             func(v map[string]any) { v["id"] = "01" },
		"unknown-field":            func(v map[string]any) { v["secret"] = "publication-canary" },
		"readiness":                func(v map[string]any) { v["readiness"] = "ready" },
		"retention":                func(v map[string]any) { v["retention_days"] = 36501 },
	} {
		t.Run(name, func(t *testing.T) {
			v := publicationTestApplication("1")
			change(v)
			raw, _ := json.Marshal(v)
			if err := decodePublication(raw, "Application", nil); err == nil || strings.Contains(err.Error(), "publication-canary") {
				t.Fatal("unsafe JSON accepted or disclosed")
			}
		})
	}
	for _, raw := range []string{string(valid) + " {}", strings.Replace(string(valid), `"revision":"1"`, `"revision":"1","revision":"2"`, 1), strings.Repeat("[", 30) + "0" + strings.Repeat("]", 30)} {
		if decodePublication([]byte(raw), "Application", nil) == nil {
			t.Fatal("ambiguous/excessive JSON accepted")
		}
	}
}

func TestPublicationReadsAreObservationalAndScoped(t *testing.T) {
	var writes atomic.Int32
	c := publicationTestClient(t, publicationTestScope, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			writes.Add(1)
		}
		switch r.URL.Path {
		case publicationTestBase + "/applications/1":
			publicationTestWrite(w, 200, publicationTestApplication("1"))
		case publicationTestBase + "/operation-keys/original", publicationTestBase + "/operations/7":
			publicationTestWrite(w, 200, publicationTestOperation(publicationTestBase, "7", "original", "application_create", "applied"))
		default:
			publicationTestWrite(w, 403, map[string]string{"code": "forbidden"})
		}
	})
	ctx := context.Background()
	if _, err := c.Application(ctx, "1"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Operation(ctx, "7"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Lookup(ctx, "original"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Application(ctx, "2"); err == nil {
		t.Fatal("denial became absence")
	}
	if _, err := c.Application(ctx, "../1"); err == nil {
		t.Fatal("unsafe ID accepted")
	}
	if _, err := c.Target(ctx, "1"); err == nil {
		t.Fatal("environment route in project scope")
	}
	if writes.Load() != 0 {
		t.Fatal("read submitted work")
	}
}

func TestPublicationHistoryRequiresOneCompleteSnapshot(t *testing.T) {
	for _, variant := range []string{"complete", "changed", "duplicate", "foreign", "wrong-kind", "loop", "missing"} {
		t.Run(variant, func(t *testing.T) {
			calls := 0
			c := publicationTestClient(t, publicationTestScope, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != "GET" || r.URL.Query().Get("state") != "all" {
					t.Error("unexpected history request")
				}
				page := map[string]any{"snapshot": "10", "complete": false, "next_cursor": "10.8.signature", "items": []any{publicationTestOperation(publicationTestBase, "7", "first", "application_create", "applied")}}
				if calls > 1 {
					if r.URL.Query().Get("cursor") != "10.8.signature" {
						t.Error("cursor lost")
					}
					page["complete"], page["next_cursor"], page["items"] = true, nil, []any{map[string]string{"id": "8", "status": "expired", "code": "receipt_expired"}}
					switch variant {
					case "changed":
						page["snapshot"] = "11"
					case "duplicate":
						page["items"] = []any{map[string]string{"id": "7", "status": "expired", "code": "receipt_expired"}}
					case "foreign":
						page["items"] = []any{publicationTestOperation("/foreign", "8", "next", "application_create", "applied")}
					case "wrong-kind":
						page["items"] = []any{publicationTestApplication("8")}
					case "loop":
						page["complete"], page["next_cursor"] = false, "10.8.signature"
					case "missing":
						delete(page, "items")
					}
				}
				publicationTestWrite(w, 200, page)
			})
			values, err := c.History(context.Background(), "all")
			if variant == "complete" {
				if err != nil || len(values) != 2 || values[1].Status != "expired" {
					t.Fatalf("%v %v", values, err)
				}
			} else if err == nil || values != nil {
				t.Fatal("incomplete history returned as complete")
			}
		})
	}
}

func TestPublicationUncertainAcceptanceRecoversExactHTTPIntent(t *testing.T) {
	methods := []string{}
	posts := 0
	logicalWrites := 0
	var original []byte
	c := publicationTestClient(t, publicationTestScope, func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method+" "+r.URL.Path)
		if r.Method == "POST" {
			body, _ := io.ReadAll(r.Body)
			posts++
			if posts == 1 {
				original = body
				logicalWrites++
				publicationTestWrite(w, 503, map[string]string{"code": "outcome_unknown", "operation_key": "original", "lookup": publicationTestBase + "/operation-keys/original"})
				return
			}
			if string(body) != string(original) {
				t.Error("recovery changed original request")
			}
			publicationTestWrite(w, 200, publicationTestOperation(publicationTestBase, "7", "original", "application_create", "accepted"))
			return
		}
		if posts == 0 {
			publicationTestWrite(w, 404, map[string]string{"code": "resource_not_found"})
			return
		}
		status := "accepted"
		if strings.Contains(r.URL.Path, "/operations/") {
			status = "applied"
		}
		publicationTestWrite(w, 200, publicationTestOperation(publicationTestBase, "7", "original", "application_create", status))
	})
	op, err := c.CreateApplication(context.Background(), "original", PublicationPolicy{"application", true, 30})
	if err != nil || op.ID != "7" || posts != 2 || logicalWrites != 1 {
		t.Fatalf("%+v %v posts=%d writes=%d", op, err, posts, logicalWrites)
	}
	want := []string{"GET " + publicationTestBase + "/operation-keys/original", "POST " + publicationTestBase + "/applications", "GET " + publicationTestBase + "/operation-keys/original", "POST " + publicationTestBase + "/applications", "GET " + publicationTestBase + "/operations/7"}
	if !reflect.DeepEqual(methods, want) {
		t.Fatalf("unsafe recovery sequence: %v", methods)
	}
}

func TestPublicationRecoveryRefusalsPreserveOriginalNamespace(t *testing.T) {
	for _, variant := range []string{"expired", "incomplete", "other-owner", "different-kind", "conflicting-intent", "changed-receipt", "terminal-denial"} {
		t.Run(variant, func(t *testing.T) {
			var posts atomic.Int32
			c := publicationTestClient(t, publicationTestScope, func(w http.ResponseWriter, r *http.Request) {
				op := publicationTestOperation(publicationTestBase, "7", "original", "application_create", "accepted")
				if r.Method == "POST" {
					posts.Add(1)
					if variant == "conflicting-intent" {
						publicationTestWrite(w, 409, map[string]string{"code": "operation_conflict"})
						return
					}
					if variant == "changed-receipt" {
						op["id"], op["location"] = "8", publicationTestBase+"/operations/8"
					}
					publicationTestWrite(w, 200, op)
					return
				}
				switch variant {
				case "expired":
					publicationTestWrite(w, 410, map[string]string{"code": "receipt_expired"})
					return
				case "incomplete":
					publicationTestWrite(w, 503, map[string]string{"code": "incomplete_result"})
					return
				case "other-owner":
					op["principal_id"] = "202"
				case "different-kind":
					op["kind"] = "target_create"
				case "terminal-denial":
					if posts.Load() > 0 {
						op["status"] = "denied"
					}
				}
				publicationTestWrite(w, 200, op)
			})
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
			defer cancel()
			op, err := c.CreateApplication(ctx, "original", PublicationPolicy{"application", true, 30})
			if err == nil {
				t.Fatal("unsafe original receipt accepted")
			}
			if variant == "expired" || variant == "incomplete" || variant == "other-owner" || variant == "different-kind" {
				if posts.Load() != 0 {
					t.Fatal("unverified observation caused a write")
				}
			}
			if variant == "terminal-denial" && op.Status != "denied" {
				t.Fatal("lost terminal refusal evidence")
			}
		})
	}
}

func TestPublicationDeclaredMutationRoutes(t *testing.T) {
	policy := PublicationPolicy{"site", false, 7}
	for _, kind := range []string{"application_create", "application_update", "application_delete", "binding_create", "binding_delete", "target_create", "target_update", "target_delete"} {
		t.Run(kind, func(t *testing.T) {
			scope := publicationTestScope
			base := publicationTestBase
			if strings.HasPrefix(kind, "target_") {
				scope.EnvironmentID = "nightly"
				base += "/environments/nightly"
			}
			c := publicationTestClient(t, scope, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "GET" {
					publicationTestWrite(w, 404, map[string]string{"code": "resource_not_found"})
					return
				}
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				if body["operation_key"] != "request" {
					t.Error("missing mutation identity")
				}
				if _, ok := body["selected_publication"]; ok {
					t.Error("policy mutated runtime head")
				}
				method, path := "POST", "/applications"
				switch kind {
				case "application_update":
					method, path = "PATCH", "/applications/1"
				case "application_delete":
					method, path = "DELETE", "/applications/1"
				case "binding_create":
					path = "/applications/1/target-bindings"
					if body["environment_id"] != "nightly" {
						t.Error("missing binding environment")
					}
				case "binding_delete":
					method, path = "DELETE", "/target-bindings/1"
				case "target_create":
					path = "/targets"
					if body["binding_id"] != "2" {
						t.Error("missing project binding")
					}
				case "target_update":
					method, path = "PATCH", "/targets/1"
				case "target_delete":
					method, path = "DELETE", "/targets/1"
				}
				if r.Method != method || r.URL.Path != base+path {
					t.Errorf("wrong route %s %s", r.Method, r.URL.Path)
				}
				if strings.HasSuffix(kind, "update") || strings.HasSuffix(kind, "delete") || kind == "binding_create" {
					if body["expected_revision"] != "2" {
						t.Error("revision was not a decimal string")
					}
				}
				publicationTestWrite(w, 202, publicationTestOperation(base, "7", "request", kind, "applied"))
			})
			ctx := context.Background()
			var err error
			switch kind {
			case "application_create":
				_, err = c.CreateApplication(ctx, "request", policy)
			case "application_update":
				_, err = c.UpdateApplication(ctx, "1", "request", "2", policy)
			case "application_delete":
				_, err = c.DeleteApplication(ctx, "1", "request", "2")
			case "binding_create":
				_, err = c.CreateBinding(ctx, "1", "request", "2", "nightly")
			case "binding_delete":
				_, err = c.DeleteBinding(ctx, "1", "request", "2")
			case "target_create":
				_, err = c.CreateTarget(ctx, "2", "request", policy)
			case "target_update":
				_, err = c.UpdateTarget(ctx, "1", "request", "2", policy)
			case "target_delete":
				_, err = c.DeleteTarget(ctx, "1", "request", "2")
			}
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPublicationProfileAndErrorIsolation(t *testing.T) {
	for _, variant := range []string{"peer", "missing-profile", "duplicate-profile", "foreign-scope", "changed-principal", "unsafe-error", "redirect", "unknown-acceptance-location"} {
		t.Run(variant, func(t *testing.T) {
			requests := 0
			s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Spoke-API-Version", PublicationVersion)
				w.Header().Set("X-Spoke-Management-Profile", PublicationProfile)
				w.Header().Set("Cache-Control", "no-store")
				caps := publicationTestCaps(publicationTestScope)
				switch variant {
				case "peer":
					w.Header().Set("X-Spoke-Management-Profile", "spoke.platform.http.publication.peer.v1")
				case "missing-profile":
					w.Header().Del("X-Spoke-Management-Profile")
				case "duplicate-profile":
					w.Header().Add("X-Spoke-Management-Profile", PublicationProfile)
				case "foreign-scope":
					caps["scope"] = "/foreign"
				case "changed-principal":
					if requests > 1 {
						caps["principal_id"] = "202"
					}
				case "unsafe-error":
					publicationTestWrite(w, 503, map[string]string{"code": "publication-canary"})
					return
				case "redirect":
					w.Header().Set("Location", "https://example.invalid/leak")
					publicationTestWrite(w, 302, map[string]string{"code": "unavailable"})
					return
				case "unknown-acceptance-location":
					if strings.Contains(r.URL.Path, "operation-keys") {
						publicationTestWrite(w, 404, map[string]string{"code": "resource_not_found"})
						return
					}
					if r.Method == "POST" {
						publicationTestWrite(w, 503, map[string]string{"code": "outcome_unknown", "operation_key": "foreign", "lookup": "https://example.invalid"})
						return
					}
				}
				publicationTestWrite(w, 200, caps)
			}))
			defer s.Close()
			c, err := NewPublication(Config{Endpoint: s.URL, Scope: publicationTestScope, Token: "publication-canary", HTTPClient: s.Client()})
			if err != nil {
				t.Fatal(err)
			}
			if variant == "unknown-acceptance-location" {
				_, err = c.CreateApplication(context.Background(), "original", PublicationPolicy{"site", false, 7})
			} else {
				_, err = c.Capabilities(context.Background())
				if variant == "changed-principal" && err == nil {
					_, err = c.Capabilities(context.Background())
				}
			}
			if err == nil || strings.Contains(err.Error(), "publication-canary") {
				t.Fatalf("unsafe response: %v", err)
			}
		})
	}
}

func TestPublicationInvalidMutationsFailBeforeNetwork(t *testing.T) {
	var requests atomic.Int32
	c := publicationTestClient(t, publicationTestScope, func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		t.Error("invalid mutation reached network")
	})
	for i, policy := range []PublicationPolicy{{"bad/name", false, 1}, {"site", false, -1}, {"site", false, 36501}, {strings.Repeat("x", 64), false, 1}} {
		if _, err := c.CreateApplication(context.Background(), fmt.Sprint(i), policy); err == nil {
			t.Fatal("invalid policy accepted")
		}
	}
	if _, err := c.UpdateApplication(context.Background(), "1", "request", "9007199254740992", PublicationPolicy{"site", false, 1}); err == nil {
		t.Fatal("oversized fence accepted")
	}
	if requests.Load() != 0 {
		t.Fatal("invalid mutation sent")
	}
	var e *PublicationError
	if errors.As(&PublicationOutcomeError{Status: "protected"}, &e) {
		t.Fatal("terminal refusal was misclassified as HTTP absence")
	}
}
