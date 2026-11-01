package platformclient

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spoke-sh/transit/sdks/go/serving"
	"golang.org/x/sys/unix"
)

type nativeServingRead struct {
	Realm     uint64 `json:"realm"`
	Principal uint64 `json:"principal"`
	Service   uint64 `json:"service"`
	Ingress   uint64 `json:"ingress"`
	Kind      string `json:"kind"`
	Operation uint64 `json:"operation"`
}
type nativeServingResponse struct {
	Service        uint64 `json:"service"`
	Resource       uint64 `json:"resource"`
	Revision       uint64 `json:"revision"`
	PolicyRevision uint64 `json:"policy_revision"`
	Stage          string `json:"stage"`
	Binding        *struct {
		Revision     uint64 `json:"revision"`
		Target       uint64 `json:"target"`
		Codec        uint16 `json:"codec"`
		ActiveLimit  uint8  `json:"active_limit"`
		WaitBeforeMS uint64 `json:"wait_before_ms"`
		Published    bool   `json:"published"`
	} `json:"binding"`
	Operation *struct {
		Principal      uint64   `json:"principal"`
		Operation      uint64   `json:"operation"`
		Binding        uint64   `json:"binding"`
		Codec          uint16   `json:"codec"`
		Deadline       uint64   `json:"deadline_ms"`
		Digest         [32]byte `json:"digest"`
		Target         uint64   `json:"target"`
		Correlation    uint64   `json:"correlation"`
		PolicyRevision uint64   `json:"policy_revision"`
		Phase          string   `json:"phase"`
		Resolution     uint64   `json:"resolution_input"`
		Result         string   `json:"result"`
		ReplyBytes     uint16   `json:"reply_bytes"`
	} `json:"operation"`
	Diagnostic   json.RawMessage `json:"diagnostic"`
	Disposition  json.RawMessage `json:"disposition"`
	Capabilities json.RawMessage `json:"capabilities"`
}

func (n *nativeClient) serving(ctx context.Context, kind string, operation uint64) (nativeServingResponse, error) {
	response, e := n.sdk.Observe(ctx, kind, operation)
	var out nativeServingResponse
	if e == nil {
		e = nativeResponseBody(response, &out)
	}
	if e == nil && (out.Service != n.config.Service || out.Resource == 0 || out.Revision == 0 || out.PolicyRevision == 0) {
		e = errors.New("native Serving target mismatch")
	}
	if e == nil && kind == "status" && (out.Operation == nil || out.Operation.Principal != n.config.Principal || out.Operation.Operation != operation || out.Operation.Target != n.config.Ingress || out.Operation.Codec != 2) {
		e = errors.New("native Serving operation mismatch")
	}
	return out, e
}

type dispatchEntry struct {
	Intent       [64]byte `json:"intent"`
	Saved        []byte   `json:"saved"`
	Sequence     uint64   `json:"sequence"`
	Acknowledged bool     `json:"acknowledged"`
}
type dispatchJournal struct {
	Version   uint8           `json:"version"`
	Realm     uint64          `json:"realm"`
	Principal uint64          `json:"principal"`
	Ingress   uint64          `json:"ingress"`
	Scope     nativeScope     `json:"scope"`
	Endpoint  string          `json:"endpoint"`
	Sequence  uint64          `json:"sequence"`
	Entries   []dispatchEntry `json:"entries"`
}

const journalLimit = 1 << 20

