package governor

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

const testBase = "/v1/ci/organizations/example/projects/operators/environments/internal"

var testTokens = map[string]Identity{"operator-canary": {"operator-one", true}, "other-canary": {"operator-two", true}, "reader-canary": {"operator-one", false}}

func testInputs(t *testing.T) ([]byte, object) {
	t.Helper()
	p := "../../../platformclient/governor-contract/"
	schema, err := os.ReadFile(p + "schema.json")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(p + "vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var vectors object
	if json.Unmarshal(raw, &vectors) != nil {
		t.Fatal("vectors")
	}
	return schema, vectors
}
func testPeer(t *testing.T) (*Server, *httptest.Server, []byte, object) {
	t.Helper()
	schema, vectors := testInputs(t)
	s, err := New(filepath.Join(t.TempDir(), "ledger.json"), testBase, schema, testTokens)
	if err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewTLSServer(s)
	t.Cleanup(httpServer.Close)
	return s, httpServer, schema, vectors
}
func wire(t *testing.T, h *httptest.Server, token, method, path string, body any) (int, object) {
	t.Helper()
	var raw []byte
	if body != nil {
		if b, ok := body.([]byte); ok {
			raw = b
		} else {
			raw, _ = json.Marshal(body)
		}
	}
	req, err := http.NewRequest(method, h.URL+testBase+path, bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Spoke-Organization", "example")
	req.Header.Set("X-Spoke-API-Version", Version)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)
	var out object
	if json.Unmarshal(data, &out) != nil {
		t.Fatal("non JSON response")
	}
	if res.Header.Get("X-Spoke-Management-Profile") != Profile || res.Header.Get("X-Spoke-API-Version") != Version || res.Header.Get("Cache-Control") != "no-store" {
		t.Fatal("response boundary")
	}
	if method != "GET" && res.StatusCode < 300 {
		if res.Header.Get("Location") != testBase+strings.TrimSuffix(strings.TrimSuffix(path, "/commit"), "/abort") {
			t.Fatal("wrong original location")
		}
	}
	return res.StatusCode, out
}
func expectCode(t *testing.T, status int, got object, want int, code string) {
	t.Helper()
	if status != want || got["code"] != code {
		t.Fatalf("wanted %d %s, got %d %v", want, code, status, got)
	}
}

func TestRawCanonicalPreparationIsDurableAndObservational(t *testing.T) {
	s, h, schema, vs := testPeer(t)
	v := vs["vectors"].([]any)[0].(object)
	request := v["request"].(object)
	intent := request["intent"].(object)
	status, op := wire(t, h, "operator-canary", "PUT", "/requests/internal-ci/0", intent)
	if status != 201 || op["status"] != "prepared" || op["fingerprint"] != v["fingerprint"] || !s.valid("Operation", op) {
		t.Fatal("independent canonical mismatch", status, op)
	}
	if len(s.state.Governors) != 0 {
		t.Fatal("prepare applied policy")
	}
	before := s.Snapshot()
	for _, path := range []string{"/capabilities", "/management/internal-ci", "/management?state=pending", "/requests/internal-ci/0", "/governors"} {
		for i := 0; i < 3; i++ {
			status, _ = wire(t, h, "operator-canary", "GET", path, nil)
			if status != 200 {
				t.Fatal(status)
			}
		}
	}
	if !bytes.Equal(before, s.Snapshot()) {
		t.Fatal("GET advanced state")
	}
	h.Close()
	reloaded, err := New(s.path, testBase, schema, testTokens)
	if err != nil {
		t.Fatal(err)
	}
	h = httptest.NewTLSServer(reloaded)
	defer h.Close()
	status, replay := wire(t, h, "operator-canary", "PUT", "/requests/internal-ci/0", intent)
	if status != 200 || !reflect.DeepEqual(op, replay) {
		t.Fatal("prepare did not survive service reload")
	}
	status, applied := wire(t, h, "operator-canary", "POST", "/requests/internal-ci/0/commit", object{"fingerprint": op["fingerprint"]})
	if status != 200 || applied["status"] != "applied" || applied["id"] != op["id"] {
		t.Fatal("commit", status, applied)
	}
	status, replay = wire(t, h, "operator-canary", "POST", "/requests/internal-ci/0/commit", object{"fingerprint": op["fingerprint"]})
	if status != 200 || !reflect.DeepEqual(applied, replay) || len(reloaded.state.Governors) != 1 {
		t.Fatal("duplicate commit")
	}
}

func TestRawScopeOwnershipSequenceAndStrictBodies(t *testing.T) {
	_, h, _, vs := testPeer(t)
	intent := clone(vs["vectors"].([]any)[0].(object)["request"].(object)["intent"].(object))
	status, op := wire(t, h, "operator-canary", "PUT", "/requests/stable/0", intent)
	if status != 201 {
		t.Fatal(status)
	}
	for _, path := range []string{"/requests/stable/0", "/management/stable"} {
		status, got := wire(t, h, "other-canary", "GET", path, nil)
		expectCode(t, status, got, 403, "forbidden")
	}
	status, got := wire(t, h, "reader-canary", "POST", "/requests/stable/0/commit", object{"fingerprint": op["fingerprint"]})
	expectCode(t, status, got, 403, "forbidden")
	status, got = wire(t, h, "operator-canary", "PUT", "/requests/stable/1", intent)
	expectCode(t, status, got, 409, "operation_pending")
	changed := clone(intent)
	changed["policy"].(object)["name"] = "changed"
	status, got = wire(t, h, "operator-canary", "PUT", "/requests/stable/0", changed)
	expectCode(t, status, got, 409, "intent_conflict")
	status, got = wire(t, h, "operator-canary", "POST", "/requests/stable/0/commit", object{"fingerprint": strings.Repeat("0", 64)})
	expectCode(t, status, got, 409, "intent_conflict")
	for _, body := range []any{object{"kind": "handoff"}, object{"kind": "create", "target_id": nil, "expected_revision": 0, "policy": nil}, []byte(`{"fingerprint":"a","fingerprint":"b"}`), []byte(strings.Repeat(" ", 16385))} {
		status, got = wire(t, h, "operator-canary", "PUT", "/requests/bad/0", body)
		if status != 422 && status != 413 {
			t.Fatal("invalid body", status, got)
		}
	}
	status, got = wire(t, h, "operator-canary", "GET", "/governors?unknown=1", nil)
	expectCode(t, status, got, 422, "invalid_request")
}

func TestRawConcurrentEntitlementsAbortAndExpiry(t *testing.T) {
	s, h, schema, vs := testPeer(t)
	intent := vs["vectors"].([]any)[0].(object)["request"].(object)["intent"]
	var wg sync.WaitGroup
	var mu sync.Mutex
	statuses := []int{}
	var winner string
	var op object
	for _, key := range []string{"first", "second"} {
		wg.Go(func() {
			status, got := wire(t, h, "operator-canary", "PUT", "/requests/"+key+"/0", intent)
			mu.Lock()
			defer mu.Unlock()
			statuses = append(statuses, status)
			if status == 201 {
				winner = key
				op = got
			}
		})
	}
	wg.Wait()
	if !(statuses[0] == 201 && statuses[1] == 409 || statuses[1] == 201 && statuses[0] == 409) {
		t.Fatal(statuses)
	}
	if err := s.Expire(winner, 0); err == nil {
		t.Fatal("expired pending receipt")
	}
	status, got := wire(t, h, "operator-canary", "POST", "/requests/"+winner+"/0/abort", object{"fingerprint": op["fingerprint"]})
	if status != 200 || got["status"] != "aborted" {
		t.Fatal(status, got)
	}
	if err := s.Expire(winner, 0); err != nil {
		t.Fatal(err)
	}
	h.Close()
	s, err := New(s.path, testBase, schema, testTokens)
	if err != nil {
		t.Fatal(err)
	}
	h = httptest.NewTLSServer(s)
	defer h.Close()
	for _, method := range []string{"GET", "PUT"} {
		status, got = wire(t, h, "operator-canary", method, "/requests/"+winner+"/0", map[string]any{"GET": nil, "PUT": intent}[method])
		expectCode(t, status, got, 410, "receipt_expired")
	}
	status, got = wire(t, h, "operator-canary", "PUT", "/requests/replacement/0", intent)
	if status != 201 {
		t.Fatal("abort retained entitlement reservation", got)
	}
}

func TestRawStaleRevisionDrainAndRetirementTombstone(t *testing.T) {
	s, h, _, vs := testPeer(t)
	vectors := vs["vectors"].([]any)
	intent := clone(vectors[0].(object)["request"].(object)["intent"].(object))
	_, op := wire(t, h, "operator-canary", "PUT", "/requests/stable/0", intent)
	_, op = wire(t, h, "operator-canary", "POST", "/requests/stable/0/commit", object{"fingerprint": op["fingerprint"]})
	id := op["resource_id"].(string)
	intent["kind"] = "update"
	intent["target_id"] = id
	intent["expected_revision"] = 2
	_, op = wire(t, h, "operator-canary", "PUT", "/requests/stable/1", intent)
	_, op = wire(t, h, "operator-canary", "POST", "/requests/stable/1/commit", object{"fingerprint": op["fingerprint"]})
	if op["status"] != "refused" || op["reason"] != "revision_conflict" {
		t.Fatal(op)
	}
	if err := s.SeedUsage(id, 2, 16, 1); err != nil {
		t.Fatal(err)
	}
	retire := object{"kind": "retire", "target_id": id, "expected_revision": 1, "policy": nil}
	_, op = wire(t, h, "operator-canary", "PUT", "/requests/stable/2", retire)
	status, op := wire(t, h, "operator-canary", "POST", "/requests/stable/2/commit", object{"fingerprint": op["fingerprint"]})
	if status != 202 || op["reason"] != "draining" {
		t.Fatal(status, op)
	}
	before := s.Snapshot()
	_, _ = wire(t, h, "operator-canary", "GET", "/requests/stable/2", nil)
	if !bytes.Equal(before, s.Snapshot()) {
		t.Fatal("read drained")
	}
	if err := s.SeedUsage(id, 0, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := s.Settle(); err != nil {
		t.Fatal(err)
	}
	status, got := wire(t, h, "operator-canary", "GET", "/governors/"+id, nil)
	expectCode(t, status, got, 404, "resource_not_found")
	status, got = wire(t, h, "operator-canary", "PUT", "/requests/stable/3", vectors[0].(object)["request"].(object)["intent"])
	expectCode(t, status, got, 409, "sequence_conflict")
	_, got = wire(t, h, "operator-canary", "GET", "/requests/stable/2", nil)
	if got["status"] != "applied" || got["resource_id"] != id {
		t.Fatal("retirement receipt lost")
	}
}

func TestRejectIncompatibleLedgerWithoutErasingIt(t *testing.T) {
	s, _, schema, _ := testPeer(t)
	raw := []byte(`{"profile":"old"}`)
	if os.WriteFile(s.path, raw, 0600) != nil {
		t.Fatal("write")
	}
	if _, err := New(s.path, testBase, schema, testTokens); err == nil {
		t.Fatal("accepted incompatible ledger")
	}
	after, _ := os.ReadFile(s.path)
	if !bytes.Equal(raw, after) {
		t.Fatal("incompatible evidence modified")
	}
}
