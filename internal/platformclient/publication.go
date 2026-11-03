package platformclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

// PublicationClient is the separately negotiated declaration HTTP client. It
// exposes no native protocol, file upload, runtime promotion or fixture profile.
type PublicationClient struct {
	transport *Client
	base      string
	mu        sync.Mutex
	principal string
}

func NewPublication(cfg Config) (*PublicationClient, error) {
	if cfg.ManagementProfile != "" && cfg.ManagementProfile != PublicationProfile {
		return nil, errors.New("unsupported publication profile")
	}
	if cfg.Scope.ProjectID == "" || len(cfg.Scope.EnvironmentID) > 63 {
		return nil, errors.New("publication requires project scope and a bounded environment identifier")
	}
	cfg.ManagementProfile = ""
	c, err := New(cfg)
	if err != nil {
		return nil, err
	}
	base := "/v1/publication/organizations/" + cfg.Scope.OrganizationID + "/projects/" + cfg.Scope.ProjectID
	if cfg.Scope.EnvironmentID != "" {
		base += "/environments/" + cfg.Scope.EnvironmentID
	}
	return &PublicationClient{transport: c, base: base}, nil
}

func (c *PublicationClient) Scope() Scope { return c.transport.scope }

func publicationTransient(err error) bool {
	var e *PublicationError
	return errors.As(err, &e) && (e.Code == "transport_failed" || e.Code == "unavailable" || e.Code == "rate_limited" || e.Code == "incomplete_result" || e.Code == "outcome_unknown")
}

func publicationErrorStatus(code string, status int) bool {
	expected := map[string]int{
		"unauthenticated": 401, "forbidden": 403, "resource_not_found": 404, "unsupported_route": 404,
		"operation_conflict": 409, "stale_snapshot": 409, "receipt_expired": 410, "body_too_large": 413,
		"invalid_request": 422, "unsupported_capability": 422, "rate_limited": 429, "capacity_exhausted": 429,
		"outcome_unknown": 503, "incomplete_result": 503, "unavailable": 503, "incompatible_api": 400,
	}
	return expected[code] == status && status != 0
}

func (c *PublicationClient) once(ctx context.Context, method, path string, payload []byte, kind string, out any) (int, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.transport.endpoint+c.base+path, bytes.NewReader(payload))
	if err != nil {
		return 0, errors.New("invalid publication request")
	}
	req.Header.Set("Authorization", "Bearer "+c.transport.token)
	req.Header.Set("X-Spoke-Organization", c.Scope().OrganizationID)
	req.Header.Set("X-Spoke-API-Version", PublicationVersion)
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	response, err := c.transport.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
		return 0, &PublicationError{Code: "transport_failed"}
	}
	defer response.Body.Close()
	invalid := func(code string) (int, error) {
		return response.StatusCode, &PublicationError{Code: code, Status: response.StatusCode}
	}
	for key, want := range map[string]string{"X-Spoke-API-Version": PublicationVersion, "X-Spoke-Management-Profile": PublicationProfile, "Cache-Control": "no-store"} {
		if len(response.Header.Values(key)) != 1 || response.Header.Get(key) != want {
			return invalid("incompatible_api")
		}
	}
	media, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || media != "application/json" || len(response.Header.Values("Content-Type")) != 1 {
		return invalid("invalid_response")
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil || len(data) > maxResponseBytes {
		return invalid("invalid_response")
	}
	if response.StatusCode != 200 && !(method != http.MethodGet && response.StatusCode == 202) {
		var remote PublicationError
		if decodePublication(data, "Error", &remote) != nil || !publicationErrorStatus(remote.Code, response.StatusCode) {
			return invalid("invalid_response")
		}
		remote.Status = response.StatusCode
		return response.StatusCode, &remote
	}
	if decodePublication(data, kind, out) != nil {
		return invalid("invalid_response")
	}
	if method != http.MethodGet && response.StatusCode == 202 {
		op, ok := out.(*PublicationOperation)
		if !ok || len(response.Header.Values("Location")) != 1 || response.Header.Get("Location") != c.base+"/operations/"+op.ID {
			return invalid("invalid_response")
		}
	}
	return response.StatusCode, nil
}

func (c *PublicationClient) get(ctx context.Context, path, kind string, out any) error {
	budget := time.Minute
	if deadline, ok := ctx.Deadline(); ok {
		budget = min(time.Until(deadline), time.Hour)
	}
	ctx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()
	for attempt := 0; ; attempt++ {
		_, err := c.once(ctx, http.MethodGet, path, nil, kind, out)
		if err == nil || !publicationTransient(err) {
			return err
		}
		if err := c.transport.wait(ctx, min(time.Second, c.transport.poll*time.Duration(1<<min(attempt, 4)))); err != nil {
			return err
		}
	}
}

