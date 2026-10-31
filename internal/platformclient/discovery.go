package platformclient

import (
	"context"
	"errors"
	"net/url"
	"regexp"
)

var fingerprintPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// WithScope narrows the configured target; it cannot escape a provider assertion.
func (c *Client) WithScope(scope Scope) (*Client, error) {
	if !scope.Valid() || scope.OrganizationID != c.scope.OrganizationID || (c.scope.ProjectID != "" && scope.ProjectID != c.scope.ProjectID) || (c.scope.EnvironmentID != "" && scope.EnvironmentID != c.scope.EnvironmentID) {
		return nil, errors.New("resource parent does not match the configured target")
	}
	child := *c
	child.scope = scope
	return &child, nil
}

func (c *Client) Capabilities(ctx context.Context) (Capabilities, error) {
	var caps Capabilities
	err := c.request(ctx, "GET", c.base()+"/capabilities", nil, &caps)
	if err == nil && (caps.Scope != c.scope || caps.Version != APIVersion || !ValidID(caps.PrincipalID) || caps.Resources == nil || caps.ReceiptPolicy != "retain-or-tombstone") {
		err = errors.New("capability response does not match target or recovery contract")
	}
	return caps, err
}

func (c *Client) AssertOwner(ctx context.Context, owner string) error {
	caps, err := c.Capabilities(ctx)
	if err != nil {
		return err
	}
	if caps.PrincipalID != owner {
		return &APIError{Code: "ownership_conflict", Status: 409}
	}
	return nil
}

func (c *Client) validObject(o Object, kind string) bool {
	return o.Scope == c.scope && o.ResourceKind == kind && ValidID(o.ID) && ValidID(o.ManagementKey) && ValidID(o.OwnerID) && o.Revision > 0 && o.Name != ""
}

func (c *Client) GetObject(ctx context.Context, kind, id string) (Object, error) {
	var o Object
	if !c.scope.AllowsKind(kind) || !ValidID(id) {
		return o, errors.New("invalid resource kind, ID or parent scope")
	}
	err := c.request(ctx, "GET", c.base()+"/"+Collection(kind)+"/"+id, nil, &o)
	if err == nil && (!c.validObject(o, kind) || o.ID != id) {
		err = errors.New("invalid resource identity or revision in response")
	}
	return o, err
}

// List returns a complete bounded snapshot or an error, never partial success.
func (c *Client) List(ctx context.Context, kind string) ([]Object, error) {
	if !c.scope.AllowsKind(kind) {
		return nil, errors.New("invalid collection scope")
	}
	var result []Object
	tokens, ids := map[string]bool{}, map[string]bool{}
	token := ""
	var snapshot int64
	for page := 0; page < c.maxPages; page++ {
		var p Page
		path := c.base() + "/" + Collection(kind)
		if token != "" {
			path += "?page_token=" + url.QueryEscape(token)
		}
		if err := c.request(ctx, "GET", path, nil, &p); err != nil {
			return nil, err
		}
		if p.Scope != c.scope || p.ResourceKind != kind || p.Items == nil || p.Snapshot < 1 || (page > 0 && p.Snapshot != snapshot) || p.Complete != (p.NextToken == "") {
			return nil, errors.New("incomplete or invalid collection response")
		}
		snapshot = p.Snapshot
		for _, o := range p.Items {
			if !c.validObject(o, kind) || ids[o.ID] {
				return nil, errors.New("invalid or duplicate collection identity")
			}
			ids[o.ID] = true
			result = append(result, o)
			if len(result) > c.maxItems {
				return nil, errors.New("collection item limit exceeded")
			}
		}
		if p.Complete {
			return result, nil
		}
		if len(p.NextToken) > 2048 || tokens[p.NextToken] {
			return nil, errors.New("invalid or repeated continuation token")
		}
		tokens[p.NextToken] = true
		token = p.NextToken
	}
	return nil, errors.New("collection page limit exceeded")
}
