package platformclient_test

import (
	"context"
	"errors"
	"testing"
	"time"

	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
	"github.com/spoke-sh/terraform-provider-spoke/internal/testutil/mockplatform"
)

func objectIntent(kind, key, name string) api.Intent {
	return api.Intent{Kind: "create", ResourceKind: kind, ManagementKey: key, Spec: &api.BucketSpec{Name: name}}
}
func rootClient(t *testing.T, s *mockplatform.Server) *api.Client {
	return scopedClient(t, s, api.Scope{OrganizationID: s.Scope.OrganizationID}, s.Token)
}
func grantIntent(key, subject, target, permission string) api.Intent {
	i := objectIntent("grant", key, key)
	i.Spec.SubjectID = subject
	i.Spec.Target = target
	i.Spec.Permission = permission
	return i
}
func TestScopeKindsInterruptedCRUDAndRevisions(t *testing.T) {
	for _, kind := range []string{"project", "environment", "service_account", "grant"} {
		t.Run(kind, func(t *testing.T) {
			s := mockplatform.New(t)
			scope := api.Scope{OrganizationID: s.Scope.OrganizationID}
			if kind == "environment" {
				scope.ProjectID = s.Scope.ProjectID
			}
			c := scopedClient(t, s, scope, s.Token)
			intent := objectIntent(kind, "managed", kind)
			if kind == "grant" {
				subject := mustMutate(t, rootClient(t, s), objectIntent("service_account", "subject", "subject"))
				intent = grantIntent("managed", subject.TargetID, scope.Path(), "read")
			}
			pending := func(i api.Intent) api.Operation {
				t.Helper()
				s.HoldOperations(true)
				tc, err := api.New(api.Config{Endpoint: s.HTTP.URL, Scope: scope, Token: s.Token, HTTPClient: s.HTTP.Client(), Wait: func(context.Context, time.Duration) error { return context.DeadlineExceeded }})
				if err != nil {
					t.Fatal(err)
				}
				op, err := tc.Mutate(context.Background(), i)
				var pe *api.PendingError
				if !errors.As(err, &pe) {
					t.Fatalf("lost pending identity: %v", err)
				}
				s.Restart(t)
				s.CompleteAll()
				s.HoldOperations(false)
				recovered := mustMutate(t, c, i)
				if recovered.ID != op.ID {
					t.Fatal("replayed a new operation")
				}
				return op
			}
			created := pending(intent)
			o, err := c.GetObject(context.Background(), kind, created.TargetID)
			if err != nil {
				t.Fatal(err)
			}
			spec := o.BucketSpec
			spec.Name = "updated"
			update := api.Intent{Kind: "update", ResourceKind: kind, ManagementKey: o.ManagementKey, TargetID: o.ID, ExpectedRevision: o.Revision + 1, Spec: &spec}
			if _, err := c.Mutate(context.Background(), update); !api.IsCode(err, "revision_conflict") {
				t.Fatalf("stale revision: %v", err)
			}
			update.ExpectedRevision = o.Revision
			pending(update)
			o, err = c.GetObject(context.Background(), kind, o.ID)
			if err != nil || o.Name != "updated" || o.Revision != 2 {
				t.Fatalf("update: %+v %v", o, err)
			}
			pending(api.Intent{Kind: "delete", ResourceKind: kind, ManagementKey: o.ManagementKey, TargetID: o.ID, ExpectedRevision: o.Revision})
			if _, err = c.GetObject(context.Background(), kind, o.ID); !api.IsCode(err, "resource_not_found") {
				t.Fatalf("absence: %v", err)
			}
		})
	}
}
func TestGrantsEnforceAccessAndIndependentRevocation(t *testing.T) {
	s := mockplatform.New(t)
	root := rootClient(t, s)
	bucket := mustMutate(t, client(t, s), create("data", "retained", false))
	subject := mustMutate(t, root, objectIntent("service_account", "reader", "reader"))
	s.AddCredential("reader-token", mockplatform.Principal{ID: subject.TargetID, OrganizationID: s.Scope.OrganizationID})
	reader := scopedClient(t, s, s.Scope, "reader-token")
	if _, err := reader.GetBucket(context.Background(), bucket.TargetID); !api.IsCode(err, "forbidden") {
		t.Fatalf("login granted access: %v", err)
	}
	g1 := mustMutate(t, root, grantIntent("first", subject.TargetID, s.Scope.Path(), "read"))
	g2 := mustMutate(t, root, grantIntent("independent", subject.TargetID, s.Scope.Path(), "read"))
	if _, err := reader.GetBucket(context.Background(), bucket.TargetID); err != nil {
		t.Fatal(err)
	}
	if _, err := reader.Mutate(context.Background(), create("unauthorized", "write", false)); !api.IsCode(err, "forbidden") {
		t.Fatalf("read grant authorized write: %v", err)
	}
	if _, err := root.Mutate(context.Background(), api.Intent{Kind: "delete", ResourceKind: "service_account", ManagementKey: "reader", TargetID: subject.TargetID, ExpectedRevision: 1}); !api.IsCode(err, "dependency_conflict") {
		t.Fatalf("subject cascade: %v", err)
	}
	for n, g := range []api.Operation{g1, g2} {
		key := "first"
		if n == 1 {
			key = "independent"
		}
		mustMutate(t, root, api.Intent{Kind: "delete", ResourceKind: "grant", ManagementKey: key, TargetID: g.TargetID, ExpectedRevision: 1})
		_, err := reader.GetBucket(context.Background(), bucket.TargetID)
		if n == 0 && err != nil {
			t.Fatalf("removal erased independent grant: %v", err)
		}
		if n == 1 && !api.IsCode(err, "forbidden") {
			t.Fatalf("revocation not enforced: %v", err)
		}
	}
	if s.BucketCount() != 1 {
		t.Fatal("revocation deleted retained data")
	}
	mustMutate(t, root, api.Intent{Kind: "delete", ResourceKind: "service_account", ManagementKey: "reader", TargetID: subject.TargetID, ExpectedRevision: 1})
}
func TestParentDependenciesIncludingPendingChildren(t *testing.T) {
	s := mockplatform.New(t)
	root := rootClient(t, s)
	project := mustMutate(t, root, objectIntent("project", "parent", "parent"))
	scope := api.Scope{OrganizationID: s.Scope.OrganizationID, ProjectID: project.TargetID}
	pc := scopedClient(t, s, scope, s.Token)
	environment := mustMutate(t, pc, objectIntent("environment", "child", "child"))
	if _, err := root.Mutate(context.Background(), api.Intent{Kind: "delete", ResourceKind: "project", ManagementKey: "parent", TargetID: project.TargetID, ExpectedRevision: 1}); !api.IsCode(err, "dependency_conflict") {
		t.Fatalf("parent deleted children: %v", err)
	}
	scope.EnvironmentID = environment.TargetID
	s.HoldOperations(true)
	pendingClient, err := api.New(api.Config{Endpoint: s.HTTP.URL, Scope: scope, Token: s.Token, HTTPClient: s.HTTP.Client(), Wait: func(context.Context, time.Duration) error { return context.DeadlineExceeded }})
	if err != nil {
		t.Fatal(err)
	}
	_, err = pendingClient.Mutate(context.Background(), create("pending-child", "data", false))
	if err == nil {
		t.Fatal("pending child completed")
	}
	if _, err := pc.Mutate(context.Background(), api.Intent{Kind: "delete", ResourceKind: "environment", ManagementKey: "child", TargetID: environment.TargetID, ExpectedRevision: 1}); !api.IsCode(err, "dependency_conflict") {
		t.Fatalf("parent deleted pending child: %v", err)
	}
	s.CompleteAll()
	if s.BucketCount() != 1 {
		t.Fatal("lost accepted child")
	}
}

