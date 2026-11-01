package platformclient

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"regexp"
)

// ContractIdentity binds the executable and qualification tools to one HTTP
// schema. A provisional test contract is never publication evidence. Platform
// owns the schema and revision when a published contract is adopted.
type ContractIdentity struct {
	Version          string `json:"version"`
	Transport        string `json:"transport"`
	Status           string `json:"status"`
	SchemaSHA256     string `json:"schema_sha256"`
	PlatformRevision string `json:"platform_revision"`
}

//go:embed contract.json
var contractJSON []byte

var currentContract = loadContract()

// CurrentContract returns a copy; configuration cannot change the wire contract
// or turn a provisional artifact into a published one.
func CurrentContract() ContractIdentity { return currentContract }

func loadContract() ContractIdentity {
	var c ContractIdentity
	d := json.NewDecoder(bytes.NewReader(contractJSON))
	d.DisallowUnknownFields()
	if d.Decode(&c) != nil || d.Decode(new(any)) != io.EOF || c.Validate() != nil || c.Version != APIVersion {
		panic("invalid embedded Platform HTTP contract identity")
	}
	return c
}

func (c ContractIdentity) Validate() error {
	if !regexp.MustCompile(`^[a-z][a-z0-9._-]{1,127}$`).MatchString(c.Version) ||
		c.Transport != "https-json" || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(c.SchemaSHA256) {
		return errors.New("invalid Platform HTTP contract identity")
	}
	switch c.Status {
	case "provisional":
		if c.PlatformRevision != "" {
			return errors.New("provisional contract cannot claim a published Platform revision")
		}
	case "published":
		if !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(c.PlatformRevision) {
			return errors.New("published HTTP contract requires an immutable Platform revision")
		}
	default:
		return errors.New("unknown Platform HTTP contract status")
	}
	return nil
}
