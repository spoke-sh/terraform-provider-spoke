package provider

import (
	"encoding/json"
	"github.com/spoke-sh/terraform-provider-spoke/internal/testutil/liveplatform"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAccLiveQualification(t *testing.T) {
	if os.Getenv("TF_ACC") != "1" || os.Getenv("SPOKE_TEST_TARGET") != "live" {
		t.Skip("explicit live lane only")
	}
	c, err := liveplatform.Load(os.Getenv("SPOKE_LIVE_CONFIG"))
	if err != nil {
		t.Fatal(err)
	}
	report, gateErr := liveplatform.Preflight(c)
	path := os.Getenv("SPOKE_LIVE_REPORT")
	if !filepath.IsAbs(path) || strings.HasPrefix(filepath.Clean(path)+"/", "/nix/store/") {
		t.Fatal("SPOKE_LIVE_REPORT must select an absolute external evidence path")
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, append(data, '\n'), 0600); err != nil {
		t.Fatal("cannot write live evidence")
	}
	if gateErr != nil {
		t.Fatal(gateErr)
	}
	if err = report.ValidateEvidence(c); err != nil {
		t.Fatal(err)
	}
}