func (c *PublicationClient) Capabilities(ctx context.Context) (PublicationCapabilities, error) {
	var caps PublicationCapabilities
	if err := c.get(ctx, "/capabilities", "Capabilities", &caps); err != nil {
		return caps, err
	}
	if caps.Profile != PublicationProfile || caps.Scope != c.base || caps.DeploymentClass != "operator_local" {
		return PublicationCapabilities{}, errors.New("publication capabilities do not match the selected target")
	}
	for _, name := range []string{"uploads", "sealing", "publications", "previews", "domains", "promotion", "peer_selection", "peer_observation"} {
		if caps.Capabilities[name] {
			return PublicationCapabilities{}, errors.New("publication capability exceeds the pinned declaration profile")
		}
	}
	if c.Scope().EnvironmentID == "" && caps.Capabilities["targets"] || c.Scope().EnvironmentID != "" && (caps.Capabilities["applications"] || caps.Capabilities["target_bindings"]) {
		return PublicationCapabilities{}, errors.New("publication capabilities do not match scope")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.principal != "" && c.principal != caps.PrincipalID {
		return PublicationCapabilities{}, errors.New("publication principal changed")
	}
	c.principal = caps.PrincipalID
	return caps, nil
}

func (c *PublicationClient) supports(ctx context.Context, capability string) (PublicationCapabilities, error) {
	caps, err := c.Capabilities(ctx)
	if err == nil && !caps.Capabilities[capability] {
		err = &PublicationError{Code: "unsupported_capability"}
	}
	return caps, err
}

func (c *PublicationClient) Application(ctx context.Context, id string) (PublicationApplication, error) {
	var value PublicationApplication
	if !publicationDecimal(id, false) {
		return value, errors.New("invalid publication application ID")
	}
	if _, err := c.supports(ctx, "applications"); err != nil {
		return value, err
	}
	err := c.get(ctx, "/applications/"+id, "Application", &value)
	if err == nil && value.ID != id {
		err = errors.New("publication application identity mismatch")
	}
	return value, err
}

func (c *PublicationClient) Binding(ctx context.Context, id string) (PublicationBinding, error) {
	var value PublicationBinding
	if !publicationDecimal(id, false) {
		return value, errors.New("invalid publication binding ID")
	}
	if _, err := c.supports(ctx, "target_bindings"); err != nil {
		return value, err
	}
	err := c.get(ctx, "/target-bindings/"+id, "Binding", &value)
	if err == nil && value.ID != id {
		err = errors.New("publication binding identity mismatch")
	}
	return value, err
}

func (c *PublicationClient) Target(ctx context.Context, id string) (PublicationTarget, error) {
	var value PublicationTarget
	if !publicationDecimal(id, false) {
		return value, errors.New("invalid publication target ID")
	}
	if _, err := c.supports(ctx, "targets"); err != nil {
		return value, err
	}
	err := c.get(ctx, "/targets/"+id, "Target", &value)
	if err == nil && value.ID != id {
		err = errors.New("publication target identity mismatch")
	}
	return value, err
}

func (c *PublicationClient) validOperation(op PublicationOperation) bool {
	return op.Location == c.base+"/operations/"+op.ID
}

func (c *PublicationClient) operation(ctx context.Context, path string) (PublicationOperation, error) {
	var op PublicationOperation
	err := c.get(ctx, path, "Operation", &op)
	if err == nil && !c.validOperation(op) {
		err = errors.New("publication operation scope mismatch")
	}
	return op, err
}

func (c *PublicationClient) Operation(ctx context.Context, id string) (PublicationOperation, error) {
	if !publicationDecimal(id, false) {
		return PublicationOperation{}, errors.New("invalid publication operation ID")
	}
	if _, err := c.supports(ctx, "operations"); err != nil {
		return PublicationOperation{}, err
	}
	op, err := c.operation(ctx, "/operations/"+id)
	if err == nil && op.ID != id {
		err = errors.New("publication operation identity mismatch")
	}
	return op, err
}

var publicationKey = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,31}$`)
var publicationCursor = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,256}$`)

func (c *PublicationClient) lookup(ctx context.Context, key string) (PublicationOperation, error) {
	op, err := c.operation(ctx, "/operation-keys/"+key)
	if err == nil && op.Key != key {
		err = errors.New("publication operation key mismatch")
	}
	return op, err
}

