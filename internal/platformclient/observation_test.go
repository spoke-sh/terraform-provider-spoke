package platformclient_test

import (
	"testing"

	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
	"github.com/spoke-sh/terraform-provider-spoke/internal/testutil/mockplatform"
)

func TestRetainedOperationObservationsDoNotMutate(t *testing.T) {
	s := mockplatform.New(t)
	c := client(t, s)
	m, err := c.InspectManagement(t.Context(), "bucket", "consumer-observation")
	if err != nil || m.Operation != nil {
		t.Fatal("missing key must be authoritative absence", err)
	}
	op := mustMutate(t, c, create("consumer-observation", "consumer-bucket", false))
	before := len(s.Requests())
	m, err = c.InspectManagement(t.Context(), "bucket", op.ManagementKey)
	if err != nil || m.Operation == nil || m.Operation.ID != op.ID {
		t.Fatal("lost retained operation", err)
	}
	observed, err := c.ObserveOperation(t.Context(), op)
	if err != nil || observed != op {
		t.Fatal("changed original operation", err)
	}
	for _, request := range s.Requests()[before:] {
		if request.Method != "GET" {
			t.Fatal("observation dispatched a mutation")
		}
	}
	for _, change := range []func(*api.Operation){
		func(o *api.Operation) { o.Scope.EnvironmentID = "other" },
		func(o *api.Operation) { o.Fingerprint = "bad" },
		func(o *api.Operation) { o.Generation++ },
		func(o *api.Operation) { o.OwnerID = "foreign" },
		func(o *api.Operation) { o.TargetID = "foreign" },
	} {
		bad := op
		change(&bad)
		if _, err := c.ObserveOperation(t.Context(), bad); err == nil {
			t.Fatal("changed operation identity accepted")
		}
	}
	if _, err := c.InspectManagement(t.Context(), "bucket", "../escape"); err == nil {
		t.Fatal("invalid receipt key accepted")
	}
	s.DenyAccess(true)
	if m, err := c.InspectManagement(t.Context(), "bucket", op.ManagementKey); err == nil || m.Operation != nil {
		t.Fatal("denied receipt converted to successful absence")
	}
}
