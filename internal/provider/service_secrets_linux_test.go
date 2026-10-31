//go:build linux

package provider

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

func TestAccMockSecretCanariesThroughRotationAndFailure(t *testing.T) {
	s := mockAcceptance(t)
	d := s.ServiceDependencies(t)
	config := serviceConfig(s, d, "secret_reference", "reference")
	p := newConsumer(t, buildInstalledProvider(t), config)
	logPath := filepath.Join(p.dir, "provider.log")
	p.env = append(p.env, "TF_LOG=TRACE", "TF_LOG_PATH="+logPath)
	canary := "SYNTHETIC_EXTERNAL_SECRET_PAYLOAD_NOT_A_REFERENCE"
	s.SetErrorCanary(canary)
	for _, version := range []string{"v1", "v2"} {
		p.config(strings.Replace(config, `version = "v1"`, `version = "`+version+`"`, 1))
		p.apply()
		// Discover the managed reference independently from the actual fixture requests.
		var id string
		for _, request := range s.Requests() {
			if request.Method == "GET" && strings.HasPrefix(request.Path, s.Scope.Path()+"/secret_references/") {
				id = strings.TrimPrefix(request.Path, s.Scope.Path()+"/secret_references/")
			}
		}
		if id == "" {
			t.Fatal("reference not read")
		}
		s.FailNext("GET", s.Scope.Path()+"/secret_references/"+id, 503, "unavailable", 3)
		failed, err := p.attempt("plan", "-input=false", "-no-color")
		if err == nil {
			t.Fatal("injected read failure succeeded")
		}
		if bytes.Contains(failed, []byte(canary)) {
			t.Fatal("raw secret error reached diagnostics")
		}
		s.SetSecretVersion(api.SecretSpec{ExternalOwner: "external", ExternalID: "credential", Version: version}, true)
		s.AdvanceControllers()
		p.run("plan", "-input=false", "-no-color", "-out=review.plan")
		plan := p.run("show", "-json", "review.plan")
		state := p.run("state", "pull")
		logs, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatal(err)
		}
		for _, output := range [][]byte{failed, plan, state, logs} {
			for _, secret := range []string{canary, s.Token} {
				if bytes.Contains(output, []byte(secret)) {
					t.Fatal("secret or credential found in state, plan, diagnostics or logs")
				}
			}
		}
	}
	p.destroy()
}
