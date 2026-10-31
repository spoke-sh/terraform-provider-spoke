package platformclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"reflect"
	"regexp"
	"slices"
	"strings"
)

var ServiceCollections = map[string]string{
	"artifact_registry": "artifact_registries", "database": "databases",
	"secret_reference": "secret_references", "service_binding": "service_bindings",
	"container_service": "container_services", "endpoint": "endpoints",
}

func Kinds() []string {
	return []string{"project", "environment", "service_account", "grant", "bucket", "artifact_registry", "database", "secret_reference", "service_binding", "container_service", "endpoint"}
}
func ServiceKind(kind string) bool { _, ok := ServiceCollections[kind]; return ok }

const DefaultCapacity int64 = 1 << 30
const MaxCapacity int64 = 1 << 50

type RegistrySpec struct {
	Format        string `json:"format"`
	CapacityBytes int64  `json:"capacity_bytes"`
	RetentionDays int64  `json:"retention_days"`
}
type StorageSpec struct {
	Profile       string `json:"profile"`
	CapacityBytes int64  `json:"capacity_bytes"`
	RetentionDays int64  `json:"retention_days"`
}
type SecretSpec struct {
	ExternalOwner string `json:"external_owner"`
	ExternalID    string `json:"external_id"`
	Version       string `json:"version"`
}
type BindingSpec struct {
	ConsumerID string `json:"consumer_id"`
	TargetKind string `json:"target_kind"`
	TargetID   string `json:"target_id"`
	Slot       string `json:"slot"`
	Intent     string `json:"intent"`
}
type ContainerSpec struct {
	RegistryID       string            `json:"registry_id"`
	Repository       string            `json:"repository"`
	Digest           string            `json:"digest"`
	ServiceAccountID string            `json:"service_account_id"`
	CPU              int64             `json:"cpu_millis"`
	Memory           int64             `json:"memory_mib"`
	Configuration    map[string]string `json:"configuration"`
	SecretReferences map[string]string `json:"secret_references"`
	RequiredBindings []string          `json:"required_bindings"`
}
type EndpointSpec struct {
	ServiceID string `json:"service_id"`
	Port      int64  `json:"port"`
	Exposure  string `json:"exposure"`
	TLS       string `json:"tls_policy"`
}

// Status is an observation; controllers alone advance effective/active revisions.
type ServiceStatus struct {
	Endpoint          string `json:"endpoint"`
	Protocol          string `json:"protocol"`
	Readiness         string `json:"readiness"`
	Reason            string `json:"reason"`
	DesiredRevision   int64  `json:"desired_revision"`
	EffectiveRevision int64  `json:"effective_revision"`
	ActiveRevision    int64  `json:"active_revision"`
	ResolvedVersion   string `json:"resolved_version"`
	TLSState          string `json:"tls_state"`
	OperationID       string `json:"operation_id"`
}

var digestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
var repositoryPattern = regexp.MustCompile(`^[a-z0-9]+([._-][a-z0-9]+)*(/[a-z0-9]+([._-][a-z0-9]+)*)*$`)
var tagPattern = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9_.-]{0,127}$`)

func ValidDigest(v string) bool     { return digestPattern.MatchString(v) }
func ValidRepository(v string) bool { return len(v) <= 255 && repositoryPattern.MatchString(v) }
func ValidTag(v string) bool        { return tagPattern.MatchString(v) }
func Permissions() []string {
	return []string{"read", "manage", "artifact_pull", "artifact_push", "database_connect", "bucket_read", "bucket_write", "secret_resolve", "endpoint_invoke"}
}
func ValidPermission(v string) bool { return slices.Contains(Permissions(), v) }
func PermissionKind(permission string) string {
	switch permission {
	case "artifact_pull", "artifact_push":
		return "artifact_registry"
	case "database_connect":
		return "database"
	case "bucket_read", "bucket_write":
		return "bucket"
	case "secret_resolve":
		return "secret_reference"
	case "endpoint_invoke":
		return "endpoint"
	}
	return ""
}
func capacityValid(capacity, retention int64) bool {
	return capacity >= 1 && capacity <= MaxCapacity && retention >= 0 && retention <= 36500
}
func (s Spec) Valid(kind string) bool {
	n := 0
	for _, present := range []bool{s.Registry != nil, s.Storage != nil, s.Secret != nil, s.Binding != nil, s.Container != nil, s.Endpoint != nil} {
		if present {
			n++
		}
	}
	if kind == "grant" {
		return n == 0 && ValidID(s.SubjectID) && ValidPermission(s.Permission) && strings.HasPrefix(s.Target, "/v4/organizations/")
	}
	if s.SubjectID != "" || s.Permission != "" || s.Target != "" {
		return false
	}
	if !ServiceKind(kind) && kind != "bucket" {
		return n == 0
	}
	if n != 1 {
		return false
	}
	switch kind {
	case "artifact_registry":
		return s.Registry != nil && s.Registry.Format == "oci" && capacityValid(s.Registry.CapacityBytes, s.Registry.RetentionDays)
	case "database", "bucket":
		if s.Storage == nil || !capacityValid(s.Storage.CapacityBytes, s.Storage.RetentionDays) {
			return false
		}
		if kind == "bucket" {
			return s.Storage.Profile == "object"
		}
		return s.Storage.Profile == "shared" || s.Storage.Profile == "dedicated"
	case "secret_reference":
		return s.Secret != nil && ValidID(s.Secret.ExternalOwner) && ValidID(s.Secret.ExternalID) && ValidID(s.Secret.Version)
	case "service_binding":
		b := s.Binding
		return b != nil && ValidID(b.ConsumerID) && ValidID(b.TargetID) && ValidID(b.Slot) && PermissionKind(b.Intent) == b.TargetKind && slices.Contains([]string{"database", "bucket", "secret_reference", "artifact_registry"}, b.TargetKind)
	case "container_service":
		c := s.Container
		if c == nil || !ValidID(c.RegistryID) || !ValidRepository(c.Repository) || !ValidDigest(c.Digest) || !ValidID(c.ServiceAccountID) || c.CPU < 100 || c.CPU > 64000 || c.Memory < 64 || c.Memory > 262144 || c.Configuration == nil || c.SecretReferences == nil || len(c.Configuration) > 64 || len(c.SecretReferences) > 64 {
			return false
		}
		if c.RequiredBindings == nil || len(c.RequiredBindings) > 64 {
			return false
		}
		seen := map[string]bool{}
		for _, slot := range c.RequiredBindings {
			if !ValidID(slot) || seen[slot] {
				return false
			}
			seen[slot] = true
		}
		for k, v := range c.Configuration {
			if !ValidID(k) || len(v) > 4096 {
				return false
			}
		}
		for k, v := range c.SecretReferences {
			if !ValidID(k) || !ValidID(v) {
				return false
			}
			if _, ok := c.Configuration[k]; ok {
				return false
			}
		}
		return true
	case "endpoint":
		e := s.Endpoint
		return e != nil && ValidID(e.ServiceID) && e.Port >= 1 && e.Port <= 65535 && (e.Exposure == "private" || e.Exposure == "public") && e.TLS == "managed"
	}
	return false
}

// Required members must not silently decode missing/null values to Go zeroes.
func requireMembers(raw []byte, typ reflect.Type) error {
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || fields == nil {
		return errors.New("invalid required object")
	}
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Anonymous {
			if err := requireMembers(raw, field.Type); err != nil {
				return err
			}
			continue
		}
		parts := strings.Split(field.Tag.Get("json"), ",")
		name := parts[0]
		if name == "" {
			continue
		}
		value, ok := fields[name]
		optional := len(parts) > 1 && parts[1] == "omitempty"
		if !ok && optional {
			continue
		}
		if !ok || string(value) == "null" {
			return errors.New("missing or null required member")
		}
		t := field.Type
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		if t.Kind() == reflect.Struct {
			if err := requireMembers(value, t); err != nil {
				return err
			}
		}
	}
	return nil
}
func ValidateSpecWire(raw []byte) error { return requireMembers(raw, reflect.TypeOf(Spec{})) }
func validateSettingsWire(raw []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	typ := reflect.TypeOf(Spec{})
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.Type.Kind() != reflect.Pointer {
			continue
		}
		name := strings.Split(f.Tag.Get("json"), ",")[0]
		if v, ok := fields[name]; ok {
			if err := requireMembers(v, f.Type.Elem()); err != nil {
				return err
			}
		}
	}
	if v, ok := fields["status"]; ok {
		return requireMembers(v, reflect.TypeOf(ServiceStatus{}))
	}
	return nil
}

type Artifact struct {
	Scope
	RegistryID string `json:"registry_id"`
	Repository string `json:"repository"`
	Digest     string `json:"digest"`
	SizeBytes  int64  `json:"size_bytes"`
}

// Artifact lookup is exact and observational. A tag is resolved once per read.
func (c *Client) GetArtifact(ctx context.Context, registry, repository, digest, tag string) (Artifact, error) {
	var a Artifact
	if !c.scope.AllowsKind("artifact_registry") || !ValidID(registry) || !ValidRepository(repository) || ((digest == "") == (tag == "")) || (digest != "" && !ValidDigest(digest)) || (tag != "" && !ValidTag(tag)) {
		return a, errors.New("artifact requires exact registry/repository and one valid digest or tag")
	}
	q := url.Values{"repository": {repository}}
	if digest != "" {
		q.Set("digest", digest)
	} else {
		q.Set("tag", tag)
	}
	err := c.request(ctx, "GET", c.base()+"/artifact_registries/"+registry+"/artifact?"+q.Encode(), nil, &a)
	if err == nil && (a.Scope != c.scope || a.RegistryID != registry || a.Repository != repository || !ValidDigest(a.Digest) || a.SizeBytes < 0 || (digest != "" && a.Digest != digest)) {
		err = errors.New("invalid artifact identity in response")
	}
	return a, err
}

func validStatus(s ServiceStatus) bool {
	if !slices.Contains([]string{"", "oci", "object", "transit-sql", "https"}, s.Protocol) || (s.ResolvedVersion != "" && !ValidID(s.ResolvedVersion)) {
		return false
	}
	if !slices.Contains([]string{"pending", "ready", "failed"}, s.Readiness) || s.DesiredRevision < 1 || s.EffectiveRevision < 1 || s.ActiveRevision < 0 || s.ActiveRevision > s.EffectiveRevision || !ValidID(s.OperationID) {
		return false
	}
	if !slices.Contains([]string{"", "pending", "ready", "failed"}, s.TLSState) {
		return false
	}
	if !slices.Contains([]string{"", "artifact_missing", "pull_denied", "dependency_unavailable", "resolution_denied", "health_failed", "tls_failed"}, s.Reason) {
		return false
	}
	if s.Endpoint != "" {
		u, err := url.Parse(s.Endpoint)
		if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return false
		}
	}
	return true
}

func (a *Artifact) UnmarshalJSON(data []byte) error {
	if err := requireMembers(data, reflect.TypeOf(Artifact{})); err != nil {
		return err
	}
	type plain Artifact
	var value plain
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&value); err != nil {
		return errors.New("invalid artifact metadata")
	}
	*a = Artifact(value)
	return nil
}
