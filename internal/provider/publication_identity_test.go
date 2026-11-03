package provider

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

// A model of the public contract's immutable, principal-bound receipt slots.
// Actual HTTP and process qualification separately exercise the implementation.
type publicationReceiptModel struct {
	principal string
	ops       map[string]api.PublicationOperation
	intents   map[string]string
	errors    map[string]error
	reads     []string
}

func newPublicationReceiptModel() *publicationReceiptModel {
	return &publicationReceiptModel{principal: "201", ops: map[string]api.PublicationOperation{}, intents: map[string]string{}, errors: map[string]error{}}
}
func (m *publicationReceiptModel) Capabilities(context.Context) (api.PublicationCapabilities, error) {
	return api.PublicationCapabilities{PrincipalID: m.principal, Capabilities: map[string]bool{"operations": true}, Limits: map[string]int64{"operations": 64}, ReceiptPolicy: "retain-or-tombstone"}, nil
}
func (m *publicationReceiptModel) Lookup(_ context.Context, key string) (api.PublicationOperation, error) {
	m.reads = append(m.reads, key)
	if err := m.errors[key]; err != nil {
		return api.PublicationOperation{}, err
	}
	if op, ok := m.ops[key]; ok {
		return op, nil
	}
	return api.PublicationOperation{}, &api.PublicationError{Status: 404, Code: "resource_not_found"}
}
func (m *publicationReceiptModel) accept(key, kind, resource, intent string) (api.PublicationOperation, error) {
	if op, ok := m.ops[key]; ok {
		if op.PrincipalID != m.principal || m.intents[key] != intent {
			return op, errors.New("operation_conflict")
		}
		return op, nil
	}
	id := fmt.Sprint(len(m.ops) + 1)
	op := api.PublicationOperation{ID: id, Key: key, Kind: kind, PrincipalID: m.principal, Status: "applied", ResourceID: &resource}
	m.ops[key], m.intents[key] = op, intent
	return op, nil
}

func TestPublicationIdentityRetainedSequence(t *testing.T) {
	m := newPublicationReceiptModel()
	ctx := context.Background()
	previous := ""
	for index, intent := range []string{"create:A", "update:B:revision1", "update:A:revision2", "delete:revision3", "create:A"} {
		key, err := publicationSuccessor("application", "site", previous)
		if err != nil {
			t.Fatal(err)
		}
		kind, id := "application_update", "100"
		if index == 0 || index == 4 {
			kind = "application_create"
		}
		if index == 3 {
			kind = "application_delete"
		}
		if index == 4 {
			id = "200"
		}
		op, err := m.accept(key, kind, id, intent)
		if err != nil {
			t.Fatal(err)
		}
		// Lose every caller-local field. A new process can discover the retained
		// incarnation and occupied slots from configuration and authorized reads.
		observed, err := publicationInspect(ctx, m, "application", "site")
		if err != nil || observed.Resource != id || len(observed.Operations) != index+1 {
			t.Fatalf("%+v: %v", observed, err)
		}
		replay, err := m.accept(key, kind, id, intent)
		if err != nil || replay.ID != op.ID {
			t.Fatal("replay replaced original identity")
		}
		if _, err := m.accept(key, kind, id, intent+":different"); err == nil {
			t.Fatal("conflicting intent reused an occupied slot")
		}
		previous = observed.lastKey()
	}
	if len(m.ops) != 5 {
		t.Fatal("return to A or recreation reused earlier intent")
	}
	// The earlier saved state still selects slot 01. It must not refresh to 05.
	stale, _ := publicationSuccessor("application", "site", "tf1a_site_00")
	if stale != "tf1a_site_01" {
		t.Fatal(stale)
	}
	if _, err := m.accept(stale, "application_update", "100", "stale:C:revision1"); err == nil {
		t.Fatal("stale plan mutated the replacement")
	}
}

func TestPublicationIdentityRefusesUncertainOrForeignEvidence(t *testing.T) {
	for _, scenario := range []string{"denied", "expired", "unavailable", "pending", "foreign", "wrong-kind", "wrong-resource"} {
		t.Run(scenario, func(t *testing.T) {
			m := newPublicationReceiptModel()
			key := "tf1a_site_00"
			op, _ := m.accept(key, "application_create", "100", "create:A")
			switch scenario {
			case "denied":
				m.errors[key] = &api.PublicationError{Code: "forbidden", Status: 403}
			case "expired":
				m.errors[key] = &api.PublicationError{Code: "receipt_expired", Status: 410}
			case "unavailable":
				m.errors[key] = &api.PublicationError{Code: "unavailable", Status: 503}
			case "pending":
				op.Status = "accepted"
				m.ops[key] = op
			case "foreign":
				m.principal = "202"
			case "wrong-kind":
				op.Kind = "target_create"
				m.ops[key] = op
			case "wrong-resource":
				_, _ = m.accept("tf1a_site_01", "application_update", "200", "update:B")
			}
			before := len(m.ops)
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
			defer cancel()
			if _, err := publicationInspect(ctx, m, "application", "site"); err == nil {
				t.Fatal("unsafe evidence accepted")
			}
			if len(m.ops) != before {
				t.Fatal("observation mutated receipts")
			}
		})
	}
}

func TestPublicationIdentityKeyBoundary(t *testing.T) {
	for _, kind := range []string{"application", "binding", "target"} {
		key, err := publicationKey(kind, "abcdefghijklmnopqrstuvwx", 63)
		if err != nil || len(key) != 32 {
			t.Fatalf("%q: %v", key, err)
		}
		management, sequence, err := publicationKeyParts(kind, key)
		if err != nil || management != "abcdefghijklmnopqrstuvwx" || sequence != 63 {
			t.Fatal("non-reversible key")
		}
		if _, err := publicationSuccessor(kind, management, key); err == nil {
			t.Fatal("reclaimed finite namespace")
		}
	}
	for _, key := range []string{"", "tf2a_site_00", "tf1t_site_00", "tf1a_site_0", "tf1a_site_64", "tf1a_site_+1", "tf1a_site_01_extra"} {
		if _, _, err := publicationKeyParts("application", key); err == nil {
			t.Fatalf("accepted %q", key)
		}
	}
	m := newPublicationReceiptModel()
	i, err := publicationInspect(context.Background(), m, "application", "site")
	if err != nil || len(i.Operations) != 0 || !reflect.DeepEqual(m.reads, []string{"tf1a_site_00"}) {
		t.Fatalf("unbounded empty lookup: %+v, %v", i, err)
	}
}

type publicationSettlingModel struct {
	*publicationReceiptModel
	observations int
}

func (m *publicationSettlingModel) Lookup(ctx context.Context, key string) (api.PublicationOperation, error) {
	op, err := m.publicationReceiptModel.Lookup(ctx, key)
	if err == nil && key == "tf1a_site_00" {
		m.observations++
		if m.observations == 1 {
			op.Status = "accepted"
			op.ResourceID = nil
		}
	}
	return op, err
}
func TestPublicationIdentityObservesOriginalSettlement(t *testing.T) {
	m := &publicationSettlingModel{publicationReceiptModel: newPublicationReceiptModel()}
	_, _ = m.accept("tf1a_site_00", "application_create", "100", "create:A")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	identity, err := publicationInspect(ctx, m, "application", "site")
	if err != nil || identity.Resource != "100" || len(identity.Operations) != 1 || m.observations != 2 {
		t.Fatalf("original settlement: %+v %v", identity, err)
	}
}