// One OS lock inode protects the entire durable read/freeze/send/ack cycle across
// aliases and separate provider processes on one host. The inode is never renamed
// or deleted. Native sequence admission additionally fences any competing writer.
// This profile does not claim distributed/NFS locking or automatic journal loss recovery.
func (n *nativeClient) dispatch(ctx context.Context, intent [64]byte) (uint64, error) {
	dir := n.config.JournalDirectory
	if !filepath.IsAbs(dir) {
		return 0, errors.New("native journal directory must be absolute")
	}
	info, e := os.Lstat(dir)
	if e != nil || !info.IsDir() || info.Mode().Perm() != 0700 {
		return 0, errors.New("precreate a private 0700 native journal directory")
	}
	fd, e := unix.Open(filepath.Join(dir, "dispatch.lock"), unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0600)
	if e != nil {
		return 0, errors.New("cannot open native dispatch lock")
	}
	defer unix.Close(fd)
	for {
		e = unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB)
		if e == nil {
			break
		}
		if e != unix.EWOULDBLOCK && e != unix.EAGAIN {
			return 0, errors.New("cannot lock native dispatch journal")
		}
		if e = n.client.wait(ctx, 20*time.Millisecond); e != nil {
			return 0, e
		}
	}
	defer unix.Flock(fd, unix.LOCK_UN)
	path := filepath.Join(dir, "dispatch.json")
	j := dispatchJournal{Version: 3, Realm: n.config.Realm, Principal: n.config.Principal, Ingress: n.config.Ingress, Scope: n.scope, Endpoint: n.client.endpoint, Sequence: n.config.InitialSequence, Entries: []dispatchEntry{}}
	f, e := os.OpenFile(path, os.O_RDONLY|unix.O_NOFOLLOW, 0)
	if e == nil {
		defer f.Close()
		raw, e := io.ReadAll(io.LimitReader(f, journalLimit+1))
		if e != nil || len(raw) > journalLimit {
			return 0, errors.New("native journal unreadable or full")
		}
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.DisallowUnknownFields()
		if dec.Decode(&j) != nil || dec.Decode(new(any)) != io.EOF {
			return 0, errors.New("invalid native dispatch journal")
		}
	} else {
		return 0, errors.New("native journal missing or unreadable; initialize explicitly before first dispatch")
	}
	if j.Version != 3 || j.Realm != n.config.Realm || j.Principal != n.config.Principal || j.Ingress != n.config.Ingress || j.Scope != n.scope || j.Endpoint != n.client.endpoint || j.Sequence < n.config.InitialSequence || j.Sequence >= nativeMax || len(j.Entries) > 256 || j.Sequence != n.config.InitialSequence+uint64(len(j.Entries)) {
		return 0, errors.New("native journal target/sequence mismatch")
	}
	// Validate the complete journal before replaying any retained operation.
	pendingEntries := make([]*serving.Pending, len(j.Entries))
	for i := range j.Entries {
		entry := &j.Entries[i]
		pending, metadata, inspectErr := n.sdk.Recover(ctx, entry.Saved)
		if inspectErr != nil || metadata.Sequence != entry.Sequence || entry.Sequence != n.config.InitialSequence+uint64(i)+1 || metadata.Command.Operation != entry.Sequence || metadata.Command.Codec != 2 || metadata.Command.Payload != hex.EncodeToString(entry.Intent[:]) {
			return 0, errors.New("invalid retained dispatch identity")
		}
		pendingEntries[i] = pending
	}
	// Reconcile every frozen unacknowledged send before allocating or returning.
	for i := range j.Entries {
		entry := &j.Entries[i]
		if !entry.Acknowledged {
			// Resubmit only the exact SDK-exported operation. Native admission
			// deduplicates it; observing a later operation never replaces it.
			if e = n.sendFrozen(ctx, pendingEntries[i]); e != nil {
				return 0, e
			}
			entry.Acknowledged = true
			if e = saveDispatch(path, j); e != nil {
				return 0, e
			}
		}

	}
	for _, entry := range j.Entries {
		if entry.Intent == intent {
			return entry.Sequence, nil
		}
	}
	if len(j.Entries) >= 256 {
		return 0, errors.New("native journal capacity reached; retain original work")
	}
	discovery, e := n.serving(ctx, "discovery", 0)
	if e != nil {
		return 0, e
	}
	binding := discovery.Binding
	if binding == nil || !binding.Published || binding.Codec != 2 || binding.Target != n.config.Ingress || binding.WaitBeforeMS <= uint64(time.Now().Add(time.Second).UnixMilli()) {
		return 0, errors.New("native binding unavailable or expired")
	}
	j.Sequence++
	pending, e := n.sdk.Prepare(ctx, j.Sequence, serving.Submit{Operation: j.Sequence, Binding: binding.Revision, Codec: 2, DeadlineMS: binding.WaitBeforeMS, Payload: intent[:]})
	if e != nil {
		return 0, e
	}
	j.Entries = append(j.Entries, dispatchEntry{Intent: intent, Saved: pending.Export(), Sequence: j.Sequence})

	if e = saveDispatch(path, j); e != nil {
		return 0, e
	} // fsync exact identity BEFORE dispatch.
	if e = n.sendFrozen(ctx, pending); e != nil {
		return 0, e
	}
	j.Entries[len(j.Entries)-1].Acknowledged = true
	if e = saveDispatch(path, j); e != nil {
		return 0, e
	}
	return j.Sequence, nil
}
func (n *nativeClient) sendFrozen(ctx context.Context, pending *serving.Pending) error {
	response, e := n.sdk.Dispatch(ctx, pending)
	if e != nil {
		return e
	}
	var result struct {
		Stage   string `json:"stage"`
		Receipt struct {
			Position uint64 `json:"position"`
		} `json:"receipt"`
	}
	if e = nativeResponseBody(response, &result); e != nil {
		return e
	}
	if result.Stage != "ingress_accepted" || result.Receipt.Position == 0 {
		return errors.New("invalid native ingress acknowledgement")
	}
	return nil
}
func saveDispatch(path string, j dispatchJournal) error {
	raw, e := json.Marshal(j)
	if e != nil || len(raw) > journalLimit {
		return errors.New("native journal capacity exceeded; retain it for recovery")
	}
	f, e := os.CreateTemp(filepath.Dir(path), ".dispatch-*")
	if e != nil {
		return errors.New("cannot prepare native journal")
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if _, e = f.Write(raw); e != nil {
		return errors.New("cannot write native journal")
	}
	if e = f.Sync(); e != nil {
		return errors.New("cannot persist native journal")
	}
	if e = f.Close(); e != nil {
		return errors.New("cannot close native journal")
	}
	if e = os.Rename(f.Name(), path); e != nil {
		return errors.New("cannot publish native journal")
	}
	d, e := os.Open(filepath.Dir(path))
	if e != nil {
		return errors.New("cannot open native journal directory")
	}
	defer d.Close()
	if e = d.Sync(); e != nil {
		return errors.New("cannot persist native journal directory")
	}
	return nil
}
