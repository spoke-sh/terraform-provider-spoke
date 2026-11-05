package platformclient

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

// GovernorClient is the HTTP client of a provisional management profile.
// It has no execution backend and never allocates a key or successor sequence.
type GovernorClient struct {
	transport *Client
	base      string
	mu        sync.Mutex
	principal string
}

func NewGovernor(cfg Config) (*GovernorClient, error) {
	if cfg.ManagementProfile != "" && cfg.ManagementProfile != GovernorProfile {
		return nil, errors.New("unsupported governor profile")
	}
	base := "/v1/ci/organizations/" + cfg.Scope.OrganizationID + "/projects/" + cfg.Scope.ProjectID + "/environments/" + cfg.Scope.EnvironmentID
	if !governorValue("Scope", base) {
		return nil, errors.New("governor requires an existing operator organization/project/environment scope")
	}
	cfg.ManagementProfile = ""
	c, err := New(cfg)
	if err != nil {
		return nil, err
	}
	return &GovernorClient{transport: c, base: base}, nil
}
func governorBound(ctx context.Context) (context.Context, context.CancelFunc) {
	budget := time.Minute
	if deadline, ok := ctx.Deadline(); ok {
		budget = min(time.Until(deadline), time.Hour)
	}
	return context.WithTimeout(ctx, budget)
}

var governorStatuses = map[string]int{
	"unauthenticated": 401, "forbidden": 403, "resource_not_found": 404, "request_not_found": 404, "unsupported_route": 404, "ownership_conflict": 409, "sequence_conflict": 409, "intent_conflict": 409, "revision_conflict": 409, "operation_pending": 409, "stale_snapshot": 409, "receipt_expired": 410, "body_too_large": 413, "invalid_request": 422, "unsupported_capability": 422, "capacity_exhausted": 429, "rate_limited": 429, "unavailable": 503, "outcome_unknown": 503, "incomplete_result": 503, "incompatible_api": 400,
}

func governorTransient(err error) bool {
	var e *GovernorError
	return errors.As(err, &e) && (e.Code == "transport_failed" || e.Code == "unavailable" || e.Code == "outcome_unknown" || e.Code == "rate_limited")
}
func governorAbsent(err error) bool {
	var e *GovernorError
	return errors.As(err, &e) && e.Code == "request_not_found" && e.Status == 404
}
func (c *GovernorClient) pause(ctx context.Context) error {
	return c.transport.wait(ctx, min(c.transport.poll, time.Second))
}
func governorPath(key string, seq uint64) string {
	return "/requests/" + key + "/" + strconv.FormatUint(seq, 10)
}
func invalidGovernor() error { return errors.New("governor response identity or contract mismatch") }

