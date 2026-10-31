//go:build linux

package provider

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestAccMockInstallationFailures(t *testing.T) {
	_ = mockAcceptance(t)
	artifact := buildInstalledProvider(t)
	data, err := os.ReadFile(artifact)
	if err != nil {
		t.Fatal(err)
	}
	version := filepath.Base(filepath.Dir(filepath.Dir(artifact)))
	for _, failure := range []string{"checksum", "version", "platform"} {
		t.Run(failure, func(t *testing.T) {
			mirror := t.TempDir()
			binary := filepath.Join(mirror, "registry.terraform.io/spoke-sh/spoke", version, "linux_amd64", filepath.Base(artifact))
			if err := os.MkdirAll(filepath.Dir(binary), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(binary, data, 0700); err != nil {
				t.Fatal(err)
			}
			p := newConsumer(t, binary, "")
			lock, err := os.ReadFile(filepath.Join(p.dir, ".terraform.lock.hcl"))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.RemoveAll(filepath.Join(p.dir, ".terraform")); err != nil {
				t.Fatal(err)
			}
			switch failure {
			case "checksum":
				if err := os.WriteFile(binary, append(append([]byte{}, data...), byte(0)), 0700); err != nil {
					t.Fatal(err)
				}
			case "version":
				p.version = "0.0.0-unavailable"
				p.config("")
			case "platform":
				if err := os.Rename(filepath.Dir(binary), filepath.Join(filepath.Dir(filepath.Dir(binary)), "linux_arm64")); err != nil {
					t.Fatal(err)
				}
			}
			out, err := p.attempt("init", "-backend=false", "-input=false", "-lockfile=readonly", "-no-color")
			if err == nil {
				t.Fatalf("accepted %s", failure)
			}
			if failure == "checksum" && !bytes.Contains(out, []byte("checksum")) {
				t.Fatalf("wrong failure: %s", out)
			}
			if bytes.Contains(out, []byte("Failed to request discovery")) || bytes.Contains(out, []byte("could not connect")) {
				t.Fatalf("Spoke escaped mirror exclusion: %s", out)
			}
			after, err := os.ReadFile(filepath.Join(p.dir, ".terraform.lock.hcl"))
			if err != nil || !bytes.Equal(lock, after) {
				t.Fatal("failed install rewrote trusted lock")
			}
		})
	}
}
