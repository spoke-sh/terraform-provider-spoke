package mockplatform

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"

	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

type artifactRecord struct {
	Artifact api.Artifact
	Tags     []string
}

func (s *Server) initServices() {
	if s.state.Artifacts == nil {
		s.state.Artifacts = map[string]artifactRecord{}
	}
	if s.state.Secrets == nil {
		s.state.Secrets = map[string]bool{}
	}
	if s.state.UsedBytes == nil {
		s.state.UsedBytes = map[string]int64{}
	}
	if s.state.Failures == nil {
		s.state.Failures = map[string]string{}
	}
}
func artifactKey(scope api.Scope, registry, repository, digest string) string {
	return objectPath(scope, "artifact_registry", registry) + "/" + repository + "@" + digest
}
func secretKey(spec api.SecretSpec) string {
	return spec.ExternalOwner + "/" + spec.ExternalID + "/" + spec.Version
}
func (s *Server) PublishArtifact(scope api.Scope, registry, repository, digest, tag string, size int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	o := s.state.Objects[objectPath(scope, "artifact_registry", registry)]
	if o.Registry == nil || !api.ValidRepository(repository) || !api.ValidDigest(digest) || (tag != "" && !api.ValidTag(tag)) || size < 0 {
		return errors.New("invalid artifact metadata")
	}
	key := artifactKey(scope, registry, repository, digest)
	used := int64(0)
	for k, a := range s.state.Artifacts {
		if a.Artifact.Scope == scope && a.Artifact.RegistryID == registry {
			used += a.Artifact.SizeBytes
			if k == key && a.Artifact.SizeBytes != size {
				return errors.New("immutable digest metadata")
			}
		}
	}
	if _, ok := s.state.Artifacts[key]; !ok {
		used += size
	}
	if used > o.Registry.CapacityBytes {
		return errors.New("quota_exceeded")
	}
	for k, a := range s.state.Artifacts {
		if a.Artifact.Scope == scope && a.Artifact.RegistryID == registry && a.Artifact.Repository == repository && tag != "" {
			a.Tags = slices.DeleteFunc(a.Tags, func(v string) bool { return v == tag })
			s.state.Artifacts[k] = a
		}
	}
	a := s.state.Artifacts[key]
	a.Artifact = api.Artifact{Scope: scope, RegistryID: registry, Repository: repository, Digest: digest, SizeBytes: size}
	if tag != "" {
		a.Tags = append(a.Tags, tag)
	}
	s.state.Artifacts[key] = a
	s.state.UsedBytes[objectPath(scope, "artifact_registry", registry)] = used
	s.state.Epoch++
	s.persist()
	return nil
}
func (s *Server) readArtifact(w http.ResponseWriter, r *http.Request, scope api.Scope, id string) {
	q, err := query(r)
	if err != nil || !scope.AllowsKind("artifact_registry") || !api.ValidID(id) || !api.ValidRepository(q["repository"]) || ((q["digest"] == "") == (q["tag"] == "")) || (q["digest"] != "" && !api.ValidDigest(q["digest"])) || (q["tag"] != "" && !api.ValidTag(q["tag"])) {
		fail(w, 400, "invalid_request")
		return
	}
	if s.state.Objects[objectPath(scope, "artifact_registry", id)].ID == "" {
		fail(w, 404, "resource_not_found")
		return
	}
	var matches []api.Artifact
	for _, record := range s.state.Artifacts {
		a := record.Artifact
		if a.Scope == scope && a.RegistryID == id && a.Repository == q["repository"] && ((q["digest"] != "" && a.Digest == q["digest"]) || (q["tag"] != "" && slices.Contains(record.Tags, q["tag"]))) {
			matches = append(matches, a)
		}
	}
	if len(matches) == 0 {
		fail(w, 404, "resource_not_found")
		return
	}
	if len(matches) != 1 {
		fail(w, 409, "incomplete_result")
		return
	}
	send(w, 200, matches[0])
}
func query(r *http.Request) (map[string]string, error) {
	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return nil, errors.New("invalid query")
	}
	out := map[string]string{}
	for k, v := range values {
		if !slices.Contains([]string{"repository", "digest", "tag"}, k) || len(v) != 1 {
			return nil, errors.New("invalid query")
		}
		out[k] = v[0]
	}
	return out, nil
}

