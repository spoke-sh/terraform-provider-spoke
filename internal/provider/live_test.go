package provider

import (
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
	path := os.Getenv("SPOKE_LIVE_REPORT")
	if !filepath.IsAbs(path) || strings.HasPrefix(filepath.Clean(path)+"/", "/nix/store/") {
		t.Fatal("SPOKE_LIVE_REPORT must select an absolute external evidence path")
	}
	if c.Gate == "hosted-metadata" && (path == c.CredentialFile || path == c.CAFile || strings.HasPrefix(filepath.Clean(path)+"/", c.RunDirectory+"/") || strings.HasPrefix(filepath.Clean(path)+"/", c.HarnessCheckout+"/")) {
		t.Fatal("consumer summary must be separate from credentials, private run state and source")
	}
	report, gateErr := liveplatform.Preflight(c)
	if gateErr == nil {
		if c.Gate == "hosted-metadata" {
			report, gateErr = liveplatform.RunHosted(c, report)
		} else if c.Gate == "publication-declarations" {
			report, gateErr = liveplatform.RunDeclarations(c, report)
		} else {
			report, gateErr = liveplatform.RunMetadata(c, report)
		}
	}
	if err = liveplatform.WritePrivateJSON(path, report); err != nil {
		t.Fatal("cannot write live evidence")
	}
	if gateErr != nil {
		t.Fatal(gateErr)
	}
	if err = report.ValidateEvidence(c); err != nil {
		t.Fatal(err)
	}
}
