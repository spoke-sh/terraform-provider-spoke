//go:build linux

package provider

import (
	"bytes"
	"context"
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
	"time"

	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
	"github.com/spoke-sh/terraform-provider-spoke/internal/testutil/liveplatform"
	"github.com/spoke-sh/terraform-provider-spoke/internal/testutil/mockplatform"
)

// Consumer transport only. No server fixture controls are used, even in the
// synthetic lane. A successful mutation reply may already be terminal.
type metadataRelay struct {
	server  *httptest.Server
	ca      string
	mu      sync.Mutex
	phase   string
	barrier *mockplatform.Barrier
	writes  int
}

func newMetadataRelay(t *testing.T, c liveplatform.Config, client *api.Client) *metadataRelay {
	t.Helper()
	ca, err := os.ReadFile(c.CAFile)
	if err != nil {
		t.Fatal("cannot read relay trust")
	}
	roots, err := x509.SystemCertPool()
	if err != nil || !roots.AppendCertsFromPEM(ca) {
		t.Fatal("invalid relay trust")
	}
	transport := &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots}, ResponseHeaderTimeout: 10 * time.Second}
	t.Cleanup(transport.CloseIdleConnections)
	p := &metadataRelay{}
	p.server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, c.Scope.Path()+"/") || r.URL.IsAbs() || (r.Method != "GET" && r.Method != "POST" && r.Method != "PATCH" && r.Method != "DELETE") {
			http.Error(w, "scope", 400)
			return
		}
		mutation := r.Method != "GET"
		p.mu.Lock()
		phase, barrier := p.phase, p.barrier
		p.mu.Unlock()
		hold := func(op api.Operation) bool {
			if barrier == nil {
				return true
			}
			select {
			case barrier.Reached <- op:
			case <-r.Context().Done():
				return false
			}
			select {
			case <-barrier.Release:
				return r.Context().Err() == nil
			case <-r.Context().Done():
				return false
			}
		}
		if mutation && phase == "before_forward" {
			// Never forward this deliberately intercepted request, even if its
			// caller has not yet closed the socket when the barrier is released.
			hold(api.Operation{})
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, (1<<20)+1))
		if err != nil || len(body) > 1<<20 {
			http.Error(w, "request", 400)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, r.Method, c.Endpoint+r.URL.RequestURI(), bytes.NewReader(body))
		if err != nil {
			http.Error(w, "request", 400)
			return
		}
		req.Header = r.Header.Clone()
		if mutation {
			p.mu.Lock()
			p.writes++
			p.mu.Unlock()
		}
		res, err := transport.RoundTrip(req)
		if err != nil {
			http.Error(w, "upstream", 502)
			return
		}
		defer res.Body.Close()
		data, err := io.ReadAll(io.LimitReader(res.Body, (1<<20)+1))
		if err != nil || len(data) > 1<<20 {
			http.Error(w, "response", 502)
			return
		}
		if mutation && barrier != nil && res.StatusCode >= 200 && res.StatusCode < 300 {
			var op api.Operation
			if json.Unmarshal(data, &op) != nil || op.ID == "" {
				http.Error(w, "receipt", 502)
				return
			}
			if phase == "after_success" {
				// Observe the exact operation through a separate verified TLS
				// client while the provider's mutation reply remains withheld.
				for op.Status == "accepted" {
					select {
					case <-ctx.Done():
						return
					case <-time.After(20 * time.Millisecond):
					}
					op, err = client.ObserveOperation(ctx, op)
					if err != nil {
						http.Error(w, "observation", 502)
						return
					}
				}
				if op.Status != "succeeded" {
					http.Error(w, "settlement", 502)
					return
				}
			}
			if !hold(op) {
				return
			}
		}
		for k, v := range res.Header {
			w.Header()[k] = v
		}
		w.WriteHeader(res.StatusCode)
		_, _ = w.Write(data)
	}))
	t.Cleanup(p.server.Close)
	p.ca = filepath.Join(t.TempDir(), "relay.pem")
	if err := os.WriteFile(p.ca, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: p.server.Certificate().Raw}), 0600); err != nil {
		t.Fatal(err)
	}
	return p
}

func (p *metadataRelay) arm(phase string) *mockplatform.Barrier {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.phase = phase
	p.barrier = &mockplatform.Barrier{Reached: make(chan api.Operation, 1), Release: make(chan struct{})}
	return p.barrier
}
func (p *metadataRelay) disarm()    { p.mu.Lock(); defer p.mu.Unlock(); p.phase = ""; p.barrier = nil }
func (p *metadataRelay) count() int { p.mu.Lock(); defer p.mu.Unlock(); return p.writes }