func TestMissingParentIsNotChildAbsence(t *testing.T) {
	s := mockplatform.New(t)
	c := client(t, s)
	op := mustMutate(t, c, create("retained-child", "data", false))
	s.Remove(s.Scope.ProjectID)
	if _, err := c.GetBucket(context.Background(), op.TargetID); !api.IsCode(err, "dependency_conflict") {
		t.Fatalf("missing parent became child absence: %v", err)
	}
	if s.BucketCount() != 1 {
		t.Fatal("parent loss erased retained child")
	}
}

func TestExactTargetGrantDoesNotRequireParentAuthority(t *testing.T) {
	s := mockplatform.New(t)
	root := rootClient(t, s)
	subject := mustMutate(t, root, objectIntent("service_account", "owner", "owner"))
	s.AddCredential("object-owner", mockplatform.Principal{ID: subject.TargetID, OrganizationID: s.Scope.OrganizationID})
	worker := scopedClient(t, s, s.Scope, "object-owner")
	broad := mustMutate(t, root, grantIntent("bootstrap", subject.TargetID, s.Scope.Path(), "manage"))
	owned := mustMutate(t, worker, create("owned", "assets", false))
	sibling := mustMutate(t, client(t, s), create("sibling", "unrelated", false))
	exact := mustMutate(t, root, grantIntent("exact", subject.TargetID, s.Scope.Path()+"/buckets/"+owned.TargetID, "manage"))
	mustMutate(t, root, api.Intent{Kind: "delete", ResourceKind: "grant", ManagementKey: "bootstrap", TargetID: broad.TargetID, ExpectedRevision: 1})
	// Scope negotiation and owned receipts remain readable without sibling visibility.
	if err := worker.AssertOwner(context.Background(), subject.TargetID); err != nil {
		t.Fatal(err)
	}
	update := api.Intent{Kind: "update", ManagementKey: "owned", TargetID: owned.TargetID, ExpectedRevision: 1, Spec: &api.BucketSpec{Name: "updated"}}
	s.LoseNextReply()
	mustMutate(t, worker, update)
	observed, err := worker.GetBucket(context.Background(), owned.TargetID)
	if err != nil || observed.Name != "updated" {
		t.Fatalf("exact target update: %+v %v", observed, err)
	}
	if _, err = worker.GetBucket(context.Background(), sibling.TargetID); !api.IsCode(err, "forbidden") {
		t.Fatalf("sibling disclosed: %v", err)
	}
	if _, err = worker.List(context.Background(), "bucket"); !api.IsCode(err, "forbidden") {
		t.Fatalf("partial authority listed collection: %v", err)
	}
	if _, err = worker.Mutate(context.Background(), create("new", "unauthorized", false)); !api.IsCode(err, "forbidden") {
		t.Fatalf("object authority allowed parent create: %v", err)
	}
	if _, err = worker.Mutate(context.Background(), api.Intent{Kind: "delete", ManagementKey: "owned", TargetID: owned.TargetID, ExpectedRevision: 2}); !api.IsCode(err, "dependency_conflict") {
		t.Fatalf("deleted its dependent grant: %v", err)
	}
	mustMutate(t, root, api.Intent{Kind: "delete", ResourceKind: "grant", ManagementKey: "exact", TargetID: exact.TargetID, ExpectedRevision: 1})
	if _, err = worker.GetBucket(context.Background(), owned.TargetID); !api.IsCode(err, "forbidden") {
		t.Fatalf("revocation ineffective: %v", err)
	}
	if _, err = root.Mutate(context.Background(), api.Intent{Kind: "delete", ResourceKind: "service_account", ManagementKey: "owner", TargetID: subject.TargetID, ExpectedRevision: 1}); !api.IsCode(err, "dependency_conflict") {
		t.Fatalf("retired owner with retained resources: %v", err)
	}
}
