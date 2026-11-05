//go:build linux

package provider

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spoke-sh/terraform-provider-spoke/internal/testutil/liveplatform"
)

// Steps are replayed from the same consumer root. Captures freeze public API
// identities and mutation inputs before dispatch; done steps retain their proof.
type consumerJournal struct {
	t      *testing.T
	path   string
	Done   map[string]bool            `json:"done"`
	Values map[string]json.RawMessage `json:"values"`
}

func openConsumerJournal(t *testing.T, dir string) *consumerJournal {
	t.Helper()
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal("cannot create private consumer root")
	}
	j := &consumerJournal{t: t, path: filepath.Join(dir, "journal.json"), Done: map[string]bool{}, Values: map[string]json.RawMessage{}}
	if data, err := os.ReadFile(j.path); err == nil {
		if json.Unmarshal(data, j) != nil || j.Done == nil || j.Values == nil {
			t.Fatal("invalid recovery journal; retain original state")
		}
	} else if !os.IsNotExist(err) {
		t.Fatal("cannot read recovery journal")
	}
	j.save()
	return j
}

func (j *consumerJournal) save() {
	j.t.Helper()
	if err := liveplatform.WritePrivateJSON(j.path, j); err != nil {
		j.t.Fatal(err)
	}
}

func (j *consumerJournal) step(name string, f func()) {
	j.t.Helper()
	if j.Done[name] {
		return
	}
	f()
	j.Done[name] = true
	j.save()
}

func captureConsumer[T any](j *consumerJournal, name string, f func() T) T {
	j.t.Helper()
	if data, ok := j.Values[name]; ok {
		var v T
		if json.Unmarshal(data, &v) != nil {
			j.t.Fatal("invalid retained consumer value")
		}
		return v
	}
	v := f()
	data, err := json.Marshal(v)
	if err != nil {
		j.t.Fatal(err)
	}
	j.Values[name] = data
	j.save()
	return v
}
