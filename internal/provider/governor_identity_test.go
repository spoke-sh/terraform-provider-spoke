package provider

import (
	"encoding/json"
	"testing"

	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

const governorTestScope = "/v1/ci/organizations/example/projects/operators/environments/internal"

func governorTestPolicy(name string) *api.GovernorPolicy {
	return &api.GovernorPolicy{Name: name, AdmissionPolicyID: "admission", BudgetPolicyID: "budget", CachePolicyID: "cache", ExecutionProfileIDs: []string{"linux-x86_64-v1"}, MaxActiveJobs: 2, MaxAllocatedVCPUs: 16, MaxJobSeconds: 3600, MaxQueueSeconds: 60, RetirementPolicy: "drain"}
}
func governorTestView(t *testing.T, kind, status, name string, seq, rev uint64) governorView {
	t.Helper()
	id := "governor-one"
	p := governorTestPolicy(name)
	intent := api.GovernorIntent{Kind: kind, Policy: p}
	if kind != "create" {
		intent.TargetID, intent.ExpectedRevision = &id, rev
	}
	if kind == "retire" {
		intent.Policy = nil
	}
	r := api.GovernorRequest{ManagementKey: "stable", Sequence: seq, Intent: intent}
	fp, err := api.GovernorFingerprint(governorTestScope, "operator", r)
	if err != nil {
		t.Fatal(err)
	}
	op := &api.GovernorOperation{Scope: governorTestScope, PrincipalID: "operator", ID: "operation-one", ManagementKey: r.ManagementKey, Sequence: seq, Fingerprint: fp, Intent: intent, Status: status, ResourceID: &id, Reason: "none"}
	g := &api.Governor{Scope: governorTestScope, PrincipalID: "operator", ID: id, ManagementKey: "stable", Revision: rev, Policy: *p}
	if kind == "create" && status != "applied" {
		op.ResourceID, g = nil, nil
	}
	if kind == "retire" && status == "applied" {
		g = nil
	}
	return governorView{Scope: governorTestScope, Owner: "operator", Latest: op, Value: g}
}

func TestGovernorMutationSelection(t *testing.T) {
	anchor := &governorAnchor{Owner: "operator", Target: "governor-one", Sequence: 0, Revision: 1}
	tests := []struct {
		name, latestKind, status, action, desired string
		seq, rev, want                            uint64
		prior                                     *governorAnchor
		err, observe                              bool
	}{
		{"create-replay", "create", "applied", "create", "a", 0, 1, 0, nil, false, false},
		{"lost-state-changed-create", "create", "applied", "create", "b", 0, 1, 0, nil, true, false},
		{"prepared-create", "create", "prepared", "create", "a", 0, 0, 0, nil, false, false},
		{"accepted-create", "create", "accepted", "create", "a", 0, 0, 0, nil, false, false},
		{"pending-create-conflict", "create", "prepared", "create", "b", 0, 0, 0, nil, true, false},
		{"refused-create-next-plan", "create", "refused", "create", "b", 0, 0, 1, nil, false, false},
		{"aborted-create-next-plan", "create", "aborted", "create", "a", 0, 0, 1, nil, false, false},
		{"update", "create", "applied", "update", "b", 0, 1, 1, anchor, false, false},
		{"no-change", "create", "applied", "update", "a", 0, 1, 0, anchor, false, true},
		{"update-prepared", "update", "prepared", "update", "a", 1, 1, 1, anchor, false, false},
		{"update-accepted", "update", "accepted", "update", "a", 1, 1, 1, anchor, false, false},
		{"pending-update-changed", "update", "accepted", "update", "b", 1, 1, 0, anchor, true, false},
		{"pending-update-destroy", "update", "accepted", "retire", "", 1, 1, 0, anchor, true, false},
		{"lost-update-completion", "update", "applied", "update", "a", 1, 1, 1, anchor, false, false},
		{"refusal-needs-refresh", "update", "refused", "update", "b", 1, 1, 0, anchor, true, false},
		{"retire", "create", "applied", "retire", "", 0, 1, 1, anchor, false, false},
		{"retire-prepared", "retire", "prepared", "retire", "", 1, 1, 1, anchor, false, false},
		{"retire-accepted", "retire", "accepted", "retire", "", 1, 1, 1, anchor, false, false},
		{"lost-retire-completion", "retire", "applied", "retire", "", 1, 1, 1, anchor, false, false},
		{"retired-key-create", "retire", "applied", "create", "a", 1, 1, 0, nil, true, false},
		{"stale-sequence", "update", "applied", "update", "b", 2, 3, 0, anchor, true, false},
		{"stale-revision", "create", "applied", "update", "b", 0, 2, 0, anchor, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := governorTestView(t, tt.latestKind, tt.status, "a", tt.seq, tt.rev)
			var p *api.GovernorPolicy
			if tt.desired != "" {
				p = governorTestPolicy(tt.desired)
			}
			f, err := governorSelect(v, "stable", tt.action, p, tt.prior)
			if (err != nil) != tt.err || err == nil && (f.Request.Sequence != tt.want || f.Observe != tt.observe) {
				t.Fatalf("selection: %+v err=%v", f, err)
			}
			if err == nil {
				b, _ := json.Marshal(f)
				decoded, err := governorDecodeFence(b)
				if err != nil || governorCheckFence(v, decoded) != nil {
					t.Fatalf("selected fence does not validate: %v", err)
				}
			}
		})
	}
}