// SetUsage and SetSecretVersion model external data/secret owners, never payloads.
func (s *Server) SetUsage(scope api.Scope, kind, id string, bytes int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if bytes < 0 {
		panic("negative usage")
	}
	s.state.UsedBytes[objectPath(scope, kind, id)] = bytes
	s.persist()
}
func (s *Server) SetSecretVersion(spec api.SecretSpec, available bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Secrets[secretKey(spec)] = available
	s.persist()
}
func (s *Server) SetHealthFailure(id, reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if reason != "" && reason != "health_failed" && reason != "tls_failed" {
		panic("invalid synthetic failure")
	}
	s.state.Failures[id] = reason
	s.persist()
}
func refs(scope api.Scope, kind string, spec api.Spec) []string {
	out := []string{}
	if spec.Binding != nil {
		out = append(out, objectPath(scope, "container_service", spec.Binding.ConsumerID), objectPath(scope, spec.Binding.TargetKind, spec.Binding.TargetID))
	}
	if spec.Container != nil {
		c := spec.Container
		out = append(out, objectPath(scope, "artifact_registry", c.RegistryID), objectPath(api.Scope{OrganizationID: scope.OrganizationID}, "service_account", c.ServiceAccountID))
		for _, id := range c.SecretReferences {
			out = append(out, objectPath(scope, "secret_reference", id))
		}
	}
	if spec.Endpoint != nil {
		out = append(out, objectPath(scope, "container_service", spec.Endpoint.ServiceID))
	}
	return out
}
func (s *Server) checkService(scope api.Scope, kind string, req api.Intent, p Principal, skip string) string {
	path := objectPath(scope, kind, req.TargetID)
	old := s.state.Objects[path]
	if req.Kind == "delete" {
		for _, o := range s.state.Objects {
			if slices.Contains(refs(o.Scope, o.ResourceKind, o.Spec), path) {
				return "dependency_conflict"
			}
		}
		for id, op := range s.state.Operations {
			if id != skip && op.Status == "accepted" {
				spec := s.state.Intents[id].Spec
				if spec != nil && slices.Contains(refs(op.Scope, op.ResourceKind, *spec), path) {
					return "dependency_conflict"
				}
			}
		}
		if s.state.UsedBytes[path] > 0 {
			return "not_empty"
		}
		if kind == "artifact_registry" {
			for _, a := range s.state.Artifacts {
				if a.Artifact.Scope == scope && a.Artifact.RegistryID == old.ID {
					return "not_empty"
				}
			}
		}
		return ""
	}
	spec := *req.Spec
	if old.Storage != nil && spec.Storage == nil {
		return "invalid_request"
	}
	for _, target := range refs(scope, kind, spec) {
		if !s.can(p, target, "read") {
			return "forbidden"
		}
		if s.state.Objects[target].ID == "" {
			return "dependency_conflict"
		}
		for id, op := range s.state.Operations {
			if id != skip && op.Status == "accepted" && op.Kind == "delete" && objectPath(op.Scope, op.ResourceKind, op.TargetID) == target {
				return "dependency_conflict"
			}
		}
	}
	if spec.Registry != nil {
		if spec.Registry.CapacityBytes < s.state.UsedBytes[path] {
			return "quota_exceeded"
		}
		if old.Registry != nil && spec.Registry.RetentionDays < old.Registry.RetentionDays && s.hasArtifacts(scope, old.ID) {
			return "dependency_conflict"
		}
	}
	if spec.Storage != nil {
		if spec.Storage.CapacityBytes < s.state.UsedBytes[path] {
			return "quota_exceeded"
		}
		if old.Storage != nil {
			if spec.Storage.Profile != old.Storage.Profile {
				return "invalid_request"
			}
			if spec.Storage.CapacityBytes < old.Storage.CapacityBytes {
				return "invalid_request"
			}
			if spec.Storage.RetentionDays < old.Storage.RetentionDays && (s.state.UsedBytes[path] > 0 || s.state.Nonempty[old.ID]) {
				return "dependency_conflict"
			}
		}
	}
	if old.Secret != nil && (old.Secret.ExternalOwner != spec.Secret.ExternalOwner || old.Secret.ExternalID != spec.Secret.ExternalID) {
		return "invalid_request"
	}
	if old.Endpoint != nil && old.Endpoint.ServiceID != spec.Endpoint.ServiceID {
		return "invalid_request"
	}
	if spec.Binding != nil {
		b := spec.Binding
		if old.Binding != nil && (old.Binding.ConsumerID != b.ConsumerID || old.Binding.TargetID != b.TargetID || old.Binding.TargetKind != b.TargetKind || old.Binding.Slot != b.Slot) {
			return "invalid_request"
		}
		for _, o := range s.state.Objects {
			if o.Scope == scope && o.Binding != nil && o.ID != req.TargetID && o.Binding.ConsumerID == b.ConsumerID && o.Binding.Slot == b.Slot {
				return "dependency_conflict"
			}
		}
		for id, op := range s.state.Operations {
			i := s.state.Intents[id]
			if id != skip && op.Scope == scope && op.Status == "accepted" && i.Spec != nil && i.Spec.Binding != nil && op.TargetID != req.TargetID && i.Spec.Binding.ConsumerID == b.ConsumerID && i.Spec.Binding.Slot == b.Slot {
				return "dependency_conflict"
			}
		}
	}
	return ""
}
func (s *Server) hasArtifacts(scope api.Scope, id string) bool {
	for _, a := range s.state.Artifacts {
		if a.Artifact.Scope == scope && a.Artifact.RegistryID == id {
			return true
		}
	}
	return false
}
func (s *Server) declarationChanged(op api.Operation, previous api.Object) {
	path := objectPath(op.Scope, op.ResourceKind, op.TargetID)
	if op.Kind != "delete" && (api.ServiceKind(op.ResourceKind) || op.ResourceKind == "bucket") {
		o := s.state.Objects[path]
		active, effective := int64(0), int64(1)
		resolved := ""
		if o.Status != nil {
			active = o.Status.ActiveRevision
			resolved = o.Status.ResolvedVersion
			effective = o.Status.EffectiveRevision + 1
		}
		o.Status = &api.ServiceStatus{DesiredRevision: o.Revision, EffectiveRevision: effective, ActiveRevision: active, Readiness: "pending", OperationID: op.ID, ResolvedVersion: resolved}
		if o.Registry != nil || o.Storage != nil || o.ResourceKind == "bucket" || o.Endpoint != nil {
			o.Status.Endpoint = "https://" + o.ID + ".mock.invalid"
		}
		if o.ResourceKind == "database" {
			o.Status.Protocol = "transit-sql"
		}
		if o.ResourceKind == "bucket" {
			o.Status.Protocol = "object"
		}
		if o.Registry != nil {
			o.Status.Protocol = "oci"
		}
		if o.Endpoint != nil {
			o.Status.Protocol = "https"
			o.Status.TLSState = "pending"
		}
		s.state.Objects[path] = o
	}
	// Propagate dependency changes through bindings and consumers without changing
	// their Terraform-owned declarations. The old binding identifies a deleted slot.
	changed := map[string]bool{path: true}
	binding := previous.Binding
	if i := s.state.Intents[op.ID]; i.Spec != nil && i.Spec.Binding != nil {
		binding = i.Spec.Binding
	}
	if binding != nil {
		changed[objectPath(op.Scope, "container_service", binding.ConsumerID)] = true
	}
	for more := true; more; {
		more = false
		for key, o := range s.state.Objects {
			if changed[key] || o.Status == nil {
				continue
			}
			affected := op.ResourceKind == "grant" && o.OrganizationID == op.OrganizationID && (o.Container != nil || o.Binding != nil || o.Endpoint != nil)
			for _, dep := range refs(o.Scope, o.ResourceKind, o.Spec) {
				affected = affected || changed[dep]
			}
			if affected {
				changed[key] = true
				more = true
			}
			// A changed binding contributes to its consumer's effective configuration.
			if affected && o.Binding != nil {
				consumer := objectPath(o.Scope, "container_service", o.Binding.ConsumerID)
				if !changed[consumer] {
					changed[consumer] = true
					more = true
				}
			}
		}
	}
	for key := range changed {
		if key == path {
			continue
		}
		o, ok := s.state.Objects[key]
		if !ok || o.Status == nil {
			continue
		}
		o.Status.EffectiveRevision++
		o.Status.Readiness = "pending"
		o.Status.Reason = ""
		s.state.Objects[key] = o
	}
}

