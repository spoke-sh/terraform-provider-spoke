package platformclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

const maxResponseBytes = 1 << 20

type Config struct {
	Endpoint     string
	Scope        Scope
	Token        string
	CAPEM        []byte
	HTTPClient   *http.Client
	PollInterval time.Duration
	Wait         func(context.Context, time.Duration) error
	MaxPages     int
	MaxItems     int
}

type Client struct {
	scope              Scope
	endpoint           string
	token              string
	http               *http.Client
	poll               time.Duration
	wait               func(context.Context, time.Duration) error
	maxPages, maxItems int
}

func New(cfg Config) (*Client, error) {
	u, err := url.Parse(cfg.Endpoint)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return nil, errors.New("endpoint must be an HTTPS origin without credentials, path, query, or fragment")
	}
	if !cfg.Scope.Valid() {
		return nil, errors.New("scope requires a valid organization and optional project/environment hierarchy")
	}
	if cfg.Token == "" || strings.ContainsAny(cfg.Token, "\r\n\t ") {
		return nil, errors.New("a nonempty machine credential without whitespace is required")
	}
	hc := &http.Client{Timeout: 10 * time.Second}
	if cfg.HTTPClient != nil {
		*hc = *cfg.HTTPClient
	} else {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		roots, err := x509.SystemCertPool()
		if err != nil {
			return nil, errors.New("cannot load system TLS trust")
		}
		if len(cfg.CAPEM) > 0 && !roots.AppendCertsFromPEM(cfg.CAPEM) {
			return nil, errors.New("CA file contains no valid certificates")
		}
		transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots}
		hc.Transport = transport
	}
	// Credentials must never be forwarded to another origin, even by an injected client.
	hc.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	if hc.Timeout <= 0 {
		hc.Timeout = 10 * time.Second
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 100 * time.Millisecond
	}
	if cfg.Wait == nil {
		cfg.Wait = wait
	}
	if cfg.MaxPages <= 0 {
		cfg.MaxPages = 100
	}
	if cfg.MaxItems <= 0 {
		cfg.MaxItems = 10000
	}
	return &Client{scope: cfg.Scope, endpoint: strings.TrimRight(cfg.Endpoint, "/"), token: cfg.Token, http: hc, poll: cfg.PollInterval, wait: cfg.Wait, maxPages: cfg.MaxPages, maxItems: cfg.MaxItems}, nil
}

func (c *Client) Scope() Scope { return c.scope }
func (c *Client) base() string {
	return c.scope.Path()
}

func wait(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func (c *Client) request(ctx context.Context, method, path string, input, output any) error {
	var payload []byte
	if input != nil {
		var err error
		payload, err = json.Marshal(input)
		if err != nil {
			return errors.New("cannot encode platform request")
		}
	}
	for attempt := 0; attempt < 3; attempt++ {
		err, retryAfter := c.requestOnce(ctx, method, path, payload, output)
		if err == nil {
			return nil
		}
		var apiErr *APIError
		transient := !errors.As(err, &apiErr) || apiErr.Code == "rate_limited" || apiErr.Code == "unavailable"
		if method != http.MethodGet || !transient || attempt == 2 || ctx.Err() != nil {
			return err
		}
		delay := c.poll * time.Duration(1<<attempt)
		if retryAfter > delay {
			delay = retryAfter
		}
		if err := c.wait(ctx, delay); err != nil {
			return err
		}
	}
	panic("unreachable")
}

func (c *Client) requestOnce(ctx context.Context, method, path string, payload []byte, output any) (error, time.Duration) {
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, bytes.NewReader(payload))
	if err != nil {
		return errors.New("cannot construct platform request"), 0
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("X-Spoke-Organization", c.scope.OrganizationID)
	req.Header.Set("X-Spoke-API-Version", APIVersion)
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err(), 0
		}
		// net/http errors may include URLs and sensitive remote data.
		return errors.New("platform transport failed; acceptance may be uncertain"), 0
	}
	defer res.Body.Close()
	if res.Header.Get("X-Spoke-API-Version") != APIVersion {
		return &APIError{Code: "incompatible_api", Status: res.StatusCode}, 0
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, maxResponseBytes+1))
	if err != nil || len(body) > maxResponseBytes {
		return &APIError{Code: "invalid_response", Status: res.StatusCode}, 0
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		var e APIError
		if json.Unmarshal(body, &e) != nil || !knownError(e.Code) {
			e.Code = "invalid_response"
		}
		e.Status = res.StatusCode
		// Absence is meaningful only for this exact typed status combination.
		if (e.Code == "resource_not_found" || e.Code == "management_not_found") && e.Status != 404 {
			e.Code = "invalid_response"
		}
		var retry time.Duration
		if seconds, err := strconv.Atoi(res.Header.Get("Retry-After")); err == nil && seconds > 0 {
			if seconds > 10 {
				seconds = 10
			}
			retry = time.Duration(seconds) * time.Second
		}
		return &e, retry
	}
	if output == nil {
		return nil, 0
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(output); err != nil || dec.Decode(new(any)) != io.EOF {
		return &APIError{Code: "invalid_response", Status: res.StatusCode}, 0
	}
	return nil, 0
}

