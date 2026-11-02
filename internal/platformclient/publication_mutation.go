package platformclient

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// The caller supplies a stable key for exactly one logical mutation. This API
// never creates a recovery key, renews a revision fence or mutates runtime heads.
// A Terraform lifecycle adapter must establish recoverable identity before
// dispatch; a new random key held only in unsaved state cannot provide that.
func (c *PublicationClient) mutate(ctx context.Context, method, path, kind, schema, capability, key, targetID string, input any) (PublicationOperation, error) {
	var zero PublicationOperation
	payload, err := json.Marshal(input)
	if err != nil || len(payload) > 65536 || decodePublication(payload, schema, nil) != nil {
		return zero, errors.New("invalid publication mutation")
	}
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	caps, err := c.supports(ctx, capability)
	if err != nil {
		return zero, err
	}
	if !caps.Capabilities["operations"] {
		return zero, errors.New("publication operation recovery is unavailable")
	}
	valid := func(op PublicationOperation) bool {
		return c.validOperation(op) && op.Key == key && op.Kind == kind && op.PrincipalID == caps.PrincipalID && (targetID == "" || op.ResourceID == nil || *op.ResourceID == targetID)
	}
	var original *PublicationOperation
	for {
		// A lookup is observational. It alone cannot verify this request's full
		// intent because the server fingerprint uses a Platform-owned codec.
		// The exact HTTP replay verifies retained intent and principal server-side.
		prior, lookupErr := c.lookup(ctx, key)
		if lookupErr == nil {
			if !valid(prior) || original != nil && !samePublicationOperation(*original, prior) {
				return zero, errors.New("publication recovery identity mismatch")
			}
			original = &prior
		} else {
			var e *PublicationError
			absent := errors.As(lookupErr, &e) && e.Status == 404 && e.Code == "resource_not_found"
			if !absent || original != nil {
				if !publicationTransient(lookupErr) {
					return zero, lookupErr
				}
				if err := c.transport.wait(ctx, c.transport.poll); err != nil {
					return zero, err
				}
				continue
			}
		}
		var accepted PublicationOperation
		_, err = c.once(ctx, method, path, payload, "Operation", &accepted)
		if err != nil {
			var e *PublicationError
			if errors.As(err, &e) && e.Code == "outcome_unknown" && (e.Key != key || e.Lookup != c.base+"/operation-keys/"+key) {
				return zero, errors.New("publication recovery location mismatch")
			}
			if !publicationTransient(err) {
				return zero, err
			}
			if err := c.transport.wait(ctx, c.transport.poll); err != nil {
				return zero, err
			}
			continue
		}
		if !valid(accepted) || original != nil && !samePublicationOperation(*original, accepted) {
			return zero, errors.New("publication acceptance identity mismatch")
		}
		original = &accepted
		break
	}
	for {
		if original.Status != "accepted" {
			if original.Status != "applied" {
				return *original, &PublicationOutcomeError{Status: original.Status}
			}
			if original.ResourceID == nil {
				return *original, errors.New("applied publication declaration has no resource identity")
			}
			return *original, nil
		}
		if err := c.transport.wait(ctx, c.transport.poll); err != nil {
			return *original, err
		}
		next, err := c.operation(ctx, "/operations/"+original.ID)
		if err != nil {
			if publicationTransient(err) {
				continue
			}
			return *original, err
		}
		if !valid(next) || !samePublicationOperation(*original, next) {
			return *original, errors.New("publication settlement identity mismatch")
		}
		original = &next
	}
}

func samePublicationOperation(a, b PublicationOperation) bool {
	return a.ID == b.ID && a.Key == b.Key && a.Kind == b.Kind && a.PrincipalID == b.PrincipalID && a.Fingerprint == b.Fingerprint && a.Location == b.Location && (a.ResourceID == nil || b.ResourceID != nil && *a.ResourceID == *b.ResourceID)
}