func (c *PublicationClient) Lookup(ctx context.Context, key string) (PublicationOperation, error) {
	if !publicationKey.MatchString(key) {
		return PublicationOperation{}, errors.New("invalid publication operation key")
	}
	if _, err := c.supports(ctx, "operations"); err != nil {
		return PublicationOperation{}, err
	}
	return c.lookup(ctx, key)
}

type publicationPage struct {
	Snapshot string            `json:"snapshot"`
	Cursor   *string           `json:"next_cursor"`
	Complete bool              `json:"complete"`
	Items    []json.RawMessage `json:"items"`
}

func (c *PublicationClient) list(ctx context.Context, path, capability, kind string) ([]json.RawMessage, error) {
	caps, err := c.supports(ctx, capability)
	if err != nil {
		return nil, err
	}
	limitName := capability
	if capability == "target_bindings" {
		limitName = "bindings"
	}
	if capability == "history" {
		limitName = "operations"
	}
	limit := min(int(caps.Limits[limitName]), c.transport.maxItems)
	values := []json.RawMessage{}
	ids, cursors := map[string]bool{}, map[string]bool{}
	snapshot, next := "", path
	for pageNumber := 0; pageNumber < min(c.transport.maxPages, 64); pageNumber++ {
		var page publicationPage
		if err := c.get(ctx, next, "Collection", &page); err != nil {
			return nil, err
		}
		if snapshot != "" && page.Snapshot != snapshot || page.Complete != (page.Cursor == nil) {
			return nil, errors.New("incomplete publication snapshot")
		}
		snapshot = page.Snapshot
		for _, raw := range page.Items {
			var identity struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			}
			if json.Unmarshal(raw, &identity) != nil {
				return nil, errors.New("invalid publication item")
			}
			itemKind := kind
			if kind == "Operation" && identity.Status == "expired" {
				itemKind = "Tombstone"
			}
			if decodePublication(raw, itemKind, nil) != nil || ids[identity.ID] {
				return nil, errors.New("invalid or duplicate publication collection item")
			}
			if itemKind == "Operation" {
				var op PublicationOperation
				_ = json.Unmarshal(raw, &op)
				if !c.validOperation(op) {
					return nil, errors.New("publication history scope mismatch")
				}
			}
			ids[identity.ID] = true
			values = append(values, raw)
			if len(values) > limit {
				return nil, errors.New("publication collection exceeds its bound")
			}
		}
		if page.Complete {
			return values, nil
		}
		if !publicationCursor.MatchString(*page.Cursor) || cursors[*page.Cursor] {
			return nil, errors.New("invalid publication continuation")
		}
		cursors[*page.Cursor] = true
		separator := "?"
		if strings.Contains(path, "?") {
			separator = "&"
		}
		next = path + separator + "cursor=" + *page.Cursor
	}
	return nil, errors.New("publication collection did not complete within page limit")
}

func publicationValues[T any](raw []json.RawMessage, err error) ([]T, error) {
	if err != nil {
		return nil, err
	}
	result := make([]T, len(raw))
	for i := range raw {
		if json.Unmarshal(raw[i], &result[i]) != nil {
			return nil, errors.New("invalid publication item")
		}
	}
	return result, nil
}

func (c *PublicationClient) Applications(ctx context.Context) ([]PublicationApplication, error) {
	raw, err := c.list(ctx, "/applications", "applications", "Application")
	return publicationValues[PublicationApplication](raw, err)
}
func (c *PublicationClient) Bindings(ctx context.Context, applicationID string) ([]PublicationBinding, error) {
	if !publicationDecimal(applicationID, false) {
		return nil, errors.New("invalid publication application ID")
	}
	raw, err := c.list(ctx, "/applications/"+applicationID+"/target-bindings", "target_bindings", "Binding")
	values, err := publicationValues[PublicationBinding](raw, err)
	for _, v := range values {
		if v.ApplicationID != applicationID {
			return nil, errors.New("publication binding parent mismatch")
		}
	}
	return values, err
}
func (c *PublicationClient) Targets(ctx context.Context) ([]PublicationTarget, error) {
	raw, err := c.list(ctx, "/targets", "targets", "Target")
	return publicationValues[PublicationTarget](raw, err)
}
func (c *PublicationClient) History(ctx context.Context, state string) ([]PublicationOperation, error) {
	if state != "all" && state != "pending" && state != "terminal" {
		return nil, errors.New("invalid publication history state")
	}
	raw, err := c.list(ctx, "/operations?state="+state, "history", "Operation")
	values, err := publicationValues[PublicationOperation](raw, err)
	for _, op := range values {
		pending := op.Status == "accepted" || op.Status == "unknown"
		if state == "pending" && !pending || state == "terminal" && pending {
			return nil, errors.New("publication history does not match its requested state")
		}
	}
	return values, err
}
