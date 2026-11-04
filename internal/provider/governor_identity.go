package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"regexp"
	"strings"

	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

const governorPrivateKey = "governor_mutation_v1"

var governorIDPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,62}$`)
var governorKeyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$`)

type governorReader interface {
	Capabilities(context.Context) (api.GovernorCapabilities, error)
	Management(context.Context, string) (api.GovernorOperation, error)
	Governor(context.Context, string) (api.Governor, error)
}

type governorView struct {
	Scope, Owner string
	Latest       *api.GovernorOperation
	Value        *api.Governor
}

// This is a saved-plan fence, never a pre-dispatch journal. Configuration owns
// the stable key; Platform retains the original request before commit.
type governorFence struct {
	Version          int                 `json:"version"`
	Scope            string              `json:"scope"`
	Owner            string              `json:"owner"`
	Request          api.GovernorRequest `json:"request"`
	Observe          bool                `json:"observe"`
	ObservedRevision uint64              `json:"observed_revision"`
	ObservedPolicy   *api.GovernorPolicy `json:"observed_policy"`
}

type governorAnchor struct {
	Owner, Target string
	Sequence      uint64
	Revision      uint64
}

func governorIs(err error, code string) bool {
	var e *api.GovernorError
	return errors.As(err, &e) && e.Code == code && (code != "resource_not_found" && code != "request_not_found" || e.Status == 404)
}
func governorPending(op *api.GovernorOperation) bool {
	return op != nil && (op.Status == "prepared" || op.Status == "accepted")
}
func governorPolicyEqual(a, b *api.GovernorPolicy) bool { return reflect.DeepEqual(a, b) }
func governorIntentMatches(i api.GovernorIntent, kind, target string, policy *api.GovernorPolicy) bool {
	return i.Kind == kind && (i.TargetID == nil && target == "" || i.TargetID != nil && *i.TargetID == target) && governorPolicyEqual(i.Policy, policy)
}
func governorStale() error {
	return errors.New("governor intent or saved-plan fence conflicts; retain the management_key, recover the original operation, then refresh and replan")
}
func governorCanonical(scope api.Scope, id string) string {
	return strings.Join([]string{scope.OrganizationID, scope.ProjectID, scope.EnvironmentID, id}, "/")
}
func governorParse(scope api.Scope, id string) (string, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 4 || id != governorCanonical(scope, parts[3]) {
		return "", errors.New("governor ID must match the configured organization/project/environment/governor_id")
	}
	for _, part := range parts {
		if !governorIDPattern.MatchString(part) {
			return "", errors.New("invalid governor scoped identity")
		}
	}
	return parts[3], nil
}

// Two matching management reads bracket the resource projection. A moving cut
// is retried a bounded number of times; GET never settles or repairs anything.
func governorInspect(ctx context.Context, c governorReader, key string) (governorView, error) {
	var v governorView
	if !governorKeyPattern.MatchString(key) {
		return v, errors.New("invalid governor management_key")
	}
	caps, err := c.Capabilities(ctx)
	if err != nil {
		return v, err
	}
	v.Scope, v.Owner = caps.Scope, caps.PrincipalID
	for range 3 {
		v.Latest, v.Value = nil, nil
		op, err := c.Management(ctx, key)
		if governorIs(err, "request_not_found") {
			return v, nil
		}
		if err != nil {
			return v, err
		}
		if op.Scope != v.Scope || op.PrincipalID != v.Owner || op.ManagementKey != key {
			return v, governorStale()
		}
		v.Latest = &op
		if op.ResourceID != nil {
			g, err := c.Governor(ctx, *op.ResourceID)
			if err != nil && !governorIs(err, "resource_not_found") {
				return v, err
			}
			if err == nil {
				if g.Scope != v.Scope || g.PrincipalID != v.Owner || g.ManagementKey != key || g.ID != *op.ResourceID {
					return v, governorStale()
				}
				v.Value = &g
			}
		}
		next, err := c.Management(ctx, key)
		if err != nil {
			return v, err
		}
		if reflect.DeepEqual(op, next) {
			return v, nil
		}
	}
	return v, errors.New("governor management snapshot changed repeatedly; retry observation without submitting new intent")
}

