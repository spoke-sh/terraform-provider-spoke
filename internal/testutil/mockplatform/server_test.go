package mockplatform

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func raw(t *testing.T, s *Server, method, path, body string) (int, []byte) {
	t.Helper()
	r, err := http.NewRequest(method, s.HTTP.URL+path, bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Authorization", "Bearer "+s.Token)
	r.Header.Set("X-Spoke-Organization", s.Scope.OrganizationID)
	r.Header.Set("X-Spoke-API-Version", "mock-v3")
	r.Header.Set("Content-Type", "application/json")
	resp, err := s.HTTP.Client().Do(r)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Header.Get("X-Spoke-API-Version") != "mock-v3" {
		t.Fatal("missing contract version")
	}
	return resp.StatusCode, b
}

func validate(t *testing.T, kind string, b []byte) {
	t.Helper()
	f, err := os.Open("../../../testdata/contracts/mock-v3.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	doc, err := jsonschema.UnmarshalJSON(f)
	if err != nil {
		t.Fatal(err)
	}
	c := jsonschema.NewCompiler()
	const uri = "https://spoke.sh/terraform-provider/mock-v3.schema.json"
	if err := c.AddResource(uri, doc); err != nil {
		t.Fatal(err)
	}
	schema, err := c.Compile(uri + "#/$defs/" + kind)
	if err != nil {
		t.Fatal(err)
	}
	value, err := jsonschema.UnmarshalJSON(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	if err := schema.Validate(value); err != nil {
		t.Fatal(err)
	}
}

func TestRawContractAndCAS(t *testing.T) {
	s := New(t)
	for path, kind := range map[string]string{"/v3/organizations/org-test/projects/project-test": "Object", s.Base(): "Object"} {
		status, b := raw(t, s, "GET", path, "")
		if status != 200 {
			t.Fatalf("%d %s", status, b)
		}
		validate(t, kind, b)
	}
	const body = `{"kind":"create","management_key":"fixture-key","target_id":"","expected_revision":0,"expected_generation":0,"spec":{"name":"assets","deletion_protection":false}}`
	validate(t, "Mutation", []byte(body))
	status, b := raw(t, s, "POST", s.Base()+"/buckets", body)
	if status != 202 {
		t.Fatalf("%d %s", status, b)
	}
	validate(t, "Operation", b)
	var op struct {
		ID       string `json:"id"`
		TargetID string `json:"target_id"`
	}
	if err := json.Unmarshal(b, &op); err != nil {
		t.Fatal(err)
	}
	for path, kind := range map[string]string{s.Base() + "/operations/" + op.ID: "Operation", s.Base() + "/management/bucket/fixture-key": "Management", s.Base() + "/buckets/" + op.TargetID: "Object"} {
		status, b = raw(t, s, "GET", path, "")
		if status != 200 {
			t.Fatalf("%d %s", status, b)
		}
		validate(t, kind, b)
	}
	status, b = raw(t, s, "POST", s.Base()+"/buckets", body)
	if status != 202 || s.OperationCount() != 1 {
		t.Fatalf("replay: %d %s", status, b)
	}
	conflict := bytes.ReplaceAll([]byte(body), []byte("assets"), []byte("other"))
	status, b = raw(t, s, "POST", s.Base()+"/buckets", string(conflict))
	if status != 409 {
		t.Fatalf("CAS: %d %s", status, b)
	}
	validate(t, "Error", b)
	status, b = raw(t, s, "POST", s.Base()+"/buckets", `{"kind":"create","unexpected":true}`)
	if status != 400 {
		t.Fatalf("strict decoding: %d %s", status, b)
	}
}

func TestRawDiscoveryAndCursorIsolation(t *testing.T) {
	s := New(t)
	status, b := raw(t, s, "GET", s.Base()+"/capabilities", "")
	if status != 200 {
		t.Fatalf("capabilities: %d %s", status, b)
	}
	validate(t, "Capabilities", b)
	for _, key := range []string{"a", "b", "c"} {
		body := `{"kind":"create","management_key":"` + key + `","target_id":"","expected_revision":0,"expected_generation":0,"spec":{"name":"` + key + `","deletion_protection":false}}`
		status, b = raw(t, s, "POST", s.Base()+"/buckets", body)
		if status != 202 {
			t.Fatalf("create: %d %s", status, b)
		}
	}
	status, b = raw(t, s, "GET", s.Base()+"/buckets", "")
	if status != 200 {
		t.Fatal(status)
	}
	validate(t, "Page", b)
	var page struct {
		Next     string `json:"next_token"`
		Complete bool   `json:"complete"`
		Items    []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(b, &page); err != nil {
		t.Fatal(err)
	}
	if page.Next == "" || page.Complete || len(page.Items) != 2 {
		t.Fatalf("invalid first page: %s", b)
	}
	token := page.Next
	status, b = raw(t, s, "GET", s.Base()+"/buckets?page_token="+token, "")
	if status != 200 {
		t.Fatalf("next: %d %s", status, b)
	}
	validate(t, "Page", b)
	if err := json.Unmarshal(b, &page); err != nil {
		t.Fatal(err)
	}
	if !page.Complete || len(page.Items) != 1 {
		t.Fatalf("invalid final page: %s", b)
	}
	status, b = raw(t, s, "GET", s.Base()+"/buckets?page_token=forged", "")
	if status != 409 {
		t.Fatalf("forged cursor: %d %s", status, b)
	}
	s.Remove(page.Items[0].ID)
	status, b = raw(t, s, "GET", s.Base()+"/buckets?page_token="+token, "")
	if status != 409 {
		t.Fatalf("stale snapshot: %d %s", status, b)
	}
}

func TestRawScopeAndGrantShapes(t *testing.T) {
	s := New(t)
	root := "/v3/organizations/org-test"
	body := `{"kind":"create","resource_kind":"service_account","management_key":"account","target_id":"","expected_revision":0,"expected_generation":0,"spec":{"name":"worker","deletion_protection":false}}`
	validate(t, "Mutation", []byte(body))
	status, b := raw(t, s, "POST", root+"/service_accounts", body)
	if status != 202 {
		t.Fatalf("account: %d %s", status, b)
	}
	validate(t, "Operation", b)
	var receipt struct {
		TargetID string `json:"target_id"`
	}
	if err := json.Unmarshal(b, &receipt); err != nil {
		t.Fatal(err)
	}
	status, b = raw(t, s, "GET", root+"/service_accounts/"+receipt.TargetID, "")
	if status != 200 {
		t.Fatal(status)
	}
	validate(t, "Object", b)
	body = `{"kind":"create","resource_kind":"grant","management_key":"grant","target_id":"","expected_revision":0,"expected_generation":0,"spec":{"name":"read","deletion_protection":false,"subject_id":"` + receipt.TargetID + `","target":"/v3/organizations/org-test","permission":"read"}}`
	validate(t, "Mutation", []byte(body))
	status, b = raw(t, s, "POST", root+"/grants", body)
	if status != 202 {
		t.Fatalf("grant: %d %s", status, b)
	}
	validate(t, "Operation", b)
	if err := json.Unmarshal(b, &receipt); err != nil {
		t.Fatal(err)
	}
	status, b = raw(t, s, "GET", root+"/grants/"+receipt.TargetID, "")
	if status != 200 {
		t.Fatal(status)
	}
	validate(t, "Object", b)
	status, b = raw(t, s, "GET", root+"/grants", "")
	if status != 200 {
		t.Fatal(status)
	}
	validate(t, "Page", b)
}

func TestRawAuthenticationAndVersionGates(t *testing.T) {
	s := New(t)
	for _, tc := range []struct {
		header, value string
		status        int
	}{
		{"Authorization", "Bearer wrong", 401},
		{"X-Spoke-Organization", "other-organization", 403},
		{"X-Spoke-API-Version", "mock-v1", 400},
	} {
		request, err := http.NewRequest("GET", s.HTTP.URL+s.Base()+"/capabilities", nil)
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Authorization", "Bearer "+s.Token)
		request.Header.Set("X-Spoke-Organization", s.Scope.OrganizationID)
		request.Header.Set("X-Spoke-API-Version", "mock-v3")
		request.Header.Set(tc.header, tc.value)
		response, err := s.HTTP.Client().Do(request)
		if err != nil {
			t.Fatal(err)
		}
		b, err := io.ReadAll(response.Body)
		response.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != tc.status {
			t.Fatalf("%s: %d %s", tc.header, response.StatusCode, b)
		}
		validate(t, "Error", b)
	}
	if s.OperationCount() != 0 {
		t.Fatal("authentication/version discovery mutated state")
	}
}
