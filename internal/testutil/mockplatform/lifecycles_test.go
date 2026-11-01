package mockplatform

import (
	"context"
	"encoding/json"
	"fmt"
	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
	"strings"
	"testing"
	"time"
)

func lifecycleClient(t *testing.T, s *Server) *api.Client {
	t.Helper()
	c, err := api.New(api.Config{Endpoint: s.HTTP.URL, Scope: s.Scope, Token: s.Token, HTTPClient: s.HTTP.Client(), PollInterval: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func editLifecycle(t *testing.T, s *Server, o api.Object, spec *api.Spec, want string) {
	t.Helper()
	action := "update"
	if spec == nil {
		action = "delete"
	}
	c, e := api.New(api.Config{Endpoint: s.HTTP.URL, Scope: o.Scope, Token: s.Token, HTTPClient: s.HTTP.Client(), PollInterval: time.Millisecond})
	if e != nil {
		t.Fatal(e)
	}
	_, err := c.Mutate(context.Background(), api.Intent{Kind: action, ResourceKind: o.ResourceKind, ManagementKey: o.ManagementKey, TargetID: o.ID, ExpectedRevision: o.Revision, Spec: spec})
	if want == "" && err != nil || want != "" && !api.IsCode(err, want) {
		t.Fatalf("%s %s: %v; want %s", action, o.ResourceKind, err, want)
	}
}
func latest(t *testing.T, s *Server, o api.Object) api.Object {
	t.Helper()
	v, err := lifecycleClient(t, s).GetObject(context.Background(), o.ResourceKind, o.ID)
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func clockAt(t *testing.T, s *Server, minutes int) {
	t.Helper()
	if err := s.AdvanceTime(time.Date(2026, 1, 1, 0, minutes, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
}
func readyWorkload(t *testing.T, s *Server, d ServiceDependencies) {
	t.Helper()
	if err := s.PublishArtifact(s.Scope, d.Registry.ID, "app/web", TestDigest, "", 1); err != nil {
		t.Fatal(err)
	}
	s.RuntimeGrant(t, d, d.Registry, "artifact_pull")
	s.AdvanceControllers()
}
func TestLifecycleWireAndCutover(t *testing.T) {
	for _, kind := range api.LifecycleKinds() {
		t.Run(kind, func(t *testing.T) {
			s := New(t)
			d := s.LifecycleDependencies(t)
			spec := d.Spec(kind)
			body, _ := json.Marshal(api.Mutation{Intent: api.Intent{Kind: "create", ResourceKind: kind, ManagementKey: "raw-new", Spec: &spec}})
			validate(t, "Mutation", body)
			status, b := raw(t, s, "POST", s.Base()+"/"+api.Collection(kind), string(body))
			if status != 202 {
				t.Fatalf("%d %s", status, b)
			}
			validate(t, "Operation", b)
			var op api.Operation
			_ = json.Unmarshal(b, &op)
			status, b = raw(t, s, "GET", s.Base()+"/"+api.Collection(kind)+"/"+op.TargetID, "")
			if status != 200 {
				t.Fatalf("%d %s", status, b)
			}
			validate(t, "Object", b)
			before := s.OperationCount()
			status, _ = raw(t, s, "POST", s.Base()+"/"+api.Collection(kind), string(body))
			if status != 202 || s.OperationCount() != before {
				t.Fatal("replay duplicated work")
			}
			status, b = raw(t, s, "GET", s.Base()+"/"+api.Collection(kind), "")
			if status != 200 {
				t.Fatal(status)
			}
			validate(t, "Page", b)
		})
	}
}
func TestScheduleAdmissionRecoveryAndUncertainty(t *testing.T) {
	s := New(t)
	d := s.LifecycleDependencies(t)
	readyWorkload(t, s, d)
	spec := d.Spec("schedule")
	spec.Schedule.Enabled = true
	o := s.Provision(t, "schedule", "timer", spec)
	c := lifecycleClient(t, s)
	before := s.OperationCount()
	for range 3 {
		items, err := c.History(context.Background(), "job", d.Job.ID)
		if err != nil || len(items) != 0 {
			t.Fatalf("read executed job: %v", err)
		}
	}
	if s.OperationCount() != before {
		t.Fatal("read mutated")
	}
	clockAt(t, s, 2)
	items, err := c.History(context.Background(), "job", d.Job.ID)
	if err != nil || len(items) != 1 {
		t.Fatalf("admission: %+v %v", items, err)
	}
	run := items[0].ID
	s.Restart(t)
	clockAt(t, s, 2)
	clockAt(t, s, 4)
	items, _ = c.History(context.Background(), "job", d.Job.ID)
	if len(items) != 1 {
		t.Fatal("restart/overlap duplicated")
	}
	if err := s.FinishExecution(run, "failed"); err != nil {
		t.Fatal(err)
	}
	items, _ = c.History(context.Background(), "job", d.Job.ID)
	if items[0].Attempt != 2 || items[0].State != "active" {
		t.Fatal(items)
	}
	if err := s.FinishExecution(run, "unknown"); err != nil {
		t.Fatal(err)
	}
	editLifecycle(t, s, o, nil, "dependency_conflict")
	clockAt(t, s, 5)
	items, _ = c.History(context.Background(), "job", d.Job.ID)
	if len(items) != 1 {
		t.Fatal("uncertainty admitted new work")
	}
	if err := s.FinishExecution(run, "succeeded"); err != nil {
		t.Fatal(err)
	}
	clockAt(t, s, 6)
	items, _ = c.History(context.Background(), "job", d.Job.ID)
	if len(items) != 2 {
		t.Fatal(items)
	}
	clockAt(t, s, 67)
	items, _ = c.History(context.Background(), "job", d.Job.ID)
	if len(items) != 2 || items[1].State != "unknown" {
		t.Fatal("timeout must retain uncertain identity", items)
	}
	status, b := raw(t, s, "GET", s.Base()+"/jobs/"+d.Job.ID+"/executions", "")
	if status != 200 {
		t.Fatal(status)
	}
	validate(t, "History", b)
	s.DenyAccess(true)
	if _, err := c.History(context.Background(), "job", d.Job.ID); !api.IsCode(err, "forbidden") {
		t.Fatal(err)
	}
}
func TestVolumeFencingAndRetainedData(t *testing.T) {
	s := New(t)
	d := s.LifecycleDependencies(t)
	a := s.Provision(t, "volume_attachment", "attach", d.Spec("volume_attachment"))
	s.AdvanceControllers()
	if latest(t, s, a).Status.Reason != "resolution_denied" {
		t.Fatal("mount lacked grant")
	}
	grant := s.RuntimeGrant(t, d, d.Volume, "volume_mount")
	s.AdvanceControllers()
	a = latest(t, s, a)
	epoch := a.Status.WriterEpoch
	if !a.Status.Attached || epoch == 0 {
		t.Fatal(a.Status)
	}
	if err := s.WriteVolume(s.Scope, d.Volume.ID, a.ID, epoch, "persistent"); err != nil {
		t.Fatal(err)
	}
	s.ReplaceWriter(s.Scope, d.Volume.ID)
	s.Restart(t)
	if err := s.WriteVolume(s.Scope, d.Volume.ID, a.ID, epoch, "stale"); err == nil {
		t.Fatal("stale writer accepted")
	}
	s.AdvanceControllers()
	a = latest(t, s, a)
	if a.Status.WriterEpoch <= epoch {
		t.Fatal("fence did not advance")
	}
	editLifecycle(t, s, d.Volume, nil, "dependency_conflict")
	editLifecycle(t, s, grant, nil, "")
	if err := s.WriteVolume(s.Scope, d.Volume.ID, a.ID, a.Status.WriterEpoch, "denied"); err == nil {
		t.Fatal("revocation bypassed")
	}
	editLifecycle(t, s, a, nil, "")
	if s.Data(s.Scope, "volume", d.Volume.ID) != "persistent" {
		t.Fatal("detach erased data")
	}
	if err := s.WriteVolume(s.Scope, d.Volume.ID, a.ID, a.Status.WriterEpoch, "detached"); err == nil {
		t.Fatal("detached writer accepted")
	}
	editLifecycle(t, s, d.Volume, nil, "not_empty")
}
func TestDomainAndPrivateConsent(t *testing.T) {
	s := New(t)
	d := s.LifecycleDependencies(t)
	readyWorkload(t, s, d)
	o := s.Provision(t, "custom_domain", "domain", d.Spec("custom_domain"))
	p := s.Provision(t, "private_connection", "private", d.Spec("private_connection"))
	s.AdvanceControllers()
	o = latest(t, s, o)
	if o.Status.Verified || latest(t, s, p).Status.PeerAccepted {
		t.Fatal("unverified readiness")
	}
	s.SetDomainProof(o.Domain.Domain, o.Status.Challenge)
	s.SetPeerApproval(s.Scope, p.ID, true)
	s.AdvanceControllers()
	if !latest(t, s, o).Status.Verified || latest(t, s, o).Status.TLSState != "ready" || !latest(t, s, p).Status.PeerAccepted {
		t.Fatal("proof not observed")
	}
	s.SetPeerApproval(s.Scope, p.ID, false)
	clockAt(t, s, 62)
	s.AdvanceControllers()
	if latest(t, s, o).Status.Verified || latest(t, s, p).Status.PeerAccepted {
		t.Fatal("stale authorization")
	}
	editLifecycle(t, s, d.Endpoint, nil, "dependency_conflict")
	editLifecycle(t, s, o, nil, "")
	editLifecycle(t, s, p, nil, "")
	if s.Count("container_service") != 1 || s.Count("endpoint") != 1 {
		t.Fatal("network retirement removed owner")
	}
}
func TestRecoveryPointFrozenCloneAndRetention(t *testing.T) {
	s := New(t)
	d := s.LifecycleDependencies(t)
	s.SetData(s.Scope, "database", d.Database.ID, "original")
	clockAt(t, s, 2)
	policy := latest(t, s, d.Backup)
	spec := d.Spec("database_clone")
	spec.Clone.RecoveryPointID = policy.Status.RecoveryPointID
	clone := s.Provision(t, "database_clone", "clone", spec)
	s.SetData(s.Scope, "database", d.Database.ID, "later")
	s.Restart(t)
	if s.Data(s.Scope, "database_clone", clone.ID) != "original" {
		t.Fatal("clone retargeted source")
	}
	s.SetData(s.Scope, "database_clone", clone.ID, "independent")
	if s.Data(s.Scope, "database", d.Database.ID) != "later" {
		t.Fatal("clone aliases source")
	}
	if err := s.ExpireRecoveryPoint(spec.Clone.RecoveryPointID); err == nil {
		t.Fatal("dependent point expired")
	}
	editLifecycle(t, s, d.Database, nil, "dependency_conflict")
	editLifecycle(t, s, policy, nil, "dependency_conflict")
	editLifecycle(t, s, clone, nil, "not_empty")
	s.SetData(s.Scope, "database_clone", clone.ID, "")
	editLifecycle(t, s, clone, nil, "")
	points, err := lifecycleClient(t, s).History(context.Background(), "backup_policy", policy.ID)
	if err != nil || len(points) != 2 {
		t.Fatalf("%+v %v", points, err)
	}
	status, b := raw(t, s, "GET", s.Base()+"/backup_policies/"+policy.ID+"/recovery_points", "")
	if status != 200 || strings.Contains(string(b), "original") {
		t.Fatal("data leaked")
	}
	validate(t, "History", b)
	if err := s.ExpireRecoveryPoint(spec.Clone.RecoveryPointID); err == nil {
		t.Fatal("retention bypassed")
	}
}
func TestMessagingRetentionAndRuntimeGrants(t *testing.T) {
	for _, kind := range []string{"stream", "queue"} {
		t.Run(kind, func(t *testing.T) {
			s := New(t)
			d := s.LifecycleDependencies(t)
			o := s.Provision(t, kind, "messaging", d.Spec(kind))
			spec := d.Spec("service_binding")
			spec.Binding.TargetKind = kind
			spec.Binding.TargetID = o.ID
			spec.Binding.Intent = kind + "_consume"
			b := s.Provision(t, "service_binding", "message-binding", spec)
			s.AdvanceControllers()
			s.AdvanceControllers()
			if latest(t, s, b).Status.Reason != "resolution_denied" {
				t.Fatal("implicit runtime access")
			}
			grant := s.RuntimeGrant(t, d, o, kind+"_consume")
			s.AdvanceControllers()
			if latest(t, s, b).Status.Readiness != "ready" {
				t.Fatal("grant not resolved")
			}
			editLifecycle(t, s, grant, nil, "")
			s.AdvanceControllers()
			if latest(t, s, b).Status.Readiness == "ready" {
				t.Fatal("revocation ignored")
			}
			editLifecycle(t, s, b, nil, "")
			s.SetUsage(s.Scope, kind, o.ID, 10)
			reduced := o.Spec
			if reduced.Stream != nil {
				reduced.Stream.RetentionDays = 0
			} else {
				reduced.Queue.RetentionDays = 0
			}
			editLifecycle(t, s, o, &reduced, "dependency_conflict")
			editLifecycle(t, s, o, nil, "not_empty")
		})
	}
}
func TestCapacityTelemetryAndPlacementConstraints(t *testing.T) {
	s := New(t)
	d := s.LifecycleDependencies(t)
	o := s.Provision(t, "scaling_policy", "scale", d.Spec("scaling_policy"))
	s.AdvanceControllers()
	if latest(t, s, o).Status.Reason != "metrics_unavailable" {
		t.Fatal("missing metrics treated as zero")
	}
	s.SetDemand(s.Scope, d.Consumer.ID, 90, 3)
	s.AdvanceControllers()
	if latest(t, s, o).Status.ObservedCapacity != 2 {
		t.Fatal("scale out")
	}
	s.AdvanceControllers()
	if latest(t, s, o).Status.ObservedCapacity != 2 {
		t.Fatal("cooldown ignored")
	}
	clockAt(t, s, 7)
	s.AdvanceControllers()
	if latest(t, s, o).Status.ObservedCapacity != 2 || latest(t, s, o).Status.Reason != "metrics_unavailable" {
		t.Fatal("stale metrics changed count")
	}
	s.SetDemand(s.Scope, d.Consumer.ID, 90, 2)
	s.AdvanceControllers()
	if latest(t, s, o).Status.Reason != "quota_exceeded" {
		t.Fatal("quota ignored")
	}
	a := s.Provision(t, "volume_attachment", "attach", d.Spec("volume_attachment"))
	p := s.Provision(t, "placement_policy", "place", d.Spec("placement_policy"))
	changed := p.Spec
	changed.Placement.Region = "region-b"
	editLifecycle(t, s, p, &changed, "dependency_conflict")
	editLifecycle(t, s, a, nil, "")
	editLifecycle(t, s, p, &changed, "")
	s.AdvanceControllers()
	if latest(t, s, p).Status.ObservedRegion != "region-b" {
		t.Fatal("stopped placement")
	}
	editLifecycle(t, s, o, nil, "")
	if s.Count("container_service") != 1 {
		t.Fatal("policy deletion removed workload")
	}
}

func TestClockBudgetFailureIsAtomic(t *testing.T) {
	s := New(t)
	d := s.LifecycleDependencies(t)
	readyWorkload(t, s, d)
	spec := d.Spec("schedule")
	spec.Schedule.Enabled = true
	s.Provision(t, "schedule", "timer", spec)
	// Exhaust the independent recovery owner's retention budget. Even if the
	// schedule was visited first, neither its admission nor clock can commit.
	s.mu.Lock()
	for i := 0; i < 2048; i++ {
		id := fmt.Sprintf("budget-%d", i)
		s.state.Lifecycle.Points[id] = recoveryPoint{ID: id}
	}
	before := s.state.Lifecycle.Clock
	s.persist()
	s.mu.Unlock()
	if err := s.AdvanceTime(time.Date(2026, 1, 1, 0, 2, 0, 0, time.UTC)); err == nil {
		t.Fatal("exhaustion passed")
	}
	s.mu.Lock()
	after := s.state.Lifecycle.Clock
	count := len(s.state.Lifecycle.Runs)
	s.mu.Unlock()
	if before != after || count != 0 {
		t.Fatal("partial clock transition escaped")
	}
	s.Restart(t)
	items, err := lifecycleClient(t, s).History(context.Background(), "job", d.Job.ID)
	if err != nil || len(items) != 0 {
		t.Fatal("failed transition persisted", err)
	}
}
func TestJobUpdatesPreserveAdmittedRetryPolicy(t *testing.T) {
	s := New(t)
	d := s.LifecycleDependencies(t)
	readyWorkload(t, s, d)
	spec := d.Spec("schedule")
	spec.Schedule.Enabled = true
	schedule := s.Provision(t, "schedule", "timer", spec)
	clockAt(t, s, 2)
	runs, _ := lifecycleClient(t, s).History(context.Background(), "job", d.Job.ID)
	if len(runs) != 1 {
		t.Fatal(runs)
	}
	changed := d.Job.Spec
	changed.Job.RetryLimit = 0
	editLifecycle(t, s, d.Job, &changed, "")
	disabled := schedule.Spec
	disabled.Schedule.Enabled = false
	editLifecycle(t, s, schedule, &disabled, "")
	if err := s.FinishExecution(runs[0].ID, "failed"); err != nil {
		t.Fatal(err)
	}
	clockAt(t, s, 5)
	runs, _ = lifecycleClient(t, s).History(context.Background(), "job", d.Job.ID)
	if len(runs) != 1 || runs[0].Revision != 1 || runs[0].Attempt != 2 || runs[0].State != "active" {
		t.Fatal("retargeted accepted execution", runs)
	}
	if err := s.FinishExecution(runs[0].ID, "succeeded"); err != nil {
		t.Fatal(err)
	}
	editLifecycle(t, s, latest(t, s, schedule), nil, "")
	editLifecycle(t, s, latest(t, s, d.Job), nil, "")
	runs, err := lifecycleClient(t, s).History(context.Background(), "job", d.Job.ID)
	if err != nil || len(runs) != 1 {
		t.Fatal("retirement lost evidence", err)
	}
}

func TestPendingPlacementReservesVolumeCompatibility(t *testing.T) {
	for _, first := range []string{"volume_attachment", "placement_policy"} {
		t.Run(first, func(t *testing.T) {
			s := New(t)
			d := s.LifecycleDependencies(t)
			s.HoldOperations(true)
			second := "placement_policy"
			if first == second {
				second = "volume_attachment"
			}
			submit := func(kind, key string) int {
				spec := d.Spec(kind)
				if spec.Placement != nil {
					spec.Placement.Region = "region-b"
				}
				body, _ := json.Marshal(api.Mutation{Intent: api.Intent{Kind: "create", ResourceKind: kind, ManagementKey: key, Spec: &spec}})
				status, _ := raw(t, s, "POST", s.Base()+"/"+api.Collection(kind), string(body))
				return status
			}
			if status := submit(first, "first"); status != 202 {
				t.Fatal(status)
			}
			if status := submit(second, "second"); status != 409 {
				t.Fatal("incompatible pending reservation accepted", status)
			}
			s.HoldOperations(false)
			s.CompleteAll()
		})
	}
}

func TestPeerConsentIsBoundToExactConnection(t *testing.T) {
	s := New(t)
	d := s.LifecycleDependencies(t)
	first := s.Provision(t, "private_connection", "first", d.Spec("private_connection"))
	other := s.Provision(t, "container_service", "other", d.Spec("container_service"))
	spec := d.Spec("private_connection")
	spec.Connection.ServiceID = other.ID
	second := s.Provision(t, "private_connection", "second", spec)
	s.SetPeerApproval(s.Scope, first.ID, true)
	s.Restart(t)
	s.AdvanceControllers()
	if !latest(t, s, first).Status.PeerAccepted || latest(t, s, second).Status.PeerAccepted {
		t.Fatal("peer consent escaped its exact local relationship")
	}
}
