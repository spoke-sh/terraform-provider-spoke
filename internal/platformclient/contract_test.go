package platformclient

import (
	"crypto/sha256"
	"fmt"
	"os"
	"testing"
)

func TestContractIdentityMatchesWireSchema(t *testing.T) {
	schema, err := os.ReadFile("../../testdata/contracts/mock-v5.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	c := CurrentContract()
	if c.Version != APIVersion || c.SchemaSHA256 != fmt.Sprintf("%x", sha256.Sum256(schema)) {
		t.Fatal("HTTP contract identity does not match the compiled client and schema")
	}
	if c.Status != "provisional" || c.PlatformRevision != "" {
		t.Fatal("the mock contract cannot claim Platform publication")
	}
	for _, change := range []func(*ContractIdentity){
		func(c *ContractIdentity) { c.Transport = "native" },
		func(c *ContractIdentity) { c.Status = "unknown" },
		func(c *ContractIdentity) { c.SchemaSHA256 = "floating" },
		func(c *ContractIdentity) { c.Status = "published" },
		func(c *ContractIdentity) { c.PlatformRevision = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" },
	} {
		bad := c
		change(&bad)
		if bad.Validate() == nil {
			t.Fatal("accepted invalid HTTP contract identity", bad)
		}
	}
}
