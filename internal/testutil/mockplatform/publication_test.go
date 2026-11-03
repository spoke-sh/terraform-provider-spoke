package mockplatform

import (
	"context"
	"os"
	"testing"

	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

func publicationFixtureClient(t *testing.T, s *PublicationServer, role, environment string) *api.PublicationClient {
	t.Helper()
	token, err := os.ReadFile(s.Tokens[role])
	if err != nil {
		t.Fatal(err)
	}
	c, err := api.NewPublication(api.Config{Endpoint: s.HTTP.URL, HTTPClient: s.HTTP.Client(), Token: string(token), Scope: api.Scope{OrganizationID: "example", ProjectID: "saga", EnvironmentID: environment}})
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func TestPublicationSyntheticReservationAndRetainedRelease(t *testing.T) {
	s := NewPublication(t)
	project := publicationFixtureClient(t, s, "project", "")
	nightly := publicationFixtureClient(t, s, "nightly", "nightly")
	ctx := context.Background()
	policy := api.PublicationPolicy{Name: "site", RetentionDays: 30}
	app, err := project.CreateApplication(ctx, "app", policy)
	if err != nil {
		t.Fatal(err)
	}
	aid := *app.ResourceID
	binding, err := project.CreateBinding(ctx, aid, "bind", "1", "nightly")
	if err != nil {
		t.Fatal(err)
	}
	bid := *binding.ResourceID
	target, err := nightly.CreateTarget(ctx, bid, "target", policy)
	if err != nil {
		t.Fatal(err)
	}
	tid := *target.ResourceID
	if _, err = nightly.CreateTarget(ctx, bid, "duplicate", policy); err == nil {
		t.Fatal("reservation consumed twice")
	}
	if _, err = project.DeleteApplication(ctx, aid, "held", "1"); err == nil {
		t.Fatal("application lost held reservation")
	}
	b, err := project.Binding(ctx, bid)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = project.DeleteBinding(ctx, bid, "cancel-live", b.Revision); err == nil {
		t.Fatal("cancelled a live target")
	}
	s.DenyRelease(true)
	if _, err = nightly.DeleteTarget(ctx, tid, "retire", "1"); err == nil {
		t.Fatal("denied release reported success")
	}
	s.Restart()
	if _, err = nightly.Target(ctx, tid); err == nil {
		t.Fatal("target tombstone lost during restart")
	}
	if _, err = project.DeleteApplication(ctx, aid, "held-again", "1"); err == nil {
		t.Fatal("lost release removed parent hold")
	}
	s.DenyRelease(false)
	b, err = project.Binding(ctx, bid)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = project.DeleteBinding(ctx, bid, "release", b.Revision); err != nil {
		t.Fatal(err)
	}
	if _, err = project.DeleteApplication(ctx, aid, "delete", "1"); err != nil {
		t.Fatal(err)
	}
}
func TestPublicationSyntheticScopedCollectionsAndCancellation(t *testing.T) {
	s := NewPublication(t)
	project := publicationFixtureClient(t, s, "project", "")
	nightly := publicationFixtureClient(t, s, "nightly", "nightly")
	ctx := context.Background()
	app, err := project.CreateApplication(ctx, "app", api.PublicationPolicy{Name: "site", RetentionDays: 30})
	if err != nil {
		t.Fatal(err)
	}
	b, err := project.CreateBinding(ctx, *app.ResourceID, "reservation", "1", "nightly")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = project.DeleteBinding(ctx, *b.ResourceID, "cancel", "1"); err != nil {
		t.Fatal(err)
	}
	if _, err = nightly.CreateTarget(ctx, *b.ResourceID, "late", api.PublicationPolicy{Name: "late", RetentionDays: 30}); err == nil {
		t.Fatal("cancelled binding consumed late")
	}
	if _, err = publicationFixtureClient(t, s, "nightly", "production").Capabilities(ctx); err == nil {
		t.Fatal("cross-environment access")
	}
	if _, err = publicationFixtureClient(t, s, "reader", "").Lookup(ctx, "app"); err == nil {
		t.Fatal("reader claimed receipt")
	}
	list, err := project.Bindings(ctx, *app.ResourceID)
	if err != nil || len(list) != 0 {
		t.Fatalf("cancelled reservation listed: %+v %v", list, err)
	}
}