func knownError(code string) bool {
	switch code {
	case "invalid_request", "unauthenticated", "forbidden", "resource_not_found", "management_not_found", "revision_conflict", "generation_conflict", "ownership_conflict", "operation_pending", "deletion_protected", "not_empty", "dependency_conflict", "quota_exceeded", "rate_limited", "unavailable", "unsupported_route", "incompatible_api", "receipt_expired", "incomplete_result":
		return true
	}
	return false
}

func (c *Client) GetProject(ctx context.Context) (Project, error) {
	parent := *c
	parent.scope = Scope{OrganizationID: c.scope.OrganizationID}
	o, err := parent.GetObject(ctx, "project", c.scope.ProjectID)
	return Project{ID: o.ID, OrganizationID: o.OrganizationID, Name: o.Name}, err
}

func (c *Client) GetEnvironment(ctx context.Context) (Environment, error) {
	parent := *c
	parent.scope.EnvironmentID = ""
	o, err := parent.GetObject(ctx, "environment", c.scope.EnvironmentID)
	return Environment{ID: o.ID, OrganizationID: o.OrganizationID, ProjectID: o.ProjectID, Name: o.Name}, err
}

func (c *Client) GetBucket(ctx context.Context, id string) (Bucket, error) {
	o, err := c.GetObject(ctx, "bucket", id)
	return o.Bucket, err
}

func (c *Client) management(ctx context.Context, kind, key, owner string) (Management, error) {
	var m Management
	err := c.request(ctx, "GET", c.base()+"/management/"+kind+"/"+key, nil, &m)
	if IsCode(err, "management_not_found") {
		return Management{Scope: c.scope, ManagementKey: key}, nil
	}
	if err == nil && (m.Scope != c.scope || m.ManagementKey != key || m.OwnerID != owner || m.ResourceKind != kind || m.Generation < 1 || m.Operation == nil || m.Operation.Generation != m.Generation || m.Operation.OwnerID != owner || m.Operation.ResourceKind != kind || !c.validOperation(*m.Operation, key)) {
		err = errors.New("invalid management receipt")
	}
	return m, err
}

func (c *Client) validOperation(op Operation, key string) bool {
	return op.Scope == c.scope && op.ManagementKey == key && ValidID(op.OwnerID) && Collection(op.ResourceKind) != "" && ValidID(op.ID) && ValidID(op.TargetID) && op.Generation > 0 && fingerprintPattern.MatchString(op.Fingerprint) && (op.Status == "accepted" || op.Status == "succeeded" || op.Status == "failed") && (op.Kind == "create" || op.Kind == "update" || op.Kind == "delete")
}