func (c *GovernorClient) once(ctx context.Context, method, path, kind string, payload []byte, out any) (int, error) {
	if len(payload) > governorBodyLimit {
		return 0, errors.New("governor request exceeds contract limit")
	}
	r, err := http.NewRequestWithContext(ctx, method, c.transport.endpoint+c.base+path, bytes.NewReader(payload))
	if err != nil {
		return 0, errors.New("invalid governor request")
	}
	r.Header.Set("Authorization", "Bearer "+c.transport.token)
	r.Header.Set("X-Spoke-Organization", c.transport.scope.OrganizationID)
	r.Header.Set("X-Spoke-API-Version", GovernorVersion)
	r.Header.Set("Accept", "application/json")
	if payload != nil {
		r.Header.Set("Content-Type", "application/json")
	}
	res, err := c.transport.http.Do(r)
	if err != nil {
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
		return 0, &GovernorError{Code: "transport_failed"}
	}
	defer res.Body.Close()
	bad := func() (int, error) {
		return res.StatusCode, &GovernorError{Code: "invalid_response", Status: res.StatusCode}
	}
	for k, v := range map[string]string{"X-Spoke-API-Version": GovernorVersion, "X-Spoke-Management-Profile": GovernorProfile, "Cache-Control": "no-store"} {
		if len(res.Header.Values(k)) != 1 || res.Header.Get(k) != v {
			return bad()
		}
	}
	media, _, err := mime.ParseMediaType(res.Header.Get("Content-Type"))
	if err != nil || media != "application/json" || len(res.Header.Values("Content-Type")) != 1 {
		return bad()
	}
	raw, err := io.ReadAll(io.LimitReader(res.Body, governorResponseLimit+1))
	if err != nil || len(raw) > governorResponseLimit {
		return bad()
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		var e GovernorError
		if decodeGovernor(raw, "Error", &e) != nil || governorStatuses[e.Code] != res.StatusCode {
			return bad()
		}
		e.Status = res.StatusCode
		return res.StatusCode, &e
	}
	allowed := res.StatusCode == 200 || method == "PUT" && res.StatusCode == 201 || method == "POST" && res.StatusCode == 202
	if !allowed || decodeGovernor(raw, kind, out) != nil {
		return bad()
	}
	if method != "GET" {
		op, ok := out.(*GovernorOperation)
		if !ok || len(res.Header.Values("Location")) != 1 || res.Header.Get("Location") != c.base+governorPath(op.ManagementKey, op.Sequence) {
			return bad()
		}
		if res.StatusCode == 201 && op.Status != "prepared" || res.StatusCode == 202 && op.Status != "accepted" {
			return bad()
		}
		if method == "POST" && res.StatusCode == 200 && (op.Status == "prepared" || op.Status == "accepted") {
			return bad()
		}
	}
	return res.StatusCode, nil
}
func (c *GovernorClient) get(ctx context.Context, path, kind string, out any) error {
	for i := 0; i < 3; i++ {
		_, err := c.once(ctx, "GET", path, kind, nil, out)
		if err == nil || !governorTransient(err) || i == 2 {
			return err
		}
		if err = c.pause(ctx); err != nil {
			return err
		}
	}
	panic("unreachable")
}
func (c *GovernorClient) capabilities(ctx context.Context) (GovernorCapabilities, error) {
	var caps GovernorCapabilities
	if err := c.get(ctx, "/capabilities", "Capabilities", &caps); err != nil {
		return caps, err
	}
	if caps.Scope != c.base {
		return GovernorCapabilities{}, invalidGovernor()
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.principal != "" && c.principal != caps.PrincipalID {
		return GovernorCapabilities{}, invalidGovernor()
	}
	c.principal = caps.PrincipalID
	return caps, nil
}
func (c *GovernorClient) Capabilities(ctx context.Context) (GovernorCapabilities, error) {
	ctx, cancel := governorBound(ctx)
	defer cancel()
	return c.capabilities(ctx)
}
func (c *GovernorClient) validOperation(op GovernorOperation, principal string) bool {
	if op.Scope != c.base || op.PrincipalID != principal {
		return false
	}
	fp, err := GovernorFingerprint(c.base, principal, op.Request())
	if err != nil || fp != op.Fingerprint {
		return false
	}
	if op.Intent.TargetID != nil && (op.ResourceID == nil || *op.ResourceID != *op.Intent.TargetID) {
		return false
	}
	if op.Intent.Kind == "create" && op.Status == "prepared" && op.ResourceID != nil {
		return false
	}
	return true
}
func (c *GovernorClient) operation(ctx context.Context, path, principal string) (GovernorOperation, error) {
	var op GovernorOperation
	if err := c.get(ctx, path, "Operation", &op); err != nil {
		return op, err
	}
	if !c.validOperation(op, principal) {
		return GovernorOperation{}, invalidGovernor()
	}
	return op, nil
}
func (c *GovernorClient) Request(ctx context.Context, key string, seq uint64) (GovernorOperation, error) {
	ctx, cancel := governorBound(ctx)
	defer cancel()
	if !governorValue("Key", key) || !governorValue("Counter", seq) {
		return GovernorOperation{}, errors.New("invalid governor request identity")
	}
	caps, err := c.capabilities(ctx)
	if err != nil {
		return GovernorOperation{}, err
	}
	op, err := c.operation(ctx, governorPath(key, seq), caps.PrincipalID)
	if err == nil && (op.ManagementKey != key || op.Sequence != seq) {
		return GovernorOperation{}, invalidGovernor()
	}
	return op, err
}
func (c *GovernorClient) Management(ctx context.Context, key string) (GovernorOperation, error) {
	ctx, cancel := governorBound(ctx)
	defer cancel()
	if !governorValue("Key", key) {
		return GovernorOperation{}, errors.New("invalid governor management key")
	}
	caps, err := c.capabilities(ctx)
	if err != nil {
		return GovernorOperation{}, err
	}
	op, err := c.operation(ctx, "/management/"+key, caps.PrincipalID)
	if err == nil && op.ManagementKey != key {
		return GovernorOperation{}, invalidGovernor()
	}
	return op, err
}
func validGovernor(g Governor, base, principal string) bool {
	return g.Scope == base && g.PrincipalID == principal && g.ActiveRevision <= g.Revision
}
func (c *GovernorClient) Governor(ctx context.Context, id string) (Governor, error) {
	ctx, cancel := governorBound(ctx)
	defer cancel()
	var g Governor
	if !governorValue("ID", id) {
		return g, errors.New("invalid governor ID")
	}
	caps, err := c.capabilities(ctx)
	if err != nil {
		return g, err
	}
	if err = c.get(ctx, "/governors/"+id, "Governor", &g); err != nil {
		return g, err
	}
	if g.ID != id || !validGovernor(g, c.base, caps.PrincipalID) {
		return Governor{}, invalidGovernor()
	}
	return g, nil
}

// Collections return either one complete checked snapshot or no items.
func (c *GovernorClient) Governors(ctx context.Context) ([]Governor, error) {
	ctx, cancel := governorBound(ctx)
	defer cancel()
	caps, err := c.capabilities(ctx)
	if err != nil {
		return nil, err
	}
	out := []Governor{}
	cursor := ""
	seen := map[string]bool{}
	var snapshot uint64
	last := ""
	for page := 0; page < min(4, c.transport.maxPages); page++ {
		path := "/governors"
		if cursor != "" {
			path += "?cursor=" + url.QueryEscape(cursor)
		}
		var p GovernorPage
		if err = c.get(ctx, path, "GovernorPage", &p); err != nil {
			return nil, err
		}
		if p.Scope != c.base || p.PrincipalID != caps.PrincipalID || page > 0 && p.Snapshot != snapshot {
			return nil, invalidGovernor()
		}
		snapshot = p.Snapshot
		for _, g := range p.Items {
			if g.ID <= last || !validGovernor(g, c.base, caps.PrincipalID) {
				return nil, invalidGovernor()
			}
			last = g.ID
			out = append(out, g)
		}
		if len(out) > min(32, c.transport.maxItems) {
			return nil, errors.New("governor collection exceeds bound")
		}
		if p.Complete {
			return out, nil
		}
		if seen[p.NextCursor] {
			return nil, errors.New("governor cursor repeated")
		}
		seen[p.NextCursor] = true
		cursor = p.NextCursor
	}
	return nil, errors.New("incomplete governor collection")
}
func (c *GovernorClient) Pending(ctx context.Context) ([]GovernorOperation, error) {
	ctx, cancel := governorBound(ctx)
	defer cancel()
	caps, err := c.capabilities(ctx)
	if err != nil {
		return nil, err
	}
	out := []GovernorOperation{}
	cursor := ""
	seen := map[string]bool{}
	keys := map[string]bool{}
	var snapshot uint64
	last := ""
	for page := 0; page < min(8, c.transport.maxPages); page++ {
		path := "/management?state=pending"
		if cursor != "" {
			path += "&cursor=" + url.QueryEscape(cursor)
		}
		var p GovernorPendingPage
		if err = c.get(ctx, path, "PendingPage", &p); err != nil {
			return nil, err
		}
		if p.Scope != c.base || p.PrincipalID != caps.PrincipalID || page > 0 && p.Snapshot != snapshot {
			return nil, invalidGovernor()
		}
		snapshot = p.Snapshot
		for _, op := range p.Items {
			if op.ID <= last || keys[op.ManagementKey] || !c.validOperation(op, caps.PrincipalID) || (op.Status != "prepared" && op.Status != "accepted") {
				return nil, invalidGovernor()
			}
			keys[op.ManagementKey] = true
			last = op.ID
			out = append(out, op)
		}
		if len(out) > min(64, c.transport.maxItems) {
			return nil, errors.New("pending governor collection exceeds bound")
		}
		if p.Complete {
			return out, nil
		}
		if seen[p.NextCursor] {
			return nil, errors.New("governor cursor repeated")
		}
		seen[p.NextCursor] = true
		cursor = p.NextCursor
	}
	return nil, errors.New("incomplete pending governor collection")
}
