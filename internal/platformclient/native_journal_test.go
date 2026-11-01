package platformclient

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spoke-sh/transit/sdks/go/serving"
)

func journalFixture(t *testing.T) Config {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	return Config{Endpoint: "https://management.test", Scope: Scope{OrganizationID: "1", ProjectID: "2", EnvironmentID: "3"}, Native: &NativeConfig{Executable: "/missing/native-client", Realm: 49, Principal: 65536, CredentialEpoch: 1, Ingress: 3, Service: 1, SeedFile: "/missing/credential", JournalDirectory: dir}}
}

func TestNativeCallerAdmissionJoinsAndCancelsQueuedWork(t *testing.T) {
	n := &nativeClient{calls: make(chan struct{}, 1)}
	entered, release, finished := make(chan struct{}), make(chan struct{}), make(chan struct{})
	go func() {
		defer close(finished)
		_, _ = nativeCall(context.Background(), n, func(*serving.Client) (int, error) {
			close(entered)
			<-release
			return 1, nil
		})
	}()
	<-entered
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := nativeCall(ctx, n, func(*serving.Client) (int, error) { t.Error("queued canceled work executed"); return 0, nil })
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("queued deadline lost: %v", err)
	}
	close(release)
	<-finished
	value, err := nativeCall(context.Background(), n, func(*serving.Client) (int, error) { return 2, nil })
	if err != nil || value != 2 {
		t.Fatalf("admission not released: %v", err)
	}
}

func TestNativeJournalInitializationNeverReplacesWork(t *testing.T) {
	cfg := journalFixture(t)
	if err := InitializeNativeJournal(cfg); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(cfg.Native.JournalDirectory, "dispatch.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = InitializeNativeJournal(cfg); err == nil {
		t.Fatal("existing journal replaced")
	}
	after, _ := os.ReadFile(path)
	if string(after) != string(before) {
		t.Fatal("journal changed")
	}
	if err = os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err = InitializeNativeJournal(cfg); err == nil {
		t.Fatal("lost journal silently recreated")
	}
	cfg = journalFixture(t)
	cfg.Native.InitialSequence = 1
	if err = InitializeNativeJournal(cfg); err == nil {
		t.Fatal("used principal initialized")
	}
}

func TestNativeJournalRefusesMissingCorruptAndLostTailBeforeDispatch(t *testing.T) {
	for _, kind := range []string{"missing", "corrupt", "sequence_gap", "wrong_scope"} {
		t.Run(kind, func(t *testing.T) {
			cfg := journalFixture(t)
			if kind != "missing" {
				if err := InitializeNativeJournal(cfg); err != nil {
					t.Fatal(err)
				}
				path := filepath.Join(cfg.Native.JournalDirectory, "dispatch.json")
				raw, _ := os.ReadFile(path)
				var j dispatchJournal
				if err := json.Unmarshal(raw, &j); err != nil {
					t.Fatal(err)
				}
				switch kind {
				case "corrupt":
					raw = []byte("{")
				case "sequence_gap":
					j.Sequence = 1
					raw, _ = json.Marshal(j)
				case "wrong_scope":
					j.Scope.Project = 4
					raw, _ = json.Marshal(j)
				}
				if err := os.WriteFile(path, raw, 0600); err != nil {
					t.Fatal(err)
				}
			}
			c, err := New(cfg)
			if err != nil {
				t.Fatal(err)
			}
			defer c.native.sdk.Close()
			if _, err = c.native.dispatch(context.Background(), [64]byte{}); err == nil {
				t.Fatal("invalid journal admitted dispatch")
			}
			// The executable and credential intentionally do not exist; refusal is
			// entirely local and cannot send or re-sign accepted work.
		})
	}
}
