package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/spoke-sh/terraform-provider-spoke/internal/provider"
)

var version = "dev"

func main() {
	debug := flag.Bool("debug", false, "enable Terraform debugger support")
	flag.Parse()
	if err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/spoke-sh/spoke",
		Debug:   *debug,
	}); err != nil {
		log.Fatal(err)
	}
}