func TestGovernorSavedFenceNeverRebases(t *testing.T) {
	v := governorTestView(t, "create", "applied", "a", 0, 1)
	a := &governorAnchor{Owner: "operator", Target: "governor-one", Sequence: 0, Revision: 1}
	f, err := governorSelect(v, "stable", "update", governorTestPolicy("b"), a)
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []string{"revision", "principal", "scope", "sequence", "intent", "pending"} {
		t.Run(change, func(t *testing.T) {
			v := governorTestView(t, "create", "applied", "a", 0, 1)
			switch change {
			case "revision":
				v.Value.Revision++
			case "principal":
				v.Owner = "foreign"
			case "scope":
				v.Scope += "-other"
			case "sequence":
				v.Latest.Sequence = 2
			case "intent":
				v = governorTestView(t, "update", "prepared", "c", 1, 1)
			case "pending":
				v.Latest.Status = "accepted"
			}
			if governorCheckFence(v, f) == nil {
				t.Fatal("stale saved fence permitted mutation")
			}
		})
	}
	for _, raw := range []string{`{}`, `{"version":2}`, `{"version":1,"extra":true}`, `null`, `{} {}`} {
		if _, err := governorDecodeFence([]byte(raw)); err == nil {
			t.Fatal("accepted incompatible plan metadata")
		}
	}
}

func TestGovernorSelectionOwnershipAndBounds(t *testing.T) {
	v := governorView{Scope: governorTestScope, Owner: "operator"}
	f, err := governorSelect(v, "stable", "create", governorTestPolicy("a"), nil)
	if err != nil || f.Request.Sequence != 0 || governorCheckFence(v, f) != nil {
		t.Fatal("initial request", err)
	}
	v = governorTestView(t, "create", "applied", "a", 0, 1)
	a := &governorAnchor{Owner: "operator", Target: "governor-one", Sequence: 0, Revision: 1}
	v.Value.Policy.DeletionProtection = true
	if _, err := governorSelect(v, "stable", "retire", nil, a); err == nil {
		t.Fatal("protected retirement")
	}
	p := governorTestPolicy("b")
	p.AdmissionPolicyID = "other"
	if _, err := governorSelect(v, "stable", "update", p, a); err == nil {
		t.Fatal("changed entitlement")
	}
	v.Latest.Sequence, a.Sequence = 4294967295, 4294967295
	if _, err := governorSelect(v, "stable", "update", governorTestPolicy("b"), a); err == nil {
		t.Fatal("sequence wraparound")
	}
	a.Owner = "foreign"
	if _, err := governorSelect(v, "stable", "update", governorTestPolicy("a"), a); err == nil {
		t.Fatal("ownership transfer")
	}
}
