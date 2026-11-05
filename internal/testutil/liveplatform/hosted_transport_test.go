package liveplatform

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
	"github.com/spoke-sh/terraform-provider-spoke/internal/testutil/mockplatform"
)

func TestConsumerPreflightRejectsUnqualifiedHTTPBeforeMutation(t *testing.T) {
	s := mockplatform.NewBucketProfile(t)
	c := Config{Endpoint: s.HTTP.URL, Scope: s.Scope, CredentialFile: s.TokenFile, CAFile: s.CAFile, ExpectedPrincipal: "principal-admin"}
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	if _, err := ConsumerClient(ctx, c); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*Config){
		func(c *Config) { c.ExpectedPrincipal = "foreign" },
		func(c *Config) { c.Scope.EnvironmentID = "other" },
		func(c *Config) { c.CredentialFile = filepath.Join(t.TempDir(), "absent") },
		func(c *Config) { c.Endpoint = "http://example.invalid" },
	} {
		bad := c
		change(&bad)
		if _, err := ConsumerClient(ctx, bad); err == nil {
			t.Fatal("invalid consumer target accepted")
		}
	}
	plain := mockplatform.New(t)
	bad := c
	bad.Endpoint = plain.HTTP.URL
	bad.CAFile = plain.CAFile
	if _, err := ConsumerClient(ctx, bad); err == nil {
		t.Fatal("generic mock selected as bounded Platform profile")
	}
	for _, body := range []string{"<html>website</html>", "<Error><Code>AccessDenied</Code></Error>", `{"version":"mock-v5"}`} {
		target := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Spoke-API-Version", api.APIVersion)
			w.Header().Set("X-Spoke-Management-Profile", api.BucketProfile)
			_, _ = w.Write([]byte(body))
		}))
		bad := c
		bad.Endpoint = target.URL
		if _, err := ConsumerClient(ctx, bad); err == nil {
			t.Fatal("non-contract response accepted")
		}
		target.Close()
	}
	var redirects atomic.Int32
	destination := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { redirects.Add(1) }))
	defer destination.Close()
	origin := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, destination.URL, http.StatusTemporaryRedirect)
	}))
	defer origin.Close()
	bad = c
	bad.Endpoint = origin.URL
	if _, err := ConsumerClient(ctx, bad); err == nil || redirects.Load() != 0 {
		t.Fatal("consumer followed redirect")
	}
	bad = c
	bad.CAFile = filepath.Join(t.TempDir(), "bad-ca")
	if err := os.WriteFile(bad.CAFile, []byte("not a certificate"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ConsumerClient(ctx, bad); err == nil {
		t.Fatal("invalid TLS trust accepted")
	}
	for _, request := range s.Requests() {
		if request.Method != "GET" {
			t.Fatal("preflight dispatched mutation")
		}
	}
}
