package platformclient_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
	"github.com/spoke-sh/terraform-provider-spoke/internal/testutil/mockplatform"
)

func TestPendingRevocationAndRetainedIndependentData(t *testing.T) {
	s := mockplatform.New(t)
	c := client(t, s)
	retained := mustMutate(t, c, create("retained", "unrelated-data", true))
	s.SetNonempty(retained.TargetID, true)
	s.AddCredential("worker", mockplatform.Principal{ID: "worker-owner", OrganizationID: s.Scope.OrganizationID, Admin: true})
	worker := scopedClient(t, s, s.Scope, "worker")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.HoldOperations(true)
	result := make(chan error, 1)
	go func() { _, err := worker.Mutate(ctx, create("revoked", "pending", false)); result <- err }()
	// Drain the earlier retained create, then wait for the worker's actual admission.
	<-s.Accepted()
	accepted := <-s.Accepted()
	s.RevokePrincipal("worker-owner")
	s.CompleteAll()
	cancel()
	if err := <-result; err == nil {
		t.Fatal("revoked work reported success")
	}
	s.Restart(t)
	if s.BucketCount() != 1 {
		t.Fatal("revocation lost unrelated data or completed unauthorized create")
	}
	if _, err := worker.GetBucket(context.Background(), accepted.TargetID); !api.IsCode(err, "forbidden") {
		t.Fatalf("revoked disclosure: %v", err)
	}
	b, err := client(t, s).GetBucket(context.Background(), retained.TargetID)
	if err != nil || !b.DeletionProtection {
		t.Fatalf("retained data: %v %+v", err, b)
	}
}

func TestConcurrentConflictingIntentAndPendingDeadline(t *testing.T) {
	s := mockplatform.New(t)
	s.HoldOperations(true)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, name := range []string{"first", "second"} {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			_, err := client(t, s).Mutate(ctx, create("same-key", name, false))
			errs <- err
		}(name)
	}
	accepted := <-s.Accepted()
	cancel()
	wg.Wait()
	close(errs)
	for err := range errs {
		if err == nil {
			t.Fatal("pending conflicting work succeeded")
		}
	}
	if s.OperationCount() != 1 {
		t.Fatal("concurrent conflict accepted multiple operations")
	}
	s.Restart(t)
	s.CompleteAll()
	b, err := client(t, s).GetBucket(context.Background(), accepted.TargetID)
	if err != nil {
		t.Fatal(err)
	}
	if b.Name != "first" && b.Name != "second" {
		t.Fatal("lost winning intent")
	}
	if s.BucketCount() != 1 {
		t.Fatal("duplicate logical resource")
	}
}

func TestPendingSettlementRetainsRecoveryContext(t *testing.T) {
	s := mockplatform.New(t)
	s.HoldOperations(true)
	op, err := timeoutClient(t, s).Mutate(context.Background(), create("timeout", "pending", false))
	var pending *api.PendingError
	if !errors.As(err, &pending) || !strings.Contains(err.Error(), op.ID) || !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("lost recovery context: %v", err)
	}
	if err := s.ExpireReceipt(s.Scope, "bucket", "timeout"); err == nil {
		t.Fatal("expired pending receipt")
	}
	s.Restart(t)
	s.CompleteAll()
	if next := mustMutate(t, client(t, s), create("timeout", "pending", false)); next.ID != op.ID {
		t.Fatal("restarted pending work duplicated")
	}
}
