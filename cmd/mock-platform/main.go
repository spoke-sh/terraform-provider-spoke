// mock-platform is a synthetic fixture launcher, never a deployment service.
package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spoke-sh/terraform-provider-spoke/internal/testutil/mockplatform"
)

type fixtureRuntime struct {
	dir     string
	cleanup []func()
}

func (*fixtureRuntime) Helper()                        {}
func (r *fixtureRuntime) TempDir() string              { return r.dir }
func (*fixtureRuntime) Fatal(v ...any)                 { log.Fatal(v...) }
func (*fixtureRuntime) Errorf(format string, v ...any) { log.Printf(format, v...) }
func (r *fixtureRuntime) Cleanup(f func())             { r.cleanup = append(r.cleanup, f) }

func main() {
	dir := flag.String("directory", "", "new absolute private runtime directory outside the Nix store")
	flag.Parse()
	if !filepath.IsAbs(*dir) {
		log.Fatal("an absolute runtime directory is required")
	}
	// Refuse reuse: this launcher seeds a new isolated fixture, never replaces a ledger.
	if err := os.Mkdir(*dir, 0700); err != nil {
		log.Fatal("cannot create fresh runtime directory")
	}
	runtime := &fixtureRuntime{dir: *dir}
	s := mockplatform.New(runtime)
	defer func() {
		for _, cleanup := range runtime.cleanup {
			cleanup()
		}
	}()
	config := map[string]string{"endpoint": s.HTTP.URL, "organization_id": s.Scope.OrganizationID, "project_id": s.Scope.ProjectID, "environment_id": s.Scope.EnvironmentID, "token_file": s.TokenFile, "ca_file": s.CAFile}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(*dir, "fixture.json"), data, 0600); err != nil {
		log.Fatal(err)
	}
	log.Print("synthetic TLS fixture ready")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
}