func governorSelect(v governorView, key, kind string, policy *api.GovernorPolicy, prior *governorAnchor) (governorFence, error) {
	f := governorFence{Version: 1, Scope: v.Scope, Owner: v.Owner}
	f.Request.ManagementKey = key
	if !governorKeyPattern.MatchString(key) || kind != "create" && kind != "update" && kind != "retire" {
		return f, governorStale()
	}
	target := ""
	if prior != nil {
		target = prior.Target
		if prior.Owner != v.Owner || v.Latest == nil || v.Latest.ResourceID == nil || *v.Latest.ResourceID != target {
			return f, governorStale()
		}
	}
	if kind != "create" && prior == nil || kind == "retire" && policy != nil || kind != "retire" && policy == nil {
		return f, governorStale()
	}
	if v.Latest != nil {
		op := v.Latest
		if prior != nil && (op.Sequence < prior.Sequence || op.Sequence > prior.Sequence+1) {
			return f, governorStale()
		}
		if governorPending(op) {
			if !governorIntentMatches(op.Intent, kind, target, policy) {
				return f, errors.New("governor has a different prepared/accepted intent; restore its configuration to resume it; no successor or automatic abort is permitted")
			}
			f.Request = op.Request()
			return f, nil
		}
		if op.Status == "applied" && op.Intent.Kind == "retire" {
			if kind != "retire" || v.Value != nil {
				return f, errors.New("governor management_key is retired; recreation requires a new configured key")
			}
			f.Request = op.Request()
			return f, nil
		}
		if prior != nil && op.Sequence != prior.Sequence {
			if op.Status != "applied" || !governorIntentMatches(op.Intent, kind, target, policy) {
				return f, governorStale()
			}
			f.Request = op.Request()
			return f, nil
		}
		if v.Value != nil && kind != "retire" && governorPolicyEqual(&v.Value.Policy, policy) {
			f.Request = op.Request()
			f.Observe, f.ObservedRevision, f.ObservedPolicy = true, v.Value.Revision, policy
			return f, nil
		}
		if kind == "create" && op.ResourceID != nil {
			return f, errors.New("management_key already owns a governor; import its scoped ID or restore the accepted policy before recovering creation")
		}
		if op.Sequence == 4294967295 {
			return f, errors.New("governor sequence exhausted; retained identities cannot be recycled")
		}
		f.Request.Sequence = op.Sequence + 1
	}
	intent := api.GovernorIntent{Kind: kind, Policy: policy}
	if kind != "create" {
		if v.Value == nil || v.Value.ID != target || v.Value.Revision != prior.Revision {
			return f, governorStale()
		}
		if kind == "retire" && v.Value.Policy.DeletionProtection {
			return f, errors.New("governor deletion protected; apply deletion_protection = false before retirement")
		}
		if kind == "update" && policy.AdmissionPolicyID != v.Value.Policy.AdmissionPolicyID {
			return f, errors.New("admission entitlement is immutable; retire this governor and use a new management_key")
		}
		intent.TargetID, intent.ExpectedRevision = &target, prior.Revision
	}
	f.Request.Intent = intent
	return f, nil
}

func governorDecodeFence(raw []byte) (governorFence, error) {
	var f governorFence
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(&f) != nil || d.Decode(new(any)) != io.EOF || f.Version != 1 {
		return f, errors.New("missing or incompatible governor plan identity; refresh and replan before mutation")
	}
	if _, err := api.GovernorFingerprint(f.Scope, f.Owner, f.Request); err != nil || f.Observe && (f.ObservedPolicy == nil || f.ObservedRevision == 0) {
		return f, governorStale()
	}
	return f, nil
}

// Apply checks the frozen plan against the current owner. It never calls the
// selector again to choose another sequence or revision after uncertainty.
func governorCheckFence(v governorView, f governorFence) error {
	if f.Version != 1 || v.Scope != f.Scope || v.Owner != f.Owner {
		return governorStale()
	}
	r := f.Request
	if v.Latest != nil && v.Latest.Sequence == r.Sequence {
		fp, err := api.GovernorFingerprint(f.Scope, f.Owner, r)
		if err != nil || v.Latest.Fingerprint != fp {
			return governorStale()
		}
		if f.Observe && (v.Value == nil || v.Value.Revision != f.ObservedRevision || !governorPolicyEqual(&v.Value.Policy, f.ObservedPolicy)) {
			return governorStale()
		}
		return nil
	}
	if f.Observe || v.Latest == nil && (r.Sequence != 0 || r.Intent.Kind != "create") || v.Latest != nil && (governorPending(v.Latest) || v.Latest.Sequence == 4294967295 || v.Latest.Sequence+1 != r.Sequence || v.Latest.Intent.Kind == "retire" && v.Latest.Status == "applied") {
		return governorStale()
	}
	if r.Intent.TargetID != nil {
		if v.Value == nil || v.Value.ID != *r.Intent.TargetID || v.Value.Revision != r.Intent.ExpectedRevision {
			return governorStale()
		}
	} else if v.Value != nil || v.Latest != nil && v.Latest.ResourceID != nil {
		return governorStale()
	}
	return nil
}
