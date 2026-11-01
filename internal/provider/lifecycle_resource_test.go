package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"regexp"
	"strings"
	"testing"
)

func TestAccMockLifecycleHistory(t *testing.T) {
	s := mockAcceptance(t)
	d := s.LifecycleDependencies(t)
	config := providerConfig(s) + fmt.Sprintf(`
data "spoke_job_executions" "runs" { parent_id = %q }
data "spoke_recovery_points" "points" { parent_id = %q }
`, d.Job.ID, d.Backup.ID)
	before := mutations(s)
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: factories(), Steps: []resource.TestStep{
		{Config: config, Check: resource.ComposeAggregateTestCheckFunc(resource.TestCheckResourceAttr("data.spoke_job_executions.runs", "items.#", "0"), resource.TestCheckResourceAttr("data.spoke_recovery_points.points", "items.#", "1"))},
		{Config: config, PlanOnly: true},
		{Config: config, PreConfig: func() { s.DenyAccess(true) }, ExpectError: regexp.MustCompile("forbidden")},
		{Config: config, PreConfig: func() { s.DenyAccess(false) }},
	}})
	if mutations(s) != before {
		t.Fatal("history performed work")
	}
}
func TestAccMockLifecycleValidation(t *testing.T) {
	for _, tc := range []struct{ kind, from, to, reason string }{
		{"schedule", `time_zone = "UTC"`, `time_zone = "Local"`, "Invalid schedule"},
		{"schedule", `cron = "* * * * *"`, `cron = "@every 1s"`, "Invalid schedule"},
		{"custom_domain", `domain = "app.example.test"`, `domain = "UPPER.example.test"`, "Invalid lifecycle settings"},
		{"private_connection", `cidr = "10.0.0.0/24"`, `cidr = "8.8.8.0/24"`, "Invalid lifecycle settings"},
		{"volume_attachment", `mount_path = "/data/files"`, `mount_path = "/data/../files"`, "Invalid lifecycle settings"},
		{"scaling_policy", `min_instances = 1`, `min_instances = 5`, "Invalid capacity bounds"},
	} {
		t.Run(tc.kind+tc.from, func(t *testing.T) {
			s := mockAcceptance(t)
			d := s.LifecycleDependencies(t)
			config := strings.Replace(serviceConfig(s, d, tc.kind, "invalid"), tc.from, tc.to, 1)
			before := mutations(s)
			resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: factories(), Steps: []resource.TestStep{{Config: config, PlanOnly: true, ExpectError: regexp.MustCompile(tc.reason)}}})
			if before != mutations(s) {
				t.Fatal("invalid plan mutated service")
			}
		})
	}
}
func TestAccMockControllerObservationsPreservePolicyPlan(t *testing.T) {
	for _, kind := range []string{"custom_domain", "private_connection", "scaling_policy", "placement_policy", "volume_attachment", "job"} {
		t.Run(kind, func(t *testing.T) {
			s := mockAcceptance(t)
			d := s.LifecycleDependencies(t)
			config := serviceConfig(s, d, kind, "policy")
			before := 0
			resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: factories(), Steps: []resource.TestStep{
				{Config: config},
				{Config: config, PreConfig: func() { s.AdvanceControllers() }, Check: resource.TestCheckResourceAttr("spoke_"+kind+".test", "revision", "1")},
				{Config: config, PlanOnly: true, PreConfig: func() { before = mutations(s); s.AdvanceControllers() }},
				{Config: config, PreConfig: func() {
					if before != mutations(s) {
						t.Fatal("observation changed policy")
					}
				}},
			}})
		})
	}
}
