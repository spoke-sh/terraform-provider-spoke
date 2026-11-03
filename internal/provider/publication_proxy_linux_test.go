//go:build linux

package provider

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
	"github.com/spoke-sh/terraform-provider-spoke/internal/testutil/mockplatform"
)

// This transport barrier forwards to the real, normal Platform API. It neither
// manufactures receipts nor modifies native data. Acknowledged receipt IDs are
// observed before a provider process is killed; fixture restart is separate.
type publicationProxy struct {
	server  *httptest.Server
	ca      string
	mu      sync.Mutex
	phase   string
	barrier *mockplatform.Barrier
	pending bool
}

func newPublicationProxy(t *testing.T, f publicationPlatformConfig) *publicationProxy {
	t.Helper()
	ca, err := os.ReadFile(f.CAFile)
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(ca) {
		t.Fatal("invalid fixture CA")
	}
	transport := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}}
	t.Cleanup(transport.CloseIdleConnections)
	p := &publicationProxy{}
	p.server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mutation := r.Method != "GET"
		if mutation {
			p.hold("before_accept", "")
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "request read", 502)
			return
		}
		request, err := http.NewRequest(r.Method, f.Endpoint+r.URL.RequestURI(), bytes.NewReader(body))
		if err != nil {
			http.Error(w, "request", 502)
			return
		}
		request.Header = r.Header.Clone()
		response, err := transport.RoundTrip(request)
		if err != nil {
			http.Error(w, "upstream", 502)
			return
		}
		defer response.Body.Close()
		data, err := io.ReadAll(response.Body)
		if err != nil {
			http.Error(w, "response", 502)
			return
		}
		var op api.PublicationOperation
		if response.StatusCode < 300 && json.Unmarshal(data, &op) == nil && op.ID != "" && op.Key != "" {
			if mutation {
				p.mu.Lock()
				p.pending = true
				p.mu.Unlock()
				p.hold("after_accept", op.ID)
			}
			p.mu.Lock()
			pending := p.pending
			p.mu.Unlock()
			if pending && op.Status == "applied" {
				p.hold("after_complete", op.ID)
			}
		}
		for k, v := range response.Header {
			w.Header()[k] = v
		}
		w.WriteHeader(response.StatusCode)
		_, _ = w.Write(data)
	}))
	t.Cleanup(p.server.Close)
	p.ca = filepath.Join(t.TempDir(), "proxy.pem")
	if err := os.WriteFile(p.ca, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: p.server.Certificate().Raw}), 0600); err != nil {
		t.Fatal(err)
	}
	return p
}
func (p *publicationProxy) arm(phase string) *mockplatform.Barrier {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.phase = phase
	p.pending = false
	p.barrier = &mockplatform.Barrier{Reached: make(chan api.Operation, 1), Release: make(chan struct{})}
	return p.barrier
}
func (p *publicationProxy) hold(phase, id string) {
	p.mu.Lock()
	b := p.barrier
	if p.phase != phase || b == nil {
		p.mu.Unlock()
		return
	}
	p.barrier = nil
	p.mu.Unlock()
	b.Reached <- api.Operation{ID: id}
	<-b.Release
}

func TestPublicationPlatformApplicationRecovery(t *testing.T) {
	f := publicationPlatformFixture(t)
	if os.Getenv("SPOKE_TEST_PROVIDER_MIRROR") == "" {
		t.Fatal("installed artifact required")
	}
	kind := os.Getenv("SPOKE_PUBLICATION_RECOVERY_KIND")
	if kind != "create" && kind != "update" && kind != "delete" {
		t.Fatal("explicit fresh fixture group required")
	}
	binary := buildInstalledProvider(t)
	for _, phase := range []string{"before_accept", "after_accept", "after_complete"} {
		t.Run(phase, func(t *testing.T) {
			proxy := newPublicationProxy(t, f)
			key := strings.ReplaceAll(kind+"-"+phase, "_", "-")
			config := func(name string) string {
				return strings.ReplaceAll(publicationApplicationConfig(proxy.server.URL, proxy.ca, f.Tokens["project"], name, false), "terraform-site", key)
			}
			p := newConsumer(t, binary, config("recovery-a"))
			if kind != "create" {
				p.apply()
			}
			if kind == "update" {
				p.config(config("recovery-b"))
			}
			b := proxy.arm(phase)
			args := []string{"apply", "-auto-approve", "-input=false", "-no-color"}
			if kind == "delete" {
				args[0] = "destroy"
			}
			op := p.killAt(b, args...)
			close(b.Release)
			if phase != "before_accept" && op.ID == "" {
				t.Fatal("receipt barrier lacks durable identity")
			}
			p.run(args...)
			client := publicationPlatformClient(t, f, "project", "")
			identity, err := publicationInspect(t.Context(), client, "application", key)
			want := 1
			if kind != "create" {
				want = 2
			}
			if err != nil || len(identity.Operations) != want {
				t.Fatalf("recovery duplicated or lost operation: %+v %v", identity, err)
			}
			if kind != "delete" {
				p.run("plan", "-detailed-exitcode", "-input=false", "-no-color")
				p.destroy()
			}
		})
	}
}