func (c *Client) settle(ctx context.Context, op Operation) (Operation, error) {
	original := op
	for {
		switch op.Status {
		case "succeeded":
			return op, nil
		case "failed":
			code := op.ErrorCode
			if !knownError(code) {
				code = "operation_failed"
			}
			return op, fmt.Errorf("operation %s failed: %s", op.ID, code)
		}
		if err := c.wait(ctx, c.poll); err != nil {
			return op, &PendingError{Operation: original, Cause: err}
		}
		var next Operation
		if err := c.request(ctx, "GET", c.base()+"/operations/"+op.ID, nil, &next); err != nil {
			return op, &PendingError{Operation: original, Cause: err}
		}
		if !c.validOperation(next, original.ManagementKey) || next.ID != original.ID || next.Fingerprint != original.Fingerprint || next.Generation != original.Generation || next.Kind != original.Kind || next.TargetID != original.TargetID || next.OwnerID != original.OwnerID || next.ResourceKind != original.ResourceKind {
			return op, &PendingError{Operation: original, Cause: errors.New("operation response changed identity")}
		}
		op = next
	}
}

// Mutate recovers a matching original receipt before dispatch. The server CASes
// the management generation and object revision before accepting new work.
func (c *Client) Mutate(ctx context.Context, intent Intent) (Operation, error) {
	kind := intent.ResourceKind
	if kind == "" {
		kind = "bucket"
	}
	if !c.scope.AllowsKind(kind) {
		return Operation{}, errors.New("resource kind requires its explicit parent scope")
	}
	if !ValidID(intent.ManagementKey) || (intent.Kind != "create" && intent.Kind != "update" && intent.Kind != "delete") {
		return Operation{}, errors.New("invalid mutation identity")
	}
	if intent.Kind != "create" && (!ValidID(intent.TargetID) || intent.ExpectedRevision < 1) {
		return Operation{}, errors.New("mutation requires a stable target and revision")
	}
	caps, err := c.Capabilities(ctx)
	if err != nil {
		return Operation{}, err
	}
	if !slices.Contains(caps.Resources, kind) {
		return Operation{}, errors.New("target does not support this resource kind")
	}
	m, err := c.management(ctx, kind, intent.ManagementKey, caps.PrincipalID)
	if err != nil {
		return Operation{}, err
	}
	fingerprint := intent.Fingerprint()
	if m.Operation != nil {
		if m.Operation.Fingerprint == fingerprint && !(intent.Kind == "create" && m.Operation.Status == "succeeded" && m.ResourceID == "") {
			return c.settle(ctx, *m.Operation)
		}
		if m.Operation.Status == "accepted" {
			return *m.Operation, &PendingError{Operation: *m.Operation, Cause: errors.New("resolve the pending operation before submitting different intent")}
		}
		if intent.Kind == "create" && m.ResourceID != "" {
			return Operation{}, &APIError{Code: "ownership_conflict", Status: 409}
		}
	}
	method, path := "POST", c.base()+"/"+Collection(kind)
	if intent.Kind == "update" {
		method, path = "PATCH", path+"/"+intent.TargetID
	} else if intent.Kind == "delete" {
		method, path = "DELETE", path+"/"+intent.TargetID
	}
	var op Operation
	err = c.request(ctx, method, path, Mutation{Intent: intent, ExpectedGeneration: m.Generation}, &op)
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.Status < 500 && apiErr.Code != "invalid_response" {
			return Operation{}, err
		}
		// A lost reply may follow durable acceptance. Inspect instead of sending
		// another mutation, and preserve uncertainty if inspection also fails.
		recovered, lookupErr := c.management(ctx, kind, intent.ManagementKey, caps.PrincipalID)
		if lookupErr == nil && recovered.Operation != nil && recovered.Operation.Fingerprint == fingerprint {
			return c.settle(ctx, *recovered.Operation)
		}
		return Operation{}, fmt.Errorf("mutation outcome uncertain for management key %s; retry with the same configuration to inspect its receipt", intent.ManagementKey)
	}
	if !c.validOperation(op, intent.ManagementKey) || op.OwnerID != caps.PrincipalID || op.ResourceKind != kind || op.Fingerprint != fingerprint || op.Kind != intent.Kind || op.Generation != m.Generation+1 || (intent.TargetID != "" && op.TargetID != intent.TargetID) {
		return Operation{}, errors.New("invalid acceptance receipt; recover using the original management key")
	}
	return c.settle(ctx, op)
}
