package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"runtime"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
	"github.com/spoke-sh/terraform-provider-spoke/internal/provider"
)

var version = "dev"
var revision = "unknown"

func main() {
	debug := flag.Bool("debug", false, "enable Terraform debugger support")
	showVersion := flag.Bool("version", false, "print artifact identity as JSON and exit")
	flag.Parse()
	if *showVersion {
		contract := platformclient.CurrentContract()
		if err := json.NewEncoder(os.Stdout).Encode(map[string]string{
			"version": version, "revision": revision, "go": runtime.Version(), "platform": runtime.GOOS + "_" + runtime.GOARCH,
			"contract": contract.Version, "transport": contract.Transport, "contract_status": contract.Status,
			"contract_schema_sha256": contract.SchemaSHA256, "platform_revision": contract.PlatformRevision,
		}); err != nil {
			log.Fatal(err)
		}
		return
	}
	if err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/spoke-sh/spoke",
		Debug:   *debug,
	}); err != nil {
		log.Fatal(err)
	}
}
