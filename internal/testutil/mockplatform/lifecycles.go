package mockplatform

import (
	"encoding/json"
	"fmt"
	"net/netip"
	"reflect"
	"slices"
	"time"

	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

type execution struct {
	Workload                                                     api.ContainerSpec
	ID, Job, Schedule, State                                     string
	Scope                                                        api.Scope
	Revision, Due, Attempt, Deadline, RetryLimit, TimeoutSeconds int64
}
type recoveryPoint struct {
	ID, Policy, Database string
	Scope                api.Scope
	Revision, Created    int64
	Data                 string
	Complete             bool
}
type writer struct {
	Attachment, Service string
	Epoch               int64
}
type metric struct{ CPU, At int64 }
type lifecycleState struct {
	Clock         int64
	Runs          map[string]execution
	Next          map[string]int64
	Points        map[string]recoveryPoint
	Writers       map[string]writer
	Epochs        map[string]int64
	Data          map[string]string
	DNS           map[string]string
	Peers         map[string]bool
	Metrics       map[string]metric
	Capacity      map[string]int64
	CapacityLimit map[string]int64
	LastScale     map[string]int64
	Regions       map[string]string
}

func (s *Server) initLifecycles() {
	s.state.Lifecycle = lifecycleState{Clock: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Unix(), Runs: map[string]execution{}, Next: map[string]int64{}, Points: map[string]recoveryPoint{}, Writers: map[string]writer{}, Epochs: map[string]int64{}, Data: map[string]string{}, DNS: map[string]string{}, Peers: map[string]bool{}, Metrics: map[string]metric{}, Capacity: map[string]int64{}, CapacityLimit: map[string]int64{}, LastScale: map[string]int64{}, Regions: map[string]string{}}
}
func lifecycleRefs(scope api.Scope, spec api.Spec) []string {
	out := []string{}
	add := func(kind, id string) { out = append(out, objectPath(scope, kind, id)) }
	if v := spec.Job; v != nil {
		out = append(out, refs(scope, "container_service", api.Spec{Container: &v.Workload})...)
	}
	if v := spec.Schedule; v != nil {
		add("job", v.JobID)
	}
	if v := spec.Attachment; v != nil {
		add("volume", v.VolumeID)
		add("container_service", v.ServiceID)
	}
	if v := spec.Domain; v != nil {
		add("endpoint", v.EndpointID)
	}
	if v := spec.Connection; v != nil {
		add("container_service", v.ServiceID)
	}
	if v := spec.Backup; v != nil {
		add("database", v.DatabaseID)
	}
	if v := spec.Clone; v != nil {
		add("database", v.SourceDatabaseID)
	}
	if v := spec.Scaling; v != nil {
		add("container_service", v.ServiceID)
	}
	if v := spec.Placement; v != nil {
		add("container_service", v.ServiceID)
	}
	return out
}
func (s *Server) checkLifecycle(scope api.Scope, kind string, req api.Intent, skip string) string {
	old := s.state.Objects[objectPath(scope, kind, req.TargetID)]
	if req.Kind == "delete" {
		for _, run := range s.state.Lifecycle.Runs {
			if run.Scope == scope && (run.Job == old.ID || run.Schedule == old.ID) && (run.State == "active" || run.State == "unknown") {
				return "dependency_conflict"
			}
		}
		for _, point := range s.state.Lifecycle.Points {
			if point.Scope == scope && (point.Database == old.ID || point.Policy == old.ID) {
				return "dependency_conflict"
			}
		}
		return ""
	}
	v := *req.Spec
	var capacity int64
	if v.Stream != nil {
		capacity = v.Stream.CapacityBytes
	}
	if v.Queue != nil {
		capacity = v.Queue.CapacityBytes
	}
	if v.Volume != nil {
		capacity = v.Volume.CapacityBytes
	}
	if capacity > 0 && capacity < s.state.UsedBytes[objectPath(scope, kind, old.ID)] {
		return "quota_exceeded"
	}
	if old.Job != nil && v.Job != nil && v.Job.MaxConcurrent < s.activeRuns(scope, old.ID) {
		return "dependency_conflict"
	}
	if old.Schedule != nil && v.Schedule != nil && old.Schedule.JobID != v.Schedule.JobID {
		return "invalid_request"
	}
	if v.Stream != nil && old.Stream != nil {
		if v.Stream.Partitions != old.Stream.Partitions || v.Stream.Ordering != old.Stream.Ordering {
			return "invalid_request"
		}
		if v.Stream.CapacityBytes < old.Stream.CapacityBytes {
			return "invalid_request"
		}
		if v.Stream.RetentionDays < old.Stream.RetentionDays && s.state.UsedBytes[objectPath(scope, kind, old.ID)] > 0 {
			return "dependency_conflict"
		}
	}
	if v.Queue != nil && old.Queue != nil {
		if v.Queue.Ordering != old.Queue.Ordering || v.Queue.CapacityBytes < old.Queue.CapacityBytes {
			return "invalid_request"
		}
		if v.Queue.RetentionDays < old.Queue.RetentionDays && s.state.UsedBytes[objectPath(scope, kind, old.ID)] > 0 {
			return "dependency_conflict"
		}
	}
	if v.Volume != nil && old.Volume != nil && (v.Volume.CapacityBytes < old.Volume.CapacityBytes || v.Volume.Region != old.Volume.Region || v.Volume.Profile != old.Volume.Profile || v.Volume.AccessMode != old.Volume.AccessMode) {
		return "invalid_request"
	}
	if old.Attachment != nil && !reflect.DeepEqual(old.Attachment, v.Attachment) {
		return "invalid_request"
	}
	if old.Domain != nil && !reflect.DeepEqual(old.Domain, v.Domain) {
		return "invalid_request"
	}
	if old.Connection != nil && !reflect.DeepEqual(old.Connection, v.Connection) {
		return "invalid_request"
	}
	if old.Backup != nil && v.Backup != nil && (old.Backup.DatabaseID != v.Backup.DatabaseID || v.Backup.RetentionDays < old.Backup.RetentionDays) {
		for _, point := range s.state.Lifecycle.Points {
			if point.Policy == old.ID && point.Scope == scope {
				return "dependency_conflict"
			}
		}
		if old.Backup.DatabaseID != v.Backup.DatabaseID {
			return "invalid_request"
		}
	}
	if old.Clone != nil && !reflect.DeepEqual(old.Clone, v.Clone) {
		return "invalid_request"
	}
	if old.Scaling != nil && v.Scaling != nil && old.Scaling.ServiceID != v.Scaling.ServiceID {
		return "invalid_request"
	}
	if old.Placement != nil && v.Placement != nil && old.Placement.ServiceID != v.Placement.ServiceID {
		return "invalid_request"
	}
	if v.Clone != nil {
		p, ok := s.state.Lifecycle.Points[v.Clone.RecoveryPointID]
		if !ok || !p.Complete || p.Scope != scope || p.Database != v.Clone.SourceDatabaseID {
			return "dependency_conflict"
		}
		if int64(len(p.Data)) > v.Clone.CapacityBytes {
			return "quota_exceeded"
		}
	}
	if v.Attachment != nil {
		volume := s.state.Objects[objectPath(scope, "volume", v.Attachment.VolumeID)]
		region := s.state.Lifecycle.Regions[objectPath(scope, "container_service", v.Attachment.ServiceID)]
		if region != "" && volume.Volume.Region != region {
			return "dependency_conflict"
		}
	}
	if v.Placement != nil {
		if v.Placement.Region != "region-a" && v.Placement.Region != "region-b" {
			return "invalid_request"
		}
		for _, o := range s.state.Objects {
			if o.Scope == scope && o.Attachment != nil && o.Attachment.ServiceID == v.Placement.ServiceID {
				volume := s.state.Objects[objectPath(scope, "volume", o.Attachment.VolumeID)]
				if volume.Volume.Region != v.Placement.Region {
					return "dependency_conflict"
				}
			}
		}
	}
	conflict := func(other api.Spec, otherScope api.Scope) bool {
		if v.Domain != nil && other.Domain != nil && v.Domain.Domain == other.Domain.Domain {
			return true
		}
		if otherScope != scope {
			return false
		}
		// Opposite kinds must reserve compatible placement while either is pending.
		attachment, placement := v.Attachment, other.Placement
		if attachment == nil || placement == nil {
			attachment, placement = other.Attachment, v.Placement
		}
		if attachment != nil && placement != nil && attachment.ServiceID == placement.ServiceID {
			volume := s.state.Objects[objectPath(scope, "volume", attachment.VolumeID)]
			if volume.Volume != nil && volume.Volume.Region != placement.Region {
				return true
			}
		}

		if v.Attachment != nil && other.Attachment != nil && (v.Attachment.VolumeID == other.Attachment.VolumeID || v.Attachment.ServiceID == other.Attachment.ServiceID && v.Attachment.MountPath == other.Attachment.MountPath) {
			return true
		}
		if v.Scaling != nil && other.Scaling != nil && v.Scaling.ServiceID == other.Scaling.ServiceID {
			return true
		}
		if v.Placement != nil && other.Placement != nil && v.Placement.ServiceID == other.Placement.ServiceID {
			return true
		}
		if v.Connection != nil && other.Connection != nil && v.Connection.ServiceID == other.Connection.ServiceID {
			a, _ := netip.ParsePrefix(v.Connection.CIDR)
			b, _ := netip.ParsePrefix(other.Connection.CIDR)
			return a.Overlaps(b)
		}
		return false
	}
	for _, o := range s.state.Objects {
		if !(o.Scope == scope && o.ResourceKind == kind && o.ID == req.TargetID) && conflict(o.Spec, o.Scope) {
			return "dependency_conflict"
		}
	}
	for id, op := range s.state.Operations {
		if id != skip && op.Status == "accepted" && op.Kind != "delete" && !(op.Scope == scope && op.ResourceKind == kind && op.TargetID == req.TargetID) {
			if i := s.state.Intents[id]; i.Spec != nil && conflict(*i.Spec, op.Scope) {
				return "dependency_conflict"
			}
		}
	}
	return ""
}
func (s *Server) lifecycleChanged(op api.Operation, previous api.Object) {
	l := &s.state.Lifecycle
	path := objectPath(op.Scope, op.ResourceKind, op.TargetID)
	if op.Kind == "delete" {
		delete(l.Next, path)
		if v := previous.Attachment; v != nil {
			volume := objectPath(op.Scope, "volume", v.VolumeID)
			l.Epochs[volume]++
			delete(l.Writers, volume)
		}
		// Last capacity/location remains frozen when its policy is removed.
		return
	}
	o := s.state.Objects[path]
	if o.Schedule != nil && (previous.Schedule == nil || !reflect.DeepEqual(o.Schedule, previous.Schedule)) {
		schedule, _ := api.ParseSchedule(o.Schedule.Cron, o.Schedule.TimeZone)
		l.Next[path] = schedule.Next(time.Unix(l.Clock, 0)).Unix()
	}
	if o.Backup != nil && (previous.Backup == nil || !reflect.DeepEqual(o.Backup, previous.Backup)) {
		schedule, _ := api.ParseSchedule(o.Backup.Cron, o.Backup.TimeZone)
		l.Next[path] = schedule.Next(time.Unix(l.Clock, 0)).Unix()
	}
	if o.Domain != nil && previous.Domain == nil {
		o.Status.Challenge = "challenge-" + o.ID
		o.Status.ChallengeExpires = l.Clock + 3600
	}
	if o.Clone != nil && previous.Clone == nil {
		point := l.Points[o.Clone.RecoveryPointID]
		o.Status.RecoveryPointID = point.ID
		o.Status.SourceRevision = point.Revision
		l.Data[path] = point.Data
		s.state.UsedBytes[path] = int64(len(point.Data))
	}
	if o.ResourceKind == "container_service" {
		if _, ok := l.Capacity[path]; !ok {
			l.Capacity[path] = 1
		}
		if l.Regions[path] == "" {
			l.Regions[path] = "region-a"
		}
	}
	s.state.Objects[path] = o
}
func (s *Server) activeRuns(scope api.Scope, job string) int64 {
	var n int64
	for _, r := range s.state.Lifecycle.Runs {
		if r.Scope == scope && r.Job == job && (r.State == "active" || r.State == "unknown") {
			n++
		}
	}
	return n
}
func (s *Server) lifecycleReadiness(o *api.Object) string {
	l := &s.state.Lifecycle
	if v := o.Job; v != nil {
		copy := *o
		copy.Container = &v.Workload
		copy.Job = nil
		return s.readinessReason(copy)
	}
	if v := o.Attachment; v != nil {
		volume := objectPath(o.Scope, "volume", v.VolumeID)
		consumer := s.state.Objects[objectPath(o.Scope, "container_service", v.ServiceID)]
		o.Status.Attached = false
		if consumer.Container == nil || !s.runtimeAllowed(o.Scope, consumer.Container.ServiceAccountID, volume, "volume_mount") {
			return "resolution_denied"
		}
		owner, ok := l.Writers[volume]
		if !ok {
			l.Epochs[volume]++
			owner = writer{Attachment: o.ID, Service: v.ServiceID, Epoch: l.Epochs[volume]}
			l.Writers[volume] = owner
		}
		if owner.Attachment != o.ID {
			return "attachment_pending"
		}
		o.Status.WriterEpoch = owner.Epoch
		o.Status.Attached = true
	}
	if v := o.Domain; v != nil {
		if l.Clock >= o.Status.ChallengeExpires {
			o.Status.Verified = false
			o.Status.Challenge = fmt.Sprintf("challenge-%s-%d", o.ID, l.Clock)
			o.Status.ChallengeExpires = l.Clock + 3600
		}
		o.Status.Verified = false
		if l.DNS[v.Domain] != o.Status.Challenge {
			return "verification_pending"
		}
		o.Status.Verified = true
		endpoint := s.state.Objects[objectPath(o.Scope, "endpoint", v.EndpointID)]
		if endpoint.Status == nil || endpoint.Status.Readiness != "ready" {
			return "dependency_unavailable"
		}
	}
	if v := o.Connection; v != nil {
		o.Status.PeerAccepted = l.Peers[v.PeerOrganizationID+"/"+v.PeerNetworkID]
		if !o.Status.PeerAccepted {
			return "peer_pending"
		}
	}
	if v := o.Scaling; v != nil {
		path := objectPath(o.Scope, "container_service", v.ServiceID)
		current := l.Capacity[path]
		o.Status.ObservedCapacity = current
		m, ok := l.Metrics[path]
		if !ok || l.Clock-m.At > 300 {
			return "metrics_unavailable"
		}
		desired := current
		if m.CPU > v.TargetCPUPercent {
			desired++
		} else if m.CPU < v.TargetCPUPercent/2 {
			desired--
		}
		desired = max(v.MinInstances, min(desired, v.MaxInstances))
		limit := int64(1000)
		if n, ok := l.CapacityLimit[path]; ok {
			limit = n
		}
		if desired > limit {
			return "quota_exceeded"
		}
		if l.Clock-l.LastScale[path] >= v.CooldownSeconds {
			l.Capacity[path] = desired
			l.LastScale[path] = l.Clock
		}
		o.Status.ObservedCapacity = l.Capacity[path]
	}
	if v := o.Placement; v != nil {
		path := objectPath(o.Scope, "container_service", v.ServiceID)
		for _, a := range s.state.Objects {
			if a.Scope == o.Scope && a.Attachment != nil && a.Attachment.ServiceID == v.ServiceID {
				volume := s.state.Objects[objectPath(o.Scope, "volume", a.Attachment.VolumeID)]
				if volume.Volume.Region != v.Region {
					return "placement_conflict"
				}
			}
		}
		l.Regions[path] = v.Region
		o.Status.ObservedRegion = v.Region
	}
	return ""
}

// AdvanceTime is an explicit fixture scheduler. GET never calls it. Each schedule
// admits at most one occurrence per call; catch-up is bounded and original run
// identity is retained across scheduler restart and missed responses.
func (s *Server) AdvanceTime(at time.Time) (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Fixture transitions are atomic with durable scheduling identity, including failures.
	before, _ := json.Marshal(s.state)
	defer func() {
		if err != nil {
			var restored ledger
			_ = json.Unmarshal(before, &restored)
			s.state = restored
		}
	}()
	l := &s.state.Lifecycle
	if at.Unix() < l.Clock {
		return fmt.Errorf("fixture clock cannot move backwards")
	}
	l.Clock = at.Unix()
	for id, run := range l.Runs {
		if run.State == "active" && l.Clock >= run.Deadline {
			run.State = "unknown"
			l.Runs[id] = run
		}
	}
	paths := []string{}
	for path := range s.state.Objects {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	for _, path := range paths {
		o := s.state.Objects[path]
		if o.Schedule != nil {
			v := o.Schedule
			schedule, _ := api.ParseSchedule(v.Cron, v.TimeZone)
			due := l.Next[path]
			if due == 0 || due > l.Clock {
				continue
			}
			l.Next[path] = schedule.Next(at).Unix()
			if !v.Enabled || due < l.Clock && v.MissedRun == "skip" {
				continue
			}
			job := s.state.Objects[objectPath(o.Scope, "job", v.JobID)]
			owner := s.currentPrincipal(o.OwnerID, o.OrganizationID)
			if !s.can(owner, path, "manage") || !s.can(owner, objectPath(o.Scope, "job", v.JobID), "read") {
				continue
			}
			if job.Job == nil || s.readinessReason(api.Object{Bucket: api.Bucket{Scope: job.Scope, Spec: api.Spec{Container: &job.Job.Workload}, ID: job.ID}, ResourceKind: "job"}) != "" {
				continue
			}
			active := s.activeRuns(o.Scope, job.ID)
			if active >= job.Job.MaxConcurrent || v.Overlap == "forbid" && active > 0 {
				continue
			}
			if len(l.Runs) >= 2048 {
				return fmt.Errorf("fixture execution retention budget exhausted")
			}
			id := fmt.Sprintf("run-%s-%d-%d", o.ID, o.Revision, due)
			if _, exists := l.Runs[id]; !exists {
				l.Runs[id] = execution{Workload: job.Job.Workload, ID: id, Job: job.ID, Schedule: o.ID, Scope: o.Scope, State: "active", Revision: job.Revision, Due: due, Attempt: 1, Deadline: l.Clock + job.Job.TimeoutSeconds, RetryLimit: job.Job.RetryLimit, TimeoutSeconds: job.Job.TimeoutSeconds}
				job.Status.ExecutionCount++
				job.Status.ActiveExecutions = active + 1
				job.Status.LastExecutionID = id
				o.Status.ExecutionCount++
				o.Status.ActiveExecutions = s.scheduleActive(o.Scope, o.ID)
				o.Status.LastExecutionID = id
				s.state.Objects[objectPath(o.Scope, "job", job.ID)] = job
				s.state.Objects[path] = o
			}
		}
		if o.Backup != nil {
			v := o.Backup
			schedule, _ := api.ParseSchedule(v.Cron, v.TimeZone)
			if l.Next[path] != 0 && l.Next[path] <= l.Clock {
				l.Next[path] = schedule.Next(at).Unix()
				if v.Enabled {
					if err := s.capture(o); err != nil {
						return err
					}
				}
			}
		}
	}
	s.state.Epoch++
	s.persist()
	return nil
}
func (s *Server) FinishExecution(id, state string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.state.Lifecycle.Runs[id]
	if !ok || (run.State != "active" && run.State != "unknown") || !slices.Contains([]string{"succeeded", "failed", "unknown"}, state) {
		return fmt.Errorf("invalid execution outcome")
	}
	jobPath := objectPath(run.Scope, "job", run.Job)
	job := s.state.Objects[jobPath]
	if state == "failed" && run.State == "active" && job.Job != nil && run.Attempt <= run.RetryLimit && s.readinessReason(api.Object{Bucket: api.Bucket{ID: run.Job, Scope: run.Scope, Spec: api.Spec{Container: &run.Workload}}, ResourceKind: "job"}) == "" {
		run.Attempt++
		run.Deadline = s.state.Lifecycle.Clock + run.TimeoutSeconds
	} else {
		run.State = state
	}
	s.state.Lifecycle.Runs[id] = run
	for _, path := range []string{jobPath, objectPath(run.Scope, "schedule", run.Schedule)} {
		o, ok := s.state.Objects[path]
		if ok {
			o.Status.ActiveExecutions = s.activeRuns(run.Scope, run.Job)
			if o.Schedule != nil {
				o.Status.ActiveExecutions = s.scheduleActive(run.Scope, run.Schedule)
			}
			s.state.Objects[path] = o
		}
	}
	s.state.Epoch++
	s.persist()
	return nil
}
func (s *Server) capture(policy api.Object) error {
	l := &s.state.Lifecycle
	if len(l.Points) >= 2048 {
		return fmt.Errorf("fixture recovery retention budget exhausted")
	}
	owner := s.currentPrincipal(policy.OwnerID, policy.OrganizationID)
	if !s.can(owner, objectPath(policy.Scope, "backup_policy", policy.ID), "manage") || !s.can(owner, objectPath(policy.Scope, "database", policy.Backup.DatabaseID), "read") {
		return fmt.Errorf("backup authority revoked")
	}
	db := s.state.Objects[objectPath(policy.Scope, "database", policy.Backup.DatabaseID)]
	if db.ID == "" {
		return fmt.Errorf("source unavailable")
	}
	id := fmt.Sprintf("point-%s-%d", policy.ID, l.Clock)
	if _, ok := l.Points[id]; !ok {
		l.Points[id] = recoveryPoint{ID: id, Policy: policy.ID, Database: db.ID, Scope: policy.Scope, Revision: db.Revision, Created: l.Clock, Data: l.Data[objectPath(db.Scope, "database", db.ID)], Complete: true}
	}
	policy.Status.RecoveryPointID = id
	policy.Status.SourceRevision = db.Revision
	s.state.Objects[objectPath(policy.Scope, "backup_policy", policy.ID)] = policy
	return nil
}
func (s *Server) ExpireRecoveryPoint(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.state.Lifecycle.Points[id]
	if !ok {
		return fmt.Errorf("missing point")
	}
	for _, o := range s.state.Objects {
		if o.Clone != nil && o.Clone.RecoveryPointID == id {
			return fmt.Errorf("point retained by clone")
		}
	}
	for _, i := range s.state.Intents {
		if i.Spec != nil && i.Spec.Clone != nil && i.Spec.Clone.RecoveryPointID == id {
			for _, op := range s.state.Operations {
				if op.Status == "accepted" && op.Fingerprint == i.Fingerprint() {
					return fmt.Errorf("point retained by accepted clone")
				}
			}
		}
	}
	policy := s.state.Objects[objectPath(p.Scope, "backup_policy", p.Policy)]
	if policy.Backup == nil || s.state.Lifecycle.Clock-p.Created < policy.Backup.RetentionDays*86400 {
		return fmt.Errorf("point retention active")
	}
	delete(s.state.Lifecycle.Points, id)
	s.state.Epoch++
	s.persist()
	return nil
}

// Test-only independent data owners store synthetic canaries, not real services.
func (s *Server) SetData(scope api.Scope, kind, id, data string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	path := objectPath(scope, kind, id)
	s.state.Lifecycle.Data[path] = data
	s.state.UsedBytes[path] = int64(len(data))
	s.persist()
}
func (s *Server) Data(scope api.Scope, kind, id string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.Lifecycle.Data[objectPath(scope, kind, id)]
}
func (s *Server) SetDomainProof(domain, challenge string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Lifecycle.DNS[domain] = challenge
	s.persist()
}
func (s *Server) SetPeerApproval(org, network string, approved bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Lifecycle.Peers[org+"/"+network] = approved
	s.persist()
}
func (s *Server) SetDemand(scope api.Scope, service string, cpu, quota int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	path := objectPath(scope, "container_service", service)
	s.state.Lifecycle.Metrics[path] = metric{CPU: cpu, At: s.state.Lifecycle.Clock}
	s.state.Lifecycle.CapacityLimit[path] = quota
	s.persist()
}
func (s *Server) WriteVolume(scope api.Scope, volume, attachment string, epoch int64, data string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	path := objectPath(scope, "volume", volume)
	w, ok := s.state.Lifecycle.Writers[path]
	if !ok || w.Attachment != attachment || w.Epoch != epoch {
		return fmt.Errorf("stale writer")
	}
	consumer := s.state.Objects[objectPath(scope, "container_service", w.Service)]
	if consumer.Container == nil || !s.runtimeAllowed(scope, consumer.Container.ServiceAccountID, path, "volume_mount") {
		return fmt.Errorf("runtime access denied")
	}
	volumeObject := s.state.Objects[path]
	if volumeObject.Volume == nil || int64(len(data)) > volumeObject.Volume.CapacityBytes {
		return fmt.Errorf("quota_exceeded")
	}
	s.state.Lifecycle.Data[path] = data
	s.state.UsedBytes[path] = int64(len(data))
	s.persist()
	return nil
}
func (s *Server) ReplaceWriter(scope api.Scope, volume string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	path := objectPath(scope, "volume", volume)
	w, ok := s.state.Lifecycle.Writers[path]
	if ok {
		s.state.Lifecycle.Epochs[path]++
		w.Epoch = s.state.Lifecycle.Epochs[path]
		s.state.Lifecycle.Writers[path] = w
	}
	s.persist()
}

func (s *Server) scheduleActive(scope api.Scope, id string) int64 {
	var n int64
	for _, r := range s.state.Lifecycle.Runs {
		if r.Scope == scope && r.Schedule == id && (r.State == "active" || r.State == "unknown") {
			n++
		}
	}
	return n
}
