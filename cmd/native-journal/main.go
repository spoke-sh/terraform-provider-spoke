// Explicit offline initialization; never invoked by Terraform configuration.
package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"os"

	"github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

func main() {
	log.SetFlags(0)
	if len(os.Args) != 2 {
		log.Fatal("usage: native-journal /private/fresh-principal.json")
	}
	f, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal("cannot open initialization manifest")
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, 16385))
	if err != nil || len(raw) > 16384 {
		log.Fatal("initialization manifest exceeds bound")
	}
	var input struct {
		FreshPrincipal bool                        `json:"fresh_unused_principal"`
		Endpoint       string                      `json:"endpoint"`
		Scope          platformclient.Scope        `json:"scope"`
		Native         platformclient.NativeConfig `json:"native"`
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(&input) != nil || d.Decode(new(any)) != io.EOF || !input.FreshPrincipal {
		log.Fatal("expected explicit fresh-unused-principal manifest")
	}
	if err = platformclient.InitializeNativeJournal(platformclient.Config{Endpoint: input.Endpoint, Scope: input.Scope, Native: &input.Native}); err != nil {
		log.Fatal(err)
	}
}