type publicationPolicyRequest struct {
	Key string `json:"operation_key"`
	PublicationPolicy
}
type publicationUpdateRequest struct {
	publicationPolicyRequest
	Revision string `json:"expected_revision"`
}
type publicationDeleteRequest struct {
	Key      string `json:"operation_key"`
	Revision string `json:"expected_revision"`
}

func (c *PublicationClient) CreateApplication(ctx context.Context, key string, policy PublicationPolicy) (PublicationOperation, error) {
	return c.mutate(ctx, "POST", "/applications", "application_create", "CreateApplication", "applications", key, "", publicationPolicyRequest{key, policy})
}
func (c *PublicationClient) UpdateApplication(ctx context.Context, id, key, revision string, policy PublicationPolicy) (PublicationOperation, error) {
	if !publicationDecimal(id, false) {
		return PublicationOperation{}, errors.New("invalid publication application ID")
	}
	return c.mutate(ctx, "PATCH", "/applications/"+id, "application_update", "UpdatePolicy", "applications", key, id, publicationUpdateRequest{publicationPolicyRequest{key, policy}, revision})
}
func (c *PublicationClient) DeleteApplication(ctx context.Context, id, key, revision string) (PublicationOperation, error) {
	if !publicationDecimal(id, false) {
		return PublicationOperation{}, errors.New("invalid publication application ID")
	}
	return c.mutate(ctx, "DELETE", "/applications/"+id, "application_delete", "DeleteDeclaration", "applications", key, id, publicationDeleteRequest{key, revision})
}
func (c *PublicationClient) CreateBinding(ctx context.Context, applicationID, key, revision, environment string) (PublicationOperation, error) {
	if !publicationDecimal(applicationID, false) {
		return PublicationOperation{}, errors.New("invalid publication application ID")
	}
	request := struct {
		publicationDeleteRequest
		Environment string `json:"environment_id"`
	}{publicationDeleteRequest{key, revision}, environment}
	return c.mutate(ctx, "POST", "/applications/"+applicationID+"/target-bindings", "binding_create", "CreateBinding", "target_bindings", key, "", request)
}
func (c *PublicationClient) DeleteBinding(ctx context.Context, id, key, revision string) (PublicationOperation, error) {
	if !publicationDecimal(id, false) {
		return PublicationOperation{}, errors.New("invalid publication binding ID")
	}
	return c.mutate(ctx, "DELETE", "/target-bindings/"+id, "binding_delete", "DeleteDeclaration", "target_bindings", key, id, publicationDeleteRequest{key, revision})
}
func (c *PublicationClient) CreateTarget(ctx context.Context, bindingID, key string, policy PublicationPolicy) (PublicationOperation, error) {
	request := struct {
		publicationPolicyRequest
		Binding string `json:"binding_id"`
	}{publicationPolicyRequest{key, policy}, bindingID}
	return c.mutate(ctx, "POST", "/targets", "target_create", "CreateTarget", "targets", key, "", request)
}
func (c *PublicationClient) UpdateTarget(ctx context.Context, id, key, revision string, policy PublicationPolicy) (PublicationOperation, error) {
	if !publicationDecimal(id, false) {
		return PublicationOperation{}, errors.New("invalid publication target ID")
	}
	return c.mutate(ctx, "PATCH", "/targets/"+id, "target_update", "UpdatePolicy", "targets", key, id, publicationUpdateRequest{publicationPolicyRequest{key, policy}, revision})
}
func (c *PublicationClient) DeleteTarget(ctx context.Context, id, key, revision string) (PublicationOperation, error) {
	if !publicationDecimal(id, false) {
		return PublicationOperation{}, errors.New("invalid publication target ID")
	}
	return c.mutate(ctx, "DELETE", "/targets/"+id, "target_delete", "DeleteDeclaration", "targets", key, id, publicationDeleteRequest{key, revision})
}
