package platformclient

import (
	"context"
	"encoding/json"
	"errors"
)

func sameGovernorOperation(a, b GovernorOperation) bool {
	if a.ID != b.ID || a.Scope != b.Scope || a.PrincipalID != b.PrincipalID || a.ManagementKey != b.ManagementKey || a.Sequence != b.Sequence || a.Fingerprint != b.Fingerprint {
		return false
	}
	if a.ResourceID != nil && (b.ResourceID == nil || *a.ResourceID != *b.ResourceID) {
		return false
	}
	if a.Status == "accepted" && (b.Status == "prepared" || b.Status == "aborted") {
		return false
	}
	if a.Status == "applied" || a.Status == "refused" || a.Status == "aborted" {
		return a.Status == b.Status && a.Reason == b.Reason
	}
	return true
}
func governorOutcome(op GovernorOperation) (GovernorOperation, error) {
	if op.Status == "applied" {
		return op, nil
	}
	return op, &GovernorOutcomeError{Status: op.Status, Reason: op.Reason}
}

func (c *GovernorClient) prepare(ctx context.Context, request GovernorRequest, principal string) (GovernorOperation, error) {
	var zero GovernorOperation
	fp, err := GovernorFingerprint(c.base, principal, request)
	if err != nil {
		return zero, err
	}
	payload, _ := json.Marshal(request.Intent)
	path := governorPath(request.ManagementKey, request.Sequence)
	valid := func(op GovernorOperation) bool {
		return c.validOperation(op, principal) && op.Fingerprint == fp && op.ManagementKey == request.ManagementKey && op.Sequence == request.Sequence
	}
	for attempt := 0; attempt < 128; attempt++ {
		op, err := c.operation(ctx, path, principal)
		if err == nil {
			if !valid(op) {
				return zero, invalidGovernor()
			}
			return op, nil
		}
		if !governorAbsent(err) {
			if !governorTransient(err) {
				return zero, err
			}
		} else {
			_, err = c.once(ctx, "PUT", path, "Operation", payload, &op)
			if err == nil {
				if !valid(op) {
					return zero, invalidGovernor()
				}
				return op, nil
			}
			if !governorTransient(err) {
				return zero, err
			}
		}
		if err = c.pause(ctx); err != nil {
			return zero, err
		}
	}
	return zero, errors.New("governor preparation recovery bound exceeded")
}

// Prepare durably stages the caller's explicit identity and intent. It does not
// commit policy. A later process can recover it with only the management key.
func (c *GovernorClient) Prepare(ctx context.Context, request GovernorRequest) (GovernorOperation, error) {
	ctx, cancel := governorBound(ctx)
	defer cancel()
	if !governorValue("Request", request) {
		return GovernorOperation{}, errors.New("invalid governor intent")
	}
	caps, err := c.capabilities(ctx)
	if err != nil {
		return GovernorOperation{}, err
	}
	return c.prepare(ctx, request, caps.PrincipalID)
}

// Apply recovers or stages exactly this sequence, commits it and waits within
// the caller's deadline. It never chooses a successor or changes a revision.
func (c *GovernorClient) Apply(ctx context.Context, request GovernorRequest) (GovernorOperation, error) {
	ctx, cancel := governorBound(ctx)
	defer cancel()
	if !governorValue("Request", request) {
		return GovernorOperation{}, errors.New("invalid governor intent")
	}
	caps, err := c.capabilities(ctx)
	if err != nil {
		return GovernorOperation{}, err
	}
	op, err := c.prepare(ctx, request, caps.PrincipalID)
	if err != nil {
		return op, err
	}
	path := governorPath(request.ManagementKey, request.Sequence)
	payload, _ := json.Marshal(map[string]string{"fingerprint": op.Fingerprint})
	for attempt := 0; attempt < 128; attempt++ {
		if op.Status != "prepared" && op.Status != "accepted" {
			return governorOutcome(op)
		}
		var next GovernorOperation
		if op.Status == "prepared" {
			_, err = c.once(ctx, "POST", path+"/commit", "Operation", payload, &next)
			if err != nil {
				if !governorTransient(err) {
					return op, err
				}
				if err = c.pause(ctx); err != nil {
					return op, err
				}
				// This known retained request may not become absent. Never stage a
				// replacement after an uncertain commit or release its ownership.
				next, err = c.operation(ctx, path, caps.PrincipalID)
			}
		} else {
			if err = c.pause(ctx); err != nil {
				return op, err
			}
			next, err = c.operation(ctx, path, caps.PrincipalID)
		}
		if err != nil {
			if governorTransient(err) {
				continue
			}
			return op, err
		}
		if !c.validOperation(next, caps.PrincipalID) || !sameGovernorOperation(op, next) {
			return op, invalidGovernor()
		}
		op = next
	}
	return op, errors.New("governor settlement recovery bound exceeded")
}

// Resume acts only on the latest server-retained intent. Choosing a later
// desired policy/sequence is explicitly a caller responsibility.
func (c *GovernorClient) Resume(ctx context.Context, key string) (GovernorOperation, error) {
	ctx, cancel := governorBound(ctx)
	defer cancel()
	op, err := c.Management(ctx, key)
	if err != nil {
		return op, err
	}
	if op.Status != "prepared" && op.Status != "accepted" {
		return governorOutcome(op)
	}
	return c.Apply(ctx, op.Request())
}

// Abort only abandons an explicitly prepared request. It never stages an absent
// request, automatically runs on timeout, or rolls back accepted work.
func (c *GovernorClient) Abort(ctx context.Context, request GovernorRequest) (GovernorOperation, error) {
	ctx, cancel := governorBound(ctx)
	defer cancel()
	if !governorValue("Request", request) {
		return GovernorOperation{}, errors.New("invalid governor intent")
	}
	caps, err := c.capabilities(ctx)
	if err != nil {
		return GovernorOperation{}, err
	}
	fp, _ := GovernorFingerprint(c.base, caps.PrincipalID, request)
	path := governorPath(request.ManagementKey, request.Sequence)
	payload, _ := json.Marshal(map[string]string{"fingerprint": fp})
	var original *GovernorOperation
	for attempt := 0; attempt < 128; attempt++ {
		op, err := c.operation(ctx, path, caps.PrincipalID)
		if err != nil {
			if governorTransient(err) {
				if err = c.pause(ctx); err != nil {
					return GovernorOperation{}, err
				}
				continue
			}
			return GovernorOperation{}, err
		}
		if op.Fingerprint != fp || op.ManagementKey != request.ManagementKey || op.Sequence != request.Sequence || original != nil && !sameGovernorOperation(*original, op) {
			return op, invalidGovernor()
		}
		if op.Status == "aborted" {
			return op, nil
		}
		if op.Status != "prepared" {
			return op, &GovernorError{Code: "operation_pending", Status: 409}
		}
		original = &op
		var next GovernorOperation
		_, err = c.once(ctx, "POST", path+"/abort", "Operation", payload, &next)
		if err == nil {
			if !c.validOperation(next, caps.PrincipalID) || !sameGovernorOperation(op, next) || next.Status != "aborted" {
				return op, invalidGovernor()
			}
			return next, nil
		}
		if !governorTransient(err) {
			return op, err
		}
		if err = c.pause(ctx); err != nil {
			return op, err
		}
	}
	return GovernorOperation{}, errors.New("governor abort recovery bound exceeded")
}
