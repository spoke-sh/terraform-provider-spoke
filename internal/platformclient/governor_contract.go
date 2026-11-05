package platformclient

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

const GovernorVersion = "ci-governor-v1"
const GovernorProfile = "spoke.platform.http.ci-governor.v1"
const governorBodyLimit = 16384
const governorResponseLimit = 131072

type GovernorContractIdentity struct {
	Name             string `json:"name"`
	Version          string `json:"version"`
	Status           string `json:"status"`
	PlatformRevision string `json:"platform_revision"`
	SchemaSHA256     string `json:"schema_sha256"`
	VectorsSHA256    string `json:"vectors_sha256"`
	ProfileSHA256    string `json:"profile_sha256"`
}

//go:embed governor-contract/profile.json
var governorIdentityJSON []byte

//go:embed governor-contract/schema.json
var governorSchemaJSON []byte

//go:embed governor-contract/vectors.json
var governorVectorsJSON []byte

//go:embed governor-contract/upstream-profile.md
var governorProfileText []byte

func CurrentGovernorContract() GovernorContractIdentity {
	var p GovernorContractIdentity
	d := json.NewDecoder(bytes.NewReader(governorIdentityJSON))
	d.DisallowUnknownFields()
	if d.Decode(&p) != nil || d.Decode(new(any)) != io.EOF {
		panic("invalid governor contract identity")
	}
	r, err := hex.DecodeString(p.PlatformRevision)
	if err != nil || len(r) != 20 || p.Name != GovernorProfile || p.Version != GovernorVersion || p.Status != "provisional" || p.SchemaSHA256 != fmt.Sprintf("%x", sha256.Sum256(governorSchemaJSON)) || p.VectorsSHA256 != fmt.Sprintf("%x", sha256.Sum256(governorVectorsJSON)) || p.ProfileSHA256 != fmt.Sprintf("%x", sha256.Sum256(governorProfileText)) {
		panic("invalid governor contract pin")
	}
	return p
}

var governorSchemas = sync.OnceValue(func() map[string]*jsonschema.Schema {
	_ = CurrentGovernorContract()
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(governorSchemaJSON))
	if err != nil {
		panic("invalid governor schema")
	}
	c := jsonschema.NewCompiler()
	c.UseLoader(publicationOfflineLoader{})
	const uri = "https://spoke.sh/contracts/http/ci-governor-v1.schema.json"
	if c.AddResource(uri, doc) != nil {
		panic("invalid governor schema resource")
	}
	out := map[string]*jsonschema.Schema{}
	for name := range doc.(map[string]any)["$defs"].(map[string]any) {
		s, err := c.Compile(uri + "#/$defs/" + name)
		if err != nil {
			panic("invalid governor schema definition")
		}
		out[name] = s
	}
	return out
})

// The shared token walker only enforces JSON structure; governor semantics and
// schema are independent of publication and bucket identity algorithms.
func decodeGovernor(raw []byte, kind string, out any) error {
	if len(raw) > governorResponseLimit {
		return errors.New("invalid governor response size")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	v, err := publicationJSON(d, 0)
	if err != nil {
		return errors.New("invalid governor JSON")
	}
	if _, err := d.Token(); err != io.EOF {
		return errors.New("invalid governor JSON")
	}
	s := governorSchemas()[kind]
	if s == nil || s.Validate(v) != nil {
		return errors.New("invalid governor contract value")
	}
	if out != nil && json.Unmarshal(raw, out) != nil {
		return errors.New("invalid governor typed value")
	}
	return nil
}

func governorValue(kind string, value any) bool {
	raw, err := json.Marshal(value)
	return err == nil && decodeGovernor(raw, kind, nil) == nil
}

// ValidateGovernorPolicy checks Terraform inputs against the same independently
// pinned policy schema as the HTTP boundary, without making a request.
func ValidateGovernorPolicy(policy GovernorPolicy) error {
	if !governorValue("Policy", policy) {
		return errors.New("policy does not satisfy the pinned ci-governor-v1 bounds or supported profile")
	}
	return nil
}

// GovernorFingerprint implements only the Platform-owned public JSON commitment.
// It creates no request identity and has no dependency on a native codec.
func GovernorFingerprint(scope, principal string, request GovernorRequest) (string, error) {
	if !governorValue("Scope", scope) || !governorValue("ID", principal) || !governorValue("Request", request) {
		return "", errors.New("invalid governor commitment input")
	}
	v := struct {
		Scope     string `json:"scope"`
		Principal string `json:"principal_id"`
		GovernorRequest
	}{scope, principal, request}
	raw, _ := json.Marshal(v)
	// Decode to maps to sort every object key. Keep numbers exact, without float64.
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var canonical any
	if d.Decode(&canonical) != nil {
		return "", errors.New("invalid governor commitment")
	}
	raw, err := json.Marshal(canonical)
	if err != nil {
		return "", errors.New("invalid governor commitment")
	}
	return fmt.Sprintf("%x", sha256.Sum256(append([]byte(GovernorProfile+"\n"), raw...))), nil
}