func (s *Server) runtimeAllowed(scope api.Scope, subject, target, action string) bool {
	return s.can(Principal{ID: subject, OrganizationID: scope.OrganizationID}, target, action)
}
func (s *Server) RuntimeAllowed(scope api.Scope, subject, kind, id, action string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.runtimeAllowed(scope, subject, objectPath(scope, kind, id), action)
}

// AdvanceControllers is test-only. GET cannot settle readiness, health, rotation or TLS.
func (s *Server) AdvanceControllers() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, kind := range []string{"artifact_registry", "database", "bucket", "secret_reference", "service_binding", "container_service", "endpoint"} {
		for key, o := range s.state.Objects {
			if o.ResourceKind != kind || o.Status == nil {
				continue
			}
			reason := s.state.Failures[o.ID]
			if reason == "" {
				reason = s.readinessReason(o)
			}
			o.Status.Readiness = "ready"
			o.Status.Reason = reason
			if reason != "" {
				o.Status.Readiness = "pending"
				if reason == "health_failed" || reason == "tls_failed" {
					o.Status.Readiness = "failed"
				}
			} else {
				o.Status.ActiveRevision = o.Status.EffectiveRevision
				if o.Secret != nil {
					o.Status.ResolvedVersion = o.Secret.Version
				}
			}
			if o.Endpoint != nil {
				o.Status.TLSState = o.Status.Readiness
			}
			s.state.Objects[key] = o
		}
	}
	s.state.Epoch++
	s.persist()
}
func (s *Server) readinessReason(o api.Object) string {
	if o.Secret != nil && !s.state.Secrets[secretKey(*o.Secret)] {
		return "dependency_unavailable"
	}
	if b := o.Binding; b != nil {
		consumer := s.state.Objects[objectPath(o.Scope, "container_service", b.ConsumerID)]
		target := s.state.Objects[objectPath(o.Scope, b.TargetKind, b.TargetID)]
		if consumer.Container == nil || target.Status == nil || target.Status.Readiness != "ready" {
			return "dependency_unavailable"
		}
		if !s.runtimeAllowed(o.Scope, consumer.Container.ServiceAccountID, objectPath(o.Scope, b.TargetKind, b.TargetID), b.Intent) {
			return "resolution_denied"
		}
	}
	if c := o.Container; c != nil {
		if _, ok := s.state.Artifacts[artifactKey(o.Scope, c.RegistryID, c.Repository, c.Digest)]; !ok {
			return "artifact_missing"
		}
		if !s.runtimeAllowed(o.Scope, c.ServiceAccountID, objectPath(o.Scope, "artifact_registry", c.RegistryID), "artifact_pull") {
			return "pull_denied"
		}
		for _, id := range c.SecretReferences {
			target := s.state.Objects[objectPath(o.Scope, "secret_reference", id)]
			if target.Status == nil || target.Status.Readiness != "ready" {
				return "dependency_unavailable"
			}
			if !s.runtimeAllowed(o.Scope, c.ServiceAccountID, objectPath(o.Scope, "secret_reference", id), "secret_resolve") {
				return "resolution_denied"
			}
		}
		slots := map[string]bool{}
		for _, b := range s.state.Objects {
			if b.Scope == o.Scope && b.Binding != nil && b.Binding.ConsumerID == o.ID {
				if b.Status == nil || b.Status.Readiness != "ready" {
					return "dependency_unavailable"
				}
				slots[b.Binding.Slot] = true
			}
		}
		for _, slot := range c.RequiredBindings {
			if !slots[slot] {
				return "dependency_unavailable"
			}
		}
	}
	if e := o.Endpoint; e != nil {
		target := s.state.Objects[objectPath(o.Scope, "container_service", e.ServiceID)]
		if target.Status == nil || target.Status.Readiness != "ready" {
			return "dependency_unavailable"
		}
	}
	return ""
}

