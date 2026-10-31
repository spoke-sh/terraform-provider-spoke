package platformclient_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
	"github.com/spoke-sh/terraform-provider-spoke/internal/testutil/mockplatform"
)

func scopedClient(t *testing.T, s *mockplatform.Server, scope api.Scope, token string) *api.Client {
	t.Helper()
	c, err := api.New(api.Config{Endpoint: s.HTTP.URL, Scope: scope, Token: token, HTTPClient: s.HTTP.Client(), PollInterval: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func TestCapabilitiesAndCompleteDiscovery(t *testing.T) {
	s := mockplatform.New(t)
	for _, tc := range []struct {
		kind   string
		scope  api.Scope
		seeded int
	}{
		{"project", api.Scope{OrganizationID: s.Scope.OrganizationID}, 1},
		{"environment", api.Scope{OrganizationID: s.Scope.OrganizationID, ProjectID: s.Scope.ProjectID}, 1},
		{"bucket", s.Scope, 0},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			c := scopedClient(t, s, tc.scope, s.Token)
			caps, err := c.Capabilities(context.Background())
			if err != nil || caps.Scope != tc.scope || caps.PrincipalID != "principal-admin" {
				t.Fatalf("capabilities: %+v %v", caps, err)
			}
			for _, key := range []string{"one", "two", "three"} {
				mustMutate(t, c, api.Intent{Kind: "create", ResourceKind: tc.kind, ManagementKey: key, Spec: &api.BucketSpec{Name: key}})
			}
			before := s.OperationCount()
			objects, err := c.List(context.Background(), tc.kind)
			if err != nil || len(objects) != tc.seeded+3 {
				t.Fatalf("discovery: %d %v", len(objects), err)
			}
			if s.OperationCount() != before {
				t.Fatal("discovery mutated resources")
			}
		})
	}
}
func TestCapabilityValidationBeforeMutation(t *testing.T) {
	scope := api.Scope{OrganizationID: "org", ProjectID: "project", EnvironmentID: "env"}
	for _, variant := range []string{"wrong target", "wrong version", "wrong principal", "missing kind", "wrong retention"} {
		t.Run(variant, func(t *testing.T) {
			requests := 0
			srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				if r.Method != "GET" || !strings.HasSuffix(r.URL.Path, "/capabilities") {
					t.Error("dispatched before validating capabilities")
				}
				caps := api.Capabilities{Scope: scope, Version: api.APIVersion, PrincipalID: "principal", Resources: []string{"bucket"}, ReceiptPolicy: "retain-or-tombstone"}
				switch variant {
				case "wrong target":
					caps.OrganizationID = "another"
				case "wrong version":
					caps.Version = "mock-v1"
				case "wrong principal":
					caps.PrincipalID = ""
				case "missing kind":
					caps.Resources = []string{}
				case "wrong retention":
					caps.ReceiptPolicy = "evict"
				}
				w.Header().Set("X-Spoke-API-Version", api.APIVersion)
				_ = json.NewEncoder(w).Encode(caps)
			}))
			defer srv.Close()
			c, err := api.New(api.Config{Endpoint: srv.URL, Scope: scope, Token: "synthetic", HTTPClient: srv.Client()})
			if err != nil {
				t.Fatal(err)
			}
			if _, err = c.Mutate(context.Background(), create("key", "assets", false)); err == nil || requests != 1 {
				t.Fatalf("unsafe capability negotiation: %v requests=%d", err, requests)
			}
		})
	}
}
func TestDiscoveryRejectsIncompleteAndUnboundedResults(t *testing.T) {
	scope := api.Scope{OrganizationID: "org", ProjectID: "project", EnvironmentID: "env"}
	object := func(id string) api.Object {
		return api.Object{Bucket: api.Bucket{Scope: scope, BucketSpec: api.BucketSpec{Name: id}, ID: id, ManagementKey: id, OwnerID: "owner", Revision: 1}, ResourceKind: "bucket"}
	}
	for _, variant := range []string{"repeated token", "partial terminal", "changed snapshot", "duplicate ID", "wrong scope", "null items", "denied page", "page limit", "item limit", "huge token"} {
		t.Run(variant, func(t *testing.T) {
			requests := 0
			srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				w.Header().Set("X-Spoke-API-Version", api.APIVersion)
				p := api.Page{Scope: scope, ResourceKind: "bucket", Items: []api.Object{object("one")}, Snapshot: 1, NextToken: "next", Complete: false}
				if requests > 1 {
					p.Items = []api.Object{object("two")}
					p.NextToken = ""
					p.Complete = true
				}
				switch variant {
				case "repeated token":
					p.NextToken = "next"
					p.Complete = false
				case "partial terminal":
					p.NextToken = ""
					p.Complete = false
				case "changed snapshot":
					if requests > 1 {
						p.Snapshot = 2
					}
				case "duplicate ID":
					p.Items = []api.Object{object("one")}
				case "wrong scope":
					p.Items[0].OrganizationID = "other"
				case "null items":
					p.Items = nil
				case "denied page":
					if requests > 1 {
						w.WriteHeader(403)
						_, _ = w.Write([]byte(`{"code":"forbidden","message":"private-canary"}`))
						return
					}
				case "huge token":
					p.NextToken = strings.Repeat("x", 2049)
				}
				_ = json.NewEncoder(w).Encode(p)
			}))
			defer srv.Close()
			cfg := api.Config{Endpoint: srv.URL, Scope: scope, Token: "synthetic", HTTPClient: srv.Client()}
			if variant == "page limit" {
				cfg.MaxPages = 1
			}
			if variant == "item limit" {
				cfg.MaxItems = 1
			}
			c, err := api.New(cfg)
			if err != nil {
				t.Fatal(err)
			}
			values, err := c.List(context.Background(), "bucket")
			if err == nil || values != nil || strings.Contains(err.Error(), "private-canary") || requests > 2 {
				t.Fatalf("partial success or unbounded work: values=%v err=%v requests=%d", values, err, requests)
			}
		})
	}
}
func TestReadRetryBoundsAndCancellation(t *testing.T) {
	s := mockplatform.New(t)
	var delays []time.Duration
	c, err := api.New(api.Config{Endpoint: s.HTTP.URL, Scope: s.Scope, Token: s.Token, HTTPClient: s.HTTP.Client(), PollInterval: time.Millisecond, Wait: func(ctx context.Context, d time.Duration) error { delays = append(delays, d); return ctx.Err() }})
	if err != nil {
		t.Fatal(err)
	}
	s.FailNext("GET", s.Base()+"/buckets", 429, "rate_limited", 3)
	if _, err = c.List(context.Background(), "bucket"); !api.IsCode(err, "rate_limited") || len(delays) != 2 || delays[0] != time.Second {
		t.Fatalf("retry bounds/hints: %v %v", err, delays)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = c.List(ctx, "bucket"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
	s.SetQuota("bucket", 0)
	before := s.OperationCount()
	if _, err = c.Mutate(context.Background(), create("quota", "assets", false)); !api.IsCode(err, "quota_exceeded") || s.OperationCount() != before {
		t.Fatalf("quota: %v", err)
	}
}
func TestReceiptOwnershipExpirationAndTenants(t *testing.T) {
	s := mockplatform.New(t)
	c := client(t, s)
	intent := create("ownership", "same-name", false)
	op := mustMutate(t, c, intent)
	s.AddCredential("other-owner", mockplatform.Principal{ID: "other-owner", OrganizationID: s.Scope.OrganizationID, Admin: true})
	other := scopedClient(t, s, s.Scope, "other-owner")
	if _, err := other.Mutate(context.Background(), intent); !api.IsCode(err, "ownership_conflict") {
		t.Fatalf("second owner took over: %v", err)
	}
	if err := other.AssertOwner(context.Background(), "principal-admin"); !api.IsCode(err, "ownership_conflict") {
		t.Fatalf("import took over: %v", err)
	}
	different := mustMutate(t, other, create("other-key", "same-name", false))
	if different.TargetID == op.TargetID {
		t.Fatal("adopted by name")
	}
	if err := s.ExpireReceipt(s.Scope, "bucket", "ownership"); err != nil {
		t.Fatal(err)
	}
	s.Restart(t)
	if _, err := client(t, s).Mutate(context.Background(), intent); !api.IsCode(err, "receipt_expired") {
		t.Fatalf("expired receipt replayed: %v", err)
	}
	if s.BucketCount() != 2 || s.OperationCount() != 2 {
		t.Fatal("expiration lost ownership or duplicated resources")
	}
	tenant := api.Scope{OrganizationID: "org-two", ProjectID: "project-two", EnvironmentID: "env-two"}
	s.SeedTenant(tenant, "tenant-token", "tenant-owner")
	outsider := scopedClient(t, s, s.Scope, "tenant-token")
	if _, err := outsider.List(context.Background(), "bucket"); !api.IsCode(err, "forbidden") {
		t.Fatalf("tenant disclosure: %v", err)
	}
	tclient := scopedClient(t, s, tenant, "tenant-token")
	mustMutate(t, tclient, create("ownership", "same-name", false))
	if s.BucketCount() != 3 {
		t.Fatal("tenant scopes shared ownership namespace")
	}
}
