package liveplatform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestMetadataProcessHelper(t *testing.T) {
	mode := os.Getenv("SPOKE_METADATA_PROCESS_HELPER")
	if mode == "" {
		return
	}
	if mode == "parent" {
		child := exec.Command(os.Args[0], "-test.run=^TestMetadataProcessHelper$")
		child.Env = append(os.Environ(), "SPOKE_METADATA_PROCESS_HELPER=child")
		child.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		if err := child.Start(); err != nil {
			os.Exit(2)
		}
		if err := os.WriteFile(os.Getenv("SPOKE_METADATA_CHILD_PID"), []byte(strconv.Itoa(child.Process.Pid)), 0600); err != nil {
			os.Exit(3)
		}
		_ = child.Wait()
	} else {
		time.Sleep(time.Minute)
	}
	os.Exit(0)
}

func TestMetadataDeadlineStopsIndependentConsumerGroups(t *testing.T) {
	file := filepath.Join(t.TempDir(), "child.pid")
	parent := exec.Command(os.Args[0], "-test.run=^TestMetadataProcessHelper$")
	parent.Env = append(os.Environ(), "SPOKE_METADATA_PROCESS_HELPER=parent", "SPOKE_METADATA_CHILD_PID="+file)
	parent.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := parent.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = stopFixtureTree(parent.Process.Pid); _ = parent.Wait() }()
	var child int
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		data, _ := os.ReadFile(file)
		child, _ = strconv.Atoi(string(data))
		if child > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if child == 0 {
		t.Fatal("independent consumer did not start")
	}
	if err := stopFixtureTree(parent.Process.Pid); err != nil {
		t.Fatal(err)
	}
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", child))
		// A zombie is already dead; it cannot continue a mutation while its killed
		// parent is being reaped. Never signal unrelated process groups.
		if os.IsNotExist(err) || (err == nil && strings.HasPrefix(strings.SplitN(string(data), ") ", 2)[1], "Z ")) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("independent consumer survived fixture deadline")
}
