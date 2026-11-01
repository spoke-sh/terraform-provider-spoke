//go:build linux

package provider

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAccMockRejectsPriorStateVersions(t *testing.T) {
	s := mockAcceptance(t)
	p := newConsumer(t, buildInstalledProvider(t), bucketConfig(s, "current", "deletion_protection = false"))
	p.apply()
	original := p.run("state", "pull")
	operations := s.OperationCount()
	for _, version := range []int{0, 1, 2} {
		var state map[string]any
		if err := json.Unmarshal(original, &state); err != nil {
			t.Fatal(err)
		}
		for _, r := range state["resources"].([]any) {
			resource := r.(map[string]any)
			if resource["mode"] != "managed" {
				continue
			}
			for _, instance := range resource["instances"].([]any) {
				instance.(map[string]any)["schema_version"] = version
			}
		}
		old, _ := json.Marshal(state)
		path := filepath.Join(p.dir, "terraform.tfstate")
		if err := os.WriteFile(path, old, 0600); err != nil {
			t.Fatal(err)
		}
		out, err := p.attempt("plan", "-input=false", "-no-color")
		if err == nil || !bytes.Contains(out, []byte("Unable to Upgrade Resource State")) {
			t.Fatalf("old state was accepted: %v\n%s", err, out)
		}
		after, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(old, after) || operations != s.OperationCount() {
			t.Fatal("failed upgrade changed state or service")
		}
	}
	if err := os.WriteFile(filepath.Join(p.dir, "terraform.tfstate"), original, 0600); err != nil {
		t.Fatal(err)
	}
	p.destroy()
}
