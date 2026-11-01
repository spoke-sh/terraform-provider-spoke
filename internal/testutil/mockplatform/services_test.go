package mockplatform

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

func TestRawServiceContractShapes(t *testing.T) {
	s := New(t)
	d := s.ServiceDependencies(t)
	specs := map[string]string{
		"artifact_registry": `"registry":{"format":"oci","capacity_bytes":1024,"retention_days":7}`,
		"database":          `"storage":{"profile":"shared","capacity_bytes":1024,"retention_days":7}`,
		"bucket":            `"storage":{"profile":"object","capacity_bytes":1024,"retention_days":7}`,
		"secret_reference":  `"secret":{"external_owner":"external","external_id":"credential","version":"v1"}`,
		"service_binding":   fmt.Sprintf(`"binding":{"consumer_kind":"container_service","consumer_id":%q,"target_kind":"database","target_id":%q,"slot":"db","intent":"database_connect"}`, d.Consumer.ID, d.Database.ID),
		"container_service": fmt.Sprintf(`"container":{"registry_id":%q,"repository":"app","digest":%q,"service_account_id":%q,"cpu_millis":100,"memory_mib":64,"configuration":{},"secret_references":{},"required_bindings":[]}`, d.Registry.ID, TestDigest, d.Account.ID),
		"endpoint":          fmt.Sprintf(`"endpoint":{"service_id":%q,"port":8080,"exposure":"private","tls_policy":"managed"}`, d.Consumer.ID),
	}
	for kind, settings := range specs {
		t.Run(kind, func(t *testing.T) {
			body := fmt.Sprintf(`{"kind":"create","resource_kind":%q,"management_key":%q,"target_id":"","expected_revision":0,"expected_generation":0,"spec":{"name":"raw","deletion_protection":false,%s}}`, kind, "raw-"+kind, settings)
			validate(t, "Mutation", []byte(body))
			status, b := raw(t, s, "POST", s.Base()+"/"+api.Collection(kind), body)
			if status != 202 {
				t.Fatalf("%d %s", status, b)
			}
			validate(t, "Operation", b)
			var op struct {
				TargetID string `json:"target_id"`
			}
			if err := json.Unmarshal(b, &op); err != nil {
				t.Fatal(err)
			}
			status, b = raw(t, s, "GET", s.Base()+"/"+api.Collection(kind)+"/"+op.TargetID, "")
			if status != 200 {
				t.Fatal(status)
			}
			validate(t, "Object", b)
			status, b = raw(t, s, "GET", s.Base()+"/"+api.Collection(kind), "")
			if status != 200 {
				t.Fatal(status)
			}
			validate(t, "Page", b)
			// This independently formed raw request must not turn missing protection into false.
			bad := strings.Replace(body, `"deletion_protection":false,`, "", 1)
			status, _ = raw(t, s, "POST", s.Base()+"/"+api.Collection(kind), bad)
			if status != 400 {
				t.Fatal("missing protection accepted")
			}
		})
	}
	for _, settings := range []string{`"registry":{"format":"oci","capacity_bytes":1}`, `"storage":{"profile":"shared","capacity_bytes":1,"retention_days":0}`, `"registry":{"format":"oci","capacity_bytes":1,"retention_days":0,"credential":"secret-canary"}`} {
		body := `{"kind":"create","resource_kind":"artifact_registry","management_key":"invalid","target_id":"","expected_revision":0,"expected_generation":0,"spec":{"name":"invalid","deletion_protection":false,` + settings + `}}`
		status, b := raw(t, s, "POST", s.Base()+"/artifact_registries", body)
		if status != 400 || strings.Contains(string(b), "secret-canary") {
			t.Fatalf("invalid spec: %d %s", status, b)
		}
	}
}
func TestRawBindingSlotSetReplay(t *testing.T) {
	s := New(t)
	d := s.ServiceDependencies(t)
	body := fmt.Sprintf(`{"kind":"create","resource_kind":"container_service","management_key":"set-replay","target_id":"","expected_revision":0,"expected_generation":0,"spec":{"name":"raw","deletion_protection":false,"container":{"registry_id":%q,"repository":"app","digest":%q,"service_account_id":%q,"cpu_millis":100,"memory_mib":64,"configuration":{"z":"last","a":"first"},"secret_references":{},"required_bindings":["z","a"]}}}`, d.Registry.ID, TestDigest, d.Account.ID)
	before := s.OperationCount()
	status, first := raw(t, s, "POST", s.Base()+"/container_services", body)
	if status != 202 {
		t.Fatalf("create: %d %s", status, first)
	}
	replay := strings.Replace(strings.Replace(body, `["z","a"]`, `["a","z"]`, 1), `{"z":"last","a":"first"}`, `{"a":"first","z":"last"}`, 1)
	status, second := raw(t, s, "POST", s.Base()+"/container_services", replay)
	var a, b struct {
		ID, Fingerprint string
		TargetID        string `json:"target_id"`
	}
	if err := json.Unmarshal(first, &a); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(second, &b); err != nil {
		t.Fatal(err)
	}
	if status != 202 || a != b || a.ID == "" || s.OperationCount() != before+1 {
		t.Fatalf("ordering duplicated effect: %d %s", status, second)
	}
}
