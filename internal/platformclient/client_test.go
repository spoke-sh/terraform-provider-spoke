package platformclient_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
	"github.com/spoke-sh/terraform-provider-spoke/internal/testutil/mockplatform"
)

func client(t *testing.T, s *mockplatform.Server) *api.Client {
	t.Helper()
	ca, err := os.ReadFile(s.CAFile)
	if err != nil {
		t.Fatal(err)
	}
	c, err := api.New(api.Config{Endpoint: s.HTTP.URL, Scope: s.Scope, Token: s.Token, CAPEM: ca, PollInterval: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func create(key, name string, protected bool) api.Intent {
	return api.Intent{Kind: "create", ManagementKey: key, Spec: &api.BucketSpec{Name: name, DeletionProtection: protected}}
}
func mustMutate(t *testing.T, c *api.Client, i api.Intent) api.Operation {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	op, err := c.Mutate(ctx, i)
	if err != nil {
		t.Fatal(err)
	}
	return op
}

func TestLifecycleRecoveryAndGenerations(t *testing.T) {
	s := mockplatform.New(t)
	c := client(t, s)
	ctx := context.Background()
	if _, err := c.GetProject(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetEnvironment(ctx); err != nil {
		t.Fatal(err)
	}
	s.LoseNextReply()
	intent := create("deployment-a", "assets", false)
	op := mustMutate(t, c, intent)
	// A new client has no process-local journal, but still recovers the receipt.
	replayed := mustMutate(t, client(t, s), intent)
	if replayed.ID != op.ID || s.BucketCount() != 1 || s.OperationCount() != 1 {
		t.Fatal("duplicate create after lost reply")
	}
	b, err := c.GetBucket(ctx, op.TargetID)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"renamed", "assets"} {
		mustMutate(t, c, api.Intent{Kind: "update", ManagementKey: b.ManagementKey, TargetID: b.ID, ExpectedRevision: b.Revision, Spec: &api.BucketSpec{Name: name}})
		b, err = c.GetBucket(ctx, b.ID)
		if err != nil {
			t.Fatal(err)
		}
	}
	if b.Revision != 3 || s.OperationCount() != 3 {
		t.Fatal("return to prior configuration was treated as replay")
	}
	del := api.Intent{Kind: "delete", ManagementKey: b.ManagementKey, TargetID: b.ID, ExpectedRevision: b.Revision}
	s.LoseNextReply()
	mustMutate(t, c, del)
	mustMutate(t, client(t, s), del)
	if _, err := c.GetBucket(ctx, b.ID); !api.IsCode(err, "resource_not_found") {
		t.Fatalf("delete: %v", err)
	}
	recreated := mustMutate(t, c, intent)
	if recreated.TargetID == b.ID || recreated.Generation != 5 {
		t.Fatal("recreation did not receive a new identity/generation")
	}
	s.Remove(recreated.TargetID)
	if next := mustMutate(t, c, intent); next.TargetID == recreated.TargetID {
		t.Fatal("external deletion recovered a stale creation")
	}
}

func TestPendingOperationIsNotAdvancedByReads(t *testing.T) {
	s := mockplatform.New(t)
	s.HoldOperations(true)
	c := client(t, s)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	op, err := c.Mutate(ctx, create("pending", "assets", false))
	var pending *api.PendingError
	if !errors.As(err, &pending) || op.Status != "accepted" || s.BucketCount() != 0 {
		t.Fatalf("pending: %v %v", op, err)
	}
	if _, err := c.GetBucket(context.Background(), op.TargetID); !api.IsCode(err, "operation_pending") {
		t.Fatalf("read pending: %v", err)
	}
	if _, err := c.Mutate(context.Background(), create("pending", "other", false)); !errors.As(err, &pending) {
		t.Fatalf("different intent bypassed pending operation: %v", err)
	}
	s.CompleteAll()
	if recovered := mustMutate(t, client(t, s), create("pending", "assets", false)); recovered.ID != op.ID || s.OperationCount() != 1 {
		t.Fatal("lost pending identity")
	}
}

func TestProtectionRevisionsAndAuthorization(t *testing.T) {
	s := mockplatform.New(t)
	c := client(t, s)
	ctx := context.Background()
	op := mustMutate(t, c, create("protected", "assets", true))
	b, err := c.GetBucket(ctx, op.TargetID)
	if err != nil {
		t.Fatal(err)
	}
	del := api.Intent{Kind: "delete", ManagementKey: b.ManagementKey, TargetID: b.ID, ExpectedRevision: b.Revision}
	if _, err := c.Mutate(ctx, del); !api.IsCode(err, "deletion_protected") {
		t.Fatalf("protected delete: %v", err)
	}
	stale := api.Intent{Kind: "update", ManagementKey: b.ManagementKey, TargetID: b.ID, ExpectedRevision: b.Revision, Spec: &api.BucketSpec{Name: "updated"}}
	s.Drift(b.ID, api.BucketSpec{Name: "external"})
	if _, err := c.Mutate(ctx, stale); !api.IsCode(err, "revision_conflict") {
		t.Fatalf("stale update: %v", err)
	}
	b, err = c.GetBucket(ctx, b.ID)
	if err != nil {
		t.Fatal(err)
	}
	s.SetNonempty(b.ID, true)
	del.ExpectedRevision = b.Revision
	if _, err := c.Mutate(ctx, del); !api.IsCode(err, "not_empty") {
		t.Fatalf("nonempty delete: %v", err)
	}
	s.DenyAccess(true)
	if _, err := c.GetBucket(ctx, b.ID); !api.IsCode(err, "forbidden") {
		t.Fatalf("denied read: %v", err)
	}
	if s.BucketCount() != 1 {
		t.Fatal("denied read deleted resource")
	}
	s.DenyAccess(false)
	s.FailBucketReads("unavailable")
	if _, err := c.GetBucket(ctx, b.ID); api.IsCode(err, "resource_not_found") || err == nil {
		t.Fatalf("outage became absence: %v", err)
	}
}

func TestConcurrentIdenticalCreate(t *testing.T) {
	s := mockplatform.New(t)
	c := client(t, s)
	var wg sync.WaitGroup
	results := make(chan api.Operation, 2)
	errs := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			op, err := c.Mutate(context.Background(), create("same", "assets", false))
			results <- op
			errs <- err
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var id string
	for op := range results {
		if id != "" && id != op.ID {
			t.Fatal("duplicate operation")
		}
		id = op.ID
	}
	if s.BucketCount() != 1 || s.OperationCount() != 1 {
		t.Fatal("duplicate resource")
	}
}

func TestTLSAndScopeAreVerified(t *testing.T) {
	s := mockplatform.New(t)
	c, err := api.New(api.Config{Endpoint: s.HTTP.URL, Scope: s.Scope, Token: s.Token, PollInterval: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetProject(context.Background()); err == nil {
		t.Fatal("trusted an unknown CA")
	}
	scope := s.Scope
	scope.OrganizationID = "other-org"
	c, err = api.New(api.Config{Endpoint: s.HTTP.URL, Scope: scope, Token: s.Token, HTTPClient: s.HTTP.Client()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetProject(context.Background()); !api.IsCode(err, "forbidden") {
		t.Fatalf("cross tenant: %v", err)
	}
}

func TestResponseValidationAndRedaction(t *testing.T) {
	for _, tc := range []struct {
		name          string
		status        int
		body, version string
	}{
		{"masked 404", 404, `{"message":"secret-token"}`, api.APIVersion},
		{"wrong not found status", 503, `{"code":"resource_not_found"}`, api.APIVersion},
		{"malformed", 200, `{"id":`, api.APIVersion},
		{"null", 200, `null`, api.APIVersion},
		{"incompatible", 200, `{}`, "other"},
		{"oversize", 200, strings.Repeat("x", (1<<20)+1), api.APIVersion},
		{"redaction", 403, `{"code":"forbidden","message":"secret-token"}`, api.APIVersion},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Spoke-API-Version", tc.version)
				w.WriteHeader(tc.status)
				_, _ = ioWrite(w, tc.body)
			}))
			defer s.Close()
			c, err := api.New(api.Config{Endpoint: s.URL, Scope: api.Scope{OrganizationID: "org", ProjectID: "project", EnvironmentID: "env"}, Token: "secret-token", HTTPClient: s.Client(), PollInterval: time.Millisecond})
			if err != nil {
				t.Fatal(err)
			}
			_, err = c.GetBucket(context.Background(), "bucket")
			if err == nil || api.IsCode(err, "resource_not_found") || strings.Contains(err.Error(), "secret-token") {
				t.Fatalf("unsafe response: %v", err)
			}
		})
	}
}

func ioWrite(w http.ResponseWriter, s string) (int, error) { return w.Write([]byte(s)) }

func TestFingerprintGolden(t *testing.T) {
	intent := create("key", "assets", false)
	b, _ := json.Marshal(intent)
	const expected = `{"kind":"create","management_key":"key","target_id":"","expected_revision":0,"spec":{"name":"assets","deletion_protection":false}}`
	if string(b) != expected {
		t.Fatalf("contract serialization changed: %s", b)
	}
}
