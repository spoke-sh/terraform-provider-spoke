package platformclient

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestPlatformProfileProvenanceAndIndependentFingerprints(t *testing.T) {
	p := CurrentProfile()
	for file, digest := range map[string]string{"schema.json": p.SchemaSHA256, "vectors.json": p.VectorsSHA256} {
		data, err := os.ReadFile("../../testdata/contracts/platform-bucket-v1/" + file)
		if err != nil || fmt.Sprintf("%x", sha256.Sum256(data)) != digest {
			t.Fatalf("profile source drift: %s %v", file, err)
		}
	}
	data, _ := os.ReadFile("../../testdata/contracts/platform-bucket-v1/vectors.json")
	var vectors struct {
		Profile string `json:"profile"`
		Vectors []struct {
			Request   Mutation `json:"request"`
			Canonical string   `json:"canonical"`
			SHA256    string   `json:"sha256"`
		} `json:"vectors"`
	}
	if err := json.Unmarshal(data, &vectors); err != nil {
		t.Fatal(err)
	}
	if vectors.Profile != p.Name || len(vectors.Vectors) != 4 {
		t.Fatal("missing independent vectors")
	}
	for _, v := range vectors.Vectors {
		if v.Request.Fingerprint() != v.SHA256 {
			t.Fatal("fingerprint differs from Platform", v.Request.ManagementKey)
		}
		encoded, _ := json.Marshal(v.Request.Intent)
		if APIVersion+"\n"+string(encoded) != v.Canonical {
			t.Fatal("canonical bytes differ from Platform")
		}
		v.Request.ExpectedGeneration++
		if v.Request.Fingerprint() != v.SHA256 {
			t.Fatal("generation incorrectly changes intent")
		}
	}
	if CurrentContract().Status != "provisional" {
		t.Fatal("subset promoted full inventory")
	}
}

var profileScope = Scope{OrganizationID: "org", ProjectID: "project", EnvironmentID: "env"}

func profilePeer(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer synthetic" || r.Header.Get("X-Spoke-Organization") != "org" || r.Header.Get("X-Spoke-API-Version") != APIVersion {
			t.Error("missing scoped authentication")
		}
		w.Header().Set("X-Spoke-API-Version", APIVersion)
		w.Header().Set("X-Spoke-Management-Profile", BucketProfile)
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	c, err := New(Config{Endpoint: srv.URL, Scope: profileScope, Token: "synthetic", HTTPClient: srv.Client(), PollInterval: time.Millisecond, ManagementProfile: BucketProfile})
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func profileCaps() Capabilities {
	return Capabilities{Scope: profileScope, Version: APIVersion, PrincipalID: "owner", Resources: []string{"bucket"}, ReceiptPolicy: "retain-or-tombstone"}
}
func pendingOperation() Operation {
	return Operation{Scope: profileScope, ID: "operation", ManagementKey: "key", Fingerprint: strings.Repeat("a", 64), Kind: "create", TargetID: "bucket", Generation: 1, Status: "accepted", ResourceKind: "bucket", OwnerID: "owner"}
}
func pendingPage() PendingPage {
	return PendingPage{Scope: profileScope, ResourceKind: "bucket", Items: []Operation{pendingOperation()}, Complete: true, Snapshot: 1}
}

