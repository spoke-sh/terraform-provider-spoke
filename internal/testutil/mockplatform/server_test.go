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
	r.Header.Set("X-Spoke-API-Version", "mock-v1")
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
	if resp.Header.Get("X-Spoke-API-Version") != "mock-v1" {
		t.Fatal("missing contract version")
	}
	return resp.StatusCode, b
}

func validate(t *testing.T, kind string, b []byte) {
	t.Helper()
	f, err := os.Open("../../../testdata/contracts/mock-v1.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	doc, err := jsonschema.UnmarshalJSON(f)
	if err != nil {
		t.Fatal(err)
	}
	c := jsonschema.NewCompiler()
	const uri = "https://spoke.sh/terraform-provider/mock-v1.schema.json"
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
	for path, kind := range map[string]string{"/v1/projects/project-test": "Project", s.Base(): "Environment"} {
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
	for path, kind := range map[string]string{s.Base() + "/operations/" + op.ID: "Operation", s.Base() + "/management/fixture-key": "Management", s.Base() + "/buckets/" + op.TargetID: "Bucket"} {
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
