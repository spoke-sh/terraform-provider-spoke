package platformclient

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestGovernorContractVectors(t *testing.T) {
	p := CurrentGovernorContract()
	if p.Status != "provisional" {
		t.Fatal("governor must remain provisional")
	}
	var fixture struct {
		Vectors []struct {
			Request     GovernorRequest   `json:"request"`
			Fingerprint string            `json:"fingerprint"`
			Prepared    GovernorOperation `json:"prepared"`
		} `json:"vectors"`
		Responses map[string]json.RawMessage `json:"responses"`
	}
	if err := json.Unmarshal(governorVectorsJSON, &fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.Vectors) != 4 {
		t.Fatal("missing required canonical vectors")
	}
	seen := map[string]bool{}
	for _, v := range fixture.Vectors {
		got, err := GovernorFingerprint(v.Prepared.Scope, v.Prepared.PrincipalID, v.Request)
		if err != nil || got != v.Fingerprint || seen[got] {
			t.Fatal("independent commitment mismatch", err)
		}
		seen[got] = true
		if !governorValue("Operation", v.Prepared) {
			t.Fatal("invalid operation vector")
		}
	}
	for kind, raw := range fixture.Responses {
		var out any
		switch kind {
		case "Capabilities":
			out = new(GovernorCapabilities)
		case "Governor":
			out = new(Governor)
		case "GovernorPage":
			out = new(GovernorPage)
		case "PendingPage":
			out = new(GovernorPendingPage)
		default:
			t.Fatal("unknown response vector")
		}
		if err := decodeGovernor(raw, kind, out); err != nil {
			t.Fatal(kind, err)
		}
		encoded, _ := json.Marshal(out)
		var a, b any
		_ = json.Unmarshal(raw, &a)
		_ = json.Unmarshal(encoded, &b)
		if !reflect.DeepEqual(a, b) {
			t.Fatal("typed projection loses contract fields", kind)
		}
	}
}

func TestGovernorRejectsAmbiguousJSON(t *testing.T) {
	for _, raw := range []string{
		`{"fingerprint":"` + strings.Repeat("a", 64) + `","fingerprint":"` + strings.Repeat("b", 64) + `"}`,
		`{"fingerprint":null}`, `{"fingerprint":"` + strings.Repeat("a", 64) + `","secret":"secret-canary"}`,
		`{} {}`, strings.Repeat("[", 26) + "0" + strings.Repeat("]", 26), strings.Repeat(" ", governorResponseLimit+1),
	} {
		err := decodeGovernor([]byte(raw), "Commit", nil)
		if err == nil || strings.Contains(err.Error(), "secret-canary") {
			t.Fatal("invalid response accepted or disclosed")
		}
	}
}
