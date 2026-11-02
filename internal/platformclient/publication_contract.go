package platformclient

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

const PublicationVersion = "publication-v1"
const PublicationProfile = "spoke.platform.http.publication.v1"

// PublicationContractIdentity describes client conformance, not Terraform
// resources, hosted delivery or qualification of the full provider inventory.
type PublicationContractIdentity struct {
	Name             string `json:"name"`
	Version          string `json:"version"`
	Status           string `json:"status"`
	PlatformRevision string `json:"platform_revision"`
	SchemaSHA256     string `json:"schema_sha256"`
}

//go:embed publication-contract/profile.json
var publicationIdentityJSON []byte

//go:embed publication-contract/schema.json
var publicationSchemaJSON []byte

func CurrentPublicationContract() PublicationContractIdentity {
	var identity PublicationContractIdentity
	d := json.NewDecoder(bytes.NewReader(publicationIdentityJSON))
	d.DisallowUnknownFields()
	if d.Decode(&identity) != nil || d.Decode(new(any)) != io.EOF || identity.Name != PublicationProfile || identity.Version != PublicationVersion || identity.Status != "development-declarations" || len(identity.PlatformRevision) != 40 || identity.SchemaSHA256 != fmt.Sprintf("%x", sha256.Sum256(publicationSchemaJSON)) {
		panic("invalid embedded publication contract")
	}
	return identity
}

type publicationOfflineLoader struct{}

func (publicationOfflineLoader) Load(string) (any, error) {
	return nil, errors.New("external schemas are disabled")
}

var publicationSchemas = sync.OnceValue(func() map[string]*jsonschema.Schema {
	_ = CurrentPublicationContract()
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(publicationSchemaJSON))
	if err != nil {
		panic("invalid publication schema")
	}
	c := jsonschema.NewCompiler()
	c.UseLoader(publicationOfflineLoader{})
	const uri = "https://spoke.sh/contracts/http/publication-v1.schema.json"
	if err := c.AddResource(uri, doc); err != nil {
		panic("invalid publication schema resource")
	}
	result := map[string]*jsonschema.Schema{}
	for name := range doc.(map[string]any)["$defs"].(map[string]any) {
		s, err := c.Compile(uri + "#/$defs/" + name)
		if err != nil {
			panic("invalid publication schema definition")
		}
		result[name] = s
	}
	return result
})

// Decode before schema validation to reject duplicate keys, excessive nesting
// and trailing values. Validation errors never disclose remote input/credentials.
func publicationJSON(d *json.Decoder, depth int) (any, error) {
	if depth > 24 {
		return nil, errors.New("excessive JSON nesting")
	}
	token, err := d.Token()
	if err != nil {
		return nil, err
	}
	switch token {
	case json.Delim('{'):
		object := map[string]any{}
		for d.More() {
			key, err := d.Token()
			if err != nil {
				return nil, err
			}
			name, ok := key.(string)
			if !ok {
				return nil, errors.New("invalid object key")
			}
			if _, exists := object[name]; exists {
				return nil, errors.New("duplicate object key")
			}
			value, err := publicationJSON(d, depth+1)
			if err != nil {
				return nil, err
			}
			object[name] = value
		}
		end, err := d.Token()
		if err != nil || end != json.Delim('}') {
			return nil, errors.New("incomplete object")
		}
		return object, nil
	case json.Delim('['):
		array := []any{}
		for d.More() {
			value, err := publicationJSON(d, depth+1)
			if err != nil {
				return nil, err
			}
			array = append(array, value)
		}
		end, err := d.Token()
		if err != nil || end != json.Delim(']') {
			return nil, errors.New("incomplete array")
		}
		return array, nil
	default:
		if _, ok := token.(json.Delim); ok {
			return nil, errors.New("unexpected delimiter")
		}
		return token, nil
	}
}

// The owned schema expresses decimal syntax; its documented native bound also
// applies to IDs, revisions and timestamps. No floating point conversion occurs.
func publicationCounters(value any) bool {
	switch v := value.(type) {
	case map[string]any:
		for key, item := range v {
			switch key {
			case "id", "revision", "expected_revision", "application_id", "binding_id", "target_id", "principal_id", "resource_id", "generation", "expected_generation", "desired_generation", "observed_generation", "observation_revision", "expected_observation_revision", "last_observation_ms", "expires_at_ms", "snapshot":
				if text, ok := item.(string); ok {
					if !publicationDecimal(text, true) {
						return false
					}
				}
			}
			if !publicationCounters(item) {
				return false
			}
		}
	case []any:
		for _, item := range v {
			if !publicationCounters(item) {
				return false
			}
		}
	}
	return true
}

func publicationDecimal(value string, zero bool) bool {
	if value == "0" {
		return zero
	}
	if len(value) == 0 || len(value) > 16 || value[0] == '0' {
		return false
	}
	for _, c := range value {
		if c < '0' || c > '9' {
			return false
		}
	}
	n, err := strconv.ParseUint(value, 10, 64)
	return err == nil && n > 0 && n <= uint64(MaxProfileCounter)
}

func decodePublication(data []byte, kind string, out any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	value, err := publicationJSON(d, 0)
	if err != nil {
		return errors.New("invalid publication JSON")
	}
	if _, err := d.Token(); err != io.EOF {
		return errors.New("invalid publication JSON")
	}
	schema := publicationSchemas()[kind]
	if schema == nil || schema.Validate(value) != nil || !publicationCounters(value) {
		return errors.New("invalid publication contract value")
	}
	if out != nil && json.Unmarshal(data, out) != nil {
		return errors.New("invalid publication typed value")
	}
	return nil
}
