package mockplatform

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"

	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

func (s *Server) check(scope api.Scope, kind string, req api.Intent, p Principal, skip string) string {
	if !s.can(p, scope.Path(), "manage") {
		return "forbidden"
	}
	if !s.parentExists(scope) {
		return "dependency_conflict"
	}
	if req.Kind == "create" {
		if limit, ok := s.quota[kind]; ok {
			n := 0
			for _, o := range s.state.Objects {
				if o.Scope == scope && o.ResourceKind == kind {
					n++
				}
			}
			for id, op := range s.state.Operations {
				if id != skip && op.Scope == scope && op.ResourceKind == kind && op.Kind == "create" && op.Status == "accepted" {
					n++
				}
			}
			if n >= limit {
				return "quota_exceeded"
			}
		}
	} else {
		o, ok := s.state.Objects[objectPath(scope, kind, req.TargetID)]
		if !ok {
			return "resource_not_found"
		}
		if o.OwnerID != p.ID || o.ManagementKey != req.ManagementKey {
			return "ownership_conflict"
		}
		if o.Revision != req.ExpectedRevision {
			return "revision_conflict"
		}
		if kind == "grant" && req.Kind == "update" && (o.SubjectID != req.Spec.SubjectID || o.Target != req.Spec.Target) {
			return "invalid_request"
		}
		if req.Kind == "delete" {
			if o.DeletionProtection {
				return "deletion_protected"
			}
			if s.state.Nonempty[o.ID] {
				return "not_empty"
			}
			target := objectPath(scope, kind, o.ID)
			for _, other := range s.state.Objects {
				if other.ID == o.ID && other.Scope == o.Scope && other.ResourceKind == kind {
					continue
				}
				if (kind == "service_account" && other.OwnerID == o.ID) || other.Scope.Path() == target || strings.HasPrefix(other.Scope.Path(), target+"/") || (other.ResourceKind == "grant" && (other.Target == target || strings.HasPrefix(other.Target, target+"/") || (kind == "service_account" && other.SubjectID == o.ID))) {
					return "dependency_conflict"
				}
			}
			for id, op := range s.state.Operations {
				if id != skip && op.Status == "accepted" && ((kind == "service_account" && op.OwnerID == o.ID) || op.Scope.Path() == target || strings.HasPrefix(op.Scope.Path(), target+"/")) {
					return "dependency_conflict"
				}
			}
		}
	}
	if kind == "grant" && req.Kind != "delete" {
		spec := req.Spec
		if s.state.Objects[objectPath(scope, "service_account", spec.SubjectID)].ID == "" {
			return "dependency_conflict"
		}
		root := scope.Path()
		if spec.Target != root {
			o, ok := s.state.Objects[spec.Target]
			if !ok || o.OrganizationID != scope.OrganizationID || o.ResourceKind == "grant" {
				return "dependency_conflict"
			}
		}
	}
	return ""
}
func errorStatus(code string) int {
	switch code {
	case "forbidden":
		return 403
	case "invalid_request":
		return 400
	case "resource_not_found":
		return 404
	case "quota_exceeded":
		return 429
	}
	return 409
}
func (s *Server) mutate(w http.ResponseWriter, r *http.Request, p Principal, scope api.Scope, kind string, parts []string) {
	if r.Header.Get("Content-Type") != "application/json" {
		fail(w, 400, "invalid_request")
		return
	}
	var req api.Mutation
	body, err := io.ReadAll(io.LimitReader(r.Body, 65537))
	if err != nil || len(body) > 65536 {
		fail(w, 400, "invalid_request")
		return
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if dec.Decode(&req) != nil || dec.Decode(new(any)) != io.EOF || !api.ValidID(req.ManagementKey) || req.ExpectedGeneration < 0 {
		fail(w, 400, "invalid_request")
		return
	}
	requested := req.ResourceKind
	if requested == "" {
		requested = "bucket"
	}
	action := map[string]string{"POST": "create", "PATCH": "update", "DELETE": "delete"}[r.Method]
	if kind != requested || req.Kind != action || (action == "create" && (req.TargetID != "" || req.ExpectedRevision != 0)) || (action != "create" && (req.TargetID != parts[1] || !api.ValidID(req.TargetID) || req.ExpectedRevision < 1)) {
		fail(w, 400, "invalid_request")
		return
	}
	if action != "delete" {
		spec := req.Spec
		if spec == nil || len(spec.Name) < 1 || utf8.RuneCountInString(spec.Name) > 63 || strings.TrimSpace(spec.Name) != spec.Name || strings.ContainsAny(spec.Name, "\r\n\t") {
			fail(w, 400, "invalid_request")
			return
		}
		if kind == "grant" {
			if !api.ValidID(spec.SubjectID) || (spec.Permission != "read" && spec.Permission != "manage") || spec.Target == "" {
				fail(w, 400, "invalid_request")
				return
			}
		} else if spec.SubjectID != "" || spec.Target != "" || spec.Permission != "" {
			fail(w, 400, "invalid_request")
			return
		}
	} else if req.Spec != nil {
		fail(w, 400, "invalid_request")
		return
	}
	if !s.pause(r, "before_accept", api.Operation{}) {
		return
	}
	p = s.currentPrincipal(p.ID, p.OrganizationID)
	if !s.can(p, scope.Path(), "manage") {
		fail(w, 403, "forbidden")
		return
	}
	// Compute the independent fixture oracle from the wire representation.
	encoded, _ := json.Marshal(req.Intent)
	hash := sha256.Sum256(encoded)
	fingerprint := hex.EncodeToString(hash[:])
	mk := receiptKey(scope, kind, req.ManagementKey)
	if owner, ok := s.state.Expired[mk]; ok {
		if owner != p.ID {
			fail(w, 409, "ownership_conflict")
		} else {
			fail(w, 410, "receipt_expired")
		}
		return
	}
	m := s.state.Management[mk]
	if m.OwnerID != "" && m.OwnerID != p.ID {
		fail(w, 409, "ownership_conflict")
		return
	}
	if m.Operation != nil && req.ExpectedGeneration == m.Generation-1 && m.Operation.Fingerprint == fingerprint {
		send(w, 202, m.Operation)
		return
	}
	if m.Generation != req.ExpectedGeneration {
		fail(w, 409, "generation_conflict")
		return
	}
	if m.Operation != nil && m.Operation.Status == "accepted" {
		fail(w, 409, "operation_pending")
		return
	}
	if action == "create" && m.ResourceID != "" {
		fail(w, 409, "ownership_conflict")
		return
	}
	if action != "create" && m.ResourceID != req.TargetID {
		fail(w, 409, "ownership_conflict")
		return
	}
	if code := s.check(scope, kind, req.Intent, p, ""); code != "" {
		fail(w, errorStatus(code), code)
		return
	}
	s.state.NextID++
	target := req.TargetID
	if action == "create" {
		target = fmt.Sprintf("%s-%d", kind, s.state.NextID)
	}
	op := api.Operation{Scope: scope, ID: fmt.Sprintf("op-%d", s.state.NextID), ManagementKey: req.ManagementKey, Fingerprint: fingerprint, Kind: action, TargetID: target, Generation: m.Generation + 1, Status: "accepted", ResourceKind: kind, OwnerID: p.ID}
	s.state.Operations[op.ID] = op
	s.state.Intents[op.ID] = req.Intent
	s.state.Management[mk] = api.Management{Scope: scope, ManagementKey: req.ManagementKey, Generation: op.Generation, ResourceID: m.ResourceID, Operation: &op, OwnerID: p.ID, ResourceKind: kind}
	s.persist()
	select {
	case s.accepted <- op:
	default:
	}
	if !s.pause(r, "after_accept", op) {
		return
	}
	if !s.hold {
		s.complete(op.ID)
	}
	if !s.pause(r, "after_complete", s.state.Operations[op.ID]) {
		return
	}
	if s.dropReply {
		s.dropReply = false
		conn, _, err := w.(http.Hijacker).Hijack()
		if err == nil {
			_ = conn.Close()
		}
		return
	}
	send(w, 202, op)
}
func (s *Server) complete(id string) {
	op := s.state.Operations[id]
	if op.Status != "accepted" {
		return
	}
	intent := s.state.Intents[id]
	mk := receiptKey(op.Scope, op.ResourceKind, op.ManagementKey)
	m := s.state.Management[mk]
	principal := Principal{Disabled: true}
	for _, p := range s.state.Principals {
		if p.ID == op.OwnerID && p.OrganizationID == op.OrganizationID && !p.Disabled {
			principal = p
			break
		}
	}
	if code := s.check(op.Scope, op.ResourceKind, intent, principal, id); code != "" {
		op.Status = "failed"
		op.ErrorCode = code
	} else {
		path := objectPath(op.Scope, op.ResourceKind, op.TargetID)
		switch op.Kind {
		case "create":
			s.state.Objects[path] = api.Object{Bucket: api.Bucket{Scope: op.Scope, BucketSpec: *intent.Spec, ID: op.TargetID, ManagementKey: op.ManagementKey, OwnerID: op.OwnerID, Revision: 1}, ResourceKind: op.ResourceKind}
			m.ResourceID = op.TargetID
		case "update":
			o := s.state.Objects[path]
			o.BucketSpec = *intent.Spec
			o.Revision++
			s.state.Objects[path] = o
		case "delete":
			delete(s.state.Objects, path)
			delete(s.state.Nonempty, op.TargetID)
			m.ResourceID = ""
		}
		op.Status = "succeeded"
		s.state.Epoch++
	}
	s.state.Operations[id] = op
	m.Operation = &op
	s.state.Management[mk] = m
	s.persist()
}
