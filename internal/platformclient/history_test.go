package platformclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHistoryRejectsPartialAndUnboundResponses(t *testing.T) {
	scope := Scope{OrganizationID: "org", ProjectID: "project", EnvironmentID: "env"}
	valid := History{Scope: scope, ParentID: "job", Kind: "job", Complete: true, Items: []HistoryItem{{ID: "run", SourceID: "schedule", Revision: 1, Timestamp: 1, State: "active", Attempt: 1}}}
	raw, _ := json.Marshal(valid)
	bad := []string{
		strings.Replace(string(raw), `"complete":true`, `"complete":false`, 1),
		strings.Replace(string(raw), `"organization_id":"org"`, `"organization_id":"foreign"`, 1),
		strings.Replace(string(raw), `"parent_id":"job"`, `"parent_id":"other"`, 1),
		strings.Replace(string(raw), `"attempt":1`, `"attempt":0`, 1),
		strings.Replace(string(raw), `"attempt":1`, `"attempt":1,"secret":"canary"`, 1),
		strings.Replace(string(raw), `"attempt":1`, `"attempt":null`, 1),
	}
	duplicate := valid
	duplicate.Items = append(duplicate.Items, duplicate.Items[0])
	b, _ := json.Marshal(duplicate)
	bad = append(bad, string(b))
	for _, body := range bad {
		t.Run(body, func(t *testing.T) {
			s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Spoke-API-Version", APIVersion)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(body))
			}))
			defer s.Close()
			c, err := New(Config{Endpoint: s.URL, Scope: scope, Token: "token", HTTPClient: s.Client(), PollInterval: time.Millisecond})
			if err != nil {
				t.Fatal(err)
			}
			items, err := c.History(context.Background(), "job", "job")
			if err == nil || items != nil || strings.Contains(err.Error(), "canary") {
				t.Fatalf("accepted/leaked history: %v %v", items, err)
			}
		})
	}
}