func TestPendingBucketsCompleteAuthorizedObservation(t *testing.T) {
	for _, variant := range []string{"valid", "foreign", "wrong-scope", "duplicate-id", "duplicate-key", "terminal", "generation", "incomplete", "continuation", "over-limit", "null", "missing-token", "unknown", "denied", "transient"} {
		t.Run(variant, func(t *testing.T) {
			requests := 0
			c := profilePeer(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Fatal("observation mutated")
				}
				if strings.HasSuffix(r.URL.Path, "/capabilities") {
					_ = json.NewEncoder(w).Encode(profileCaps())
					return
				}
				if r.URL.Path != profileScope.Path()+"/management/bucket" || r.URL.RawQuery != "state=accepted" {
					t.Error("wrong pending route", r.URL)
				}
				requests++
				if variant == "denied" {
					w.WriteHeader(403)
					fmt.Fprint(w, `{"code":"forbidden"}`)
					return
				}
				if variant == "transient" && requests < 3 {
					w.WriteHeader(503)
					fmt.Fprint(w, `{"code":"unavailable"}`)
					return
				}
				p := pendingPage()
				switch variant {
				case "foreign":
					p.Items[0].OwnerID = "another"
				case "wrong-scope":
					p.EnvironmentID = "another"
				case "duplicate-id":
					p.Items = append(p.Items, p.Items[0])
					p.Items[1].ManagementKey = "other"
				case "duplicate-key":
					p.Items = append(p.Items, p.Items[0])
					p.Items[1].ID = "other"
				case "terminal":
					p.Items[0].Status = "succeeded"
				case "generation":
					p.Items[0].Generation = MaxProfileCounter + 1
				case "incomplete":
					p.Complete = false
				case "continuation":
					p.NextToken = "more"
				case "over-limit":
					for len(p.Items) < 33 {
						p.Items = append(p.Items, p.Items[0])
					}
				}
				data, _ := json.Marshal(p)
				var wire map[string]any
				_ = json.Unmarshal(data, &wire)
				switch variant {
				case "null":
					wire["items"] = nil
				case "missing-token":
					delete(wire, "next_token")
				case "unknown":
					wire["future"] = true
				}
				_ = json.NewEncoder(w).Encode(wire)
			})
			page, err := c.PendingBuckets(context.Background())
			if variant == "valid" || variant == "transient" {
				if err != nil || len(page.Items) != 1 {
					t.Fatal(page, err)
				}
			} else if err == nil || len(page.Items) != 0 {
				t.Fatal("accepted invalid or partial result", page, err)
			}
			want := 1
			if variant == "transient" {
				want = 3
			}
			if requests != want {
				t.Fatal("unexpected retry", requests)
			}
		})
	}
}

func TestProfileRefusesUnknownMissingAndChangingHeaders(t *testing.T) {
	for _, name := range []string{"", "unknown", BucketProfile} {
		t.Run(name, func(t *testing.T) {
			calls := 0
			c := profilePeer(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.Header().Set("X-Spoke-Management-Profile", name)
				_ = json.NewEncoder(w).Encode(profileCaps())
			})
			_, err := c.Capabilities(context.Background())
			if (err == nil) != (name == BucketProfile) || calls != 1 {
				t.Fatal("profile mismatch accepted or retried", err, calls)
			}
		})
	}
	c := profilePeer(t, func(w http.ResponseWriter, r *http.Request) { _ = json.NewEncoder(w).Encode(profileCaps()) })
	if _, err := c.Capabilities(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := c.profile.bind(""); err == nil {
		t.Fatal("endpoint profile switched")
	}
}

func TestProfileInputBoundsBeforeDispatch(t *testing.T) {
	valid := Intent{Kind: "create", ResourceKind: "bucket", ManagementKey: strings.Repeat("k", 128), Spec: &Spec{Name: "abc", Storage: &StorageSpec{Profile: "object", CapacityBytes: 1}}}
	for _, variant := range []string{"nil-spec", "short-name", "unicode", "uppercase", "capacity", "retention", "wrong-kind", "revision"} {
		t.Run(variant, func(t *testing.T) {
			i := valid
			s := *valid.Spec
			storage := *s.Storage
			s.Storage = &storage
			i.Spec = &s
			switch variant {
			case "nil-spec":
				i.Spec = nil
			case "short-name":
				s.Name = "ab"
			case "unicode":
				s.Name = "äbc"
			case "uppercase":
				s.Name = "Abc"
			case "capacity":
				storage.CapacityBytes = MaxCapacity + 1
			case "retention":
				storage.RetentionDays = 36501
			case "wrong-kind":
				i.ResourceKind = "database"
			case "revision":
				i.ExpectedRevision = MaxProfileCounter + 1
			}
			c := profilePeer(t, func(w http.ResponseWriter, r *http.Request) {
				if !strings.HasSuffix(r.URL.Path, "/capabilities") {
					t.Error("invalid intent reached admission")
				}
				_ = json.NewEncoder(w).Encode(profileCaps())
			})
			if _, err := c.Mutate(context.Background(), i); err == nil {
				t.Fatal("invalid intent accepted")
			}
		})
	}
	for _, name := range []string{"abc", strings.Repeat("a", 63)} {
		valid.Spec.Name = name
		valid.Spec.Storage.CapacityBytes = MaxCapacity
		valid.Spec.Storage.RetentionDays = 36500
		if !validBucketProfileIntent(valid) {
			t.Fatal("boundary rejected")
		}
	}
}
