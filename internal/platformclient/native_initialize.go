package platformclient

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// InitializeNativeJournal is an explicit operator action for a freshly allocated,
// unused principal. It never repairs/replaces an existing or lost journal.
// The operator must retain the bootstrap witness separately.
func InitializeNativeJournal(config Config) error {
	if config.Native == nil || config.Native.InitialSequence != 0 {
		return errors.New("journal initialization requires a fresh unused principal at sequence zero")
	}
	c, err := New(config)
	if err != nil {
		return err
	}
	defer c.native.sdk.Close()
	n := c.native
	dir := n.config.JournalDirectory
	info, err := os.Lstat(dir)
	if !filepath.IsAbs(dir) || err != nil || !info.IsDir() || info.Mode().Perm() != 0700 {
		return errors.New("precreate a private absolute 0700 journal directory")
	}
	// Check before creating the lock: a retained journal with a missing lock
	// may still have a writer holding the original unlinked inode.
	if _, err := os.Lstat(filepath.Join(dir, "dispatch.json")); !errors.Is(err, os.ErrNotExist) {
		return errors.New("native journal already exists or cannot be inspected; preserve original files")
	}
	fd, err := unix.Open(filepath.Join(dir, "dispatch.lock"), unix.O_CREAT|unix.O_EXCL|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0600)
	if err != nil {
		return errors.New("journal already initialized or initialization interrupted; preserve original files")
	}
	defer unix.Close(fd)
	// Keep this inode even on failure. Retrying initialization after partial
	// creation must fail and require operator reconciliation, never reset.
	if err = unix.Fsync(fd); err != nil {
		return errors.New("cannot persist journal lock")
	}
	j := dispatchJournal{Version: 3, Realm: n.config.Realm, Principal: n.config.Principal, Ingress: n.config.Ingress, Scope: n.scope, Endpoint: c.endpoint, Entries: []dispatchEntry{}}
	raw, err := json.Marshal(j)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(dir, "dispatch.json"), os.O_WRONLY|os.O_CREATE|os.O_EXCL|unix.O_NOFOLLOW, 0600)
	if err != nil {
		return errors.New("journal exists or cannot be initialized")
	}
	defer f.Close()
	if _, err = f.Write(raw); err != nil {
		return errors.New("cannot write initial journal")
	}
	if err = f.Sync(); err != nil {
		return errors.New("cannot persist initial journal")
	}
	d, err := os.Open(dir)
	if err != nil {
		return errors.New("cannot open journal directory")
	}
	defer d.Close()
	if err = d.Sync(); err != nil {
		return errors.New("cannot persist journal directory")
	}
	return nil
}