// LoadV2Ledger is an explicit test-only migration, never an HTTP operation.
// Historical receipts keep their original fingerprint; v3 fingerprints are version-separated.
func (s *Server) LoadV2Ledger(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var legacy ledger
	if err := json.Unmarshal(data, &legacy); err != nil {
		return err
	}
	if legacy.Version != "mock-v2" {
		return errors.New("expected mock-v2 ledger")
	}
	for _, op := range legacy.Operations {
		if op.Status == "accepted" {
			return errors.New("recover pending v2 work before migration")
		}
	}
	objects := map[string]api.Object{}
	management := map[string]api.Management{}
	expired := map[string]string{}
	convert := func(p string) string { return strings.Replace(p, "/v2/organizations/", "/v3/organizations/", 1) }
	for key, o := range legacy.Objects {
		if o.Target != "" {
			o.Target = convert(o.Target)
		}
		objects[convert(key)] = o
	}
	for key, m := range legacy.Management {
		management[convert(key)] = m
	}
	for key, owner := range legacy.Expired {
		expired[convert(key)] = owner
	}
	legacy.Objects = objects
	legacy.Management = management
	legacy.Expired = expired
	legacy.Version = api.APIVersion
	s.state = legacy
	s.initServices()
	s.state.Epoch++
	s.persist()
	return nil
}

func (s *Server) Object(scope api.Scope, kind, id string) (api.Object, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.state.Objects[objectPath(scope, kind, id)]
	// Return an independent snapshot, including nested maps/pointers.
	b, _ := json.Marshal(o)
	var copy api.Object
	_ = json.Unmarshal(b, &copy)
	return copy, ok
}
func (s *Server) DebugCounts() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fmt.Sprintf("objects=%d operations=%d", len(s.state.Objects), len(s.state.Operations))
}
