package executor_test

import (
	"errors"
	"syscall"
	"testing"

	"CodeSandboxAPI/executor"
)

// These tests replace the executor's mount syscall hooks, so they must not run
// in parallel with each other.

func TestBindMountReadOnlyUsesRemountReadOnly(t *testing.T) {
	type mountCall struct {
		source string
		target string
		fstype string
		flags  uintptr
		data   string
	}
	calls := make([]mountCall, 0, 2)

	restore := executor.SetMountFuncsForTest(
		func(source, target, fstype string, flags uintptr, data string) error {
			calls = append(calls, mountCall{source: source, target: target, fstype: fstype, flags: flags, data: data})
			return nil
		},
		nil,
	)
	t.Cleanup(restore)

	if err := executor.BindMountReadOnly("/src", "/dst"); err != nil {
		t.Fatalf("BindMountReadOnly returned error: %v", err)
	}

	if len(calls) != 2 {
		t.Fatalf("unexpected mount call count: got=%d want=2", len(calls))
	}

	if calls[0].source != "/src" || calls[0].target != "/dst" {
		t.Fatalf("unexpected first mount call: %+v", calls[0])
	}
	wantBindFlags := uintptr(syscall.MS_BIND | syscall.MS_REC)
	if calls[0].flags != wantBindFlags {
		t.Fatalf("unexpected first mount flags: got=%#x want=%#x", calls[0].flags, wantBindFlags)
	}

	if calls[1].source != "" || calls[1].target != "/dst" {
		t.Fatalf("unexpected second mount call: %+v", calls[1])
	}
	wantRemountFlags := uintptr(syscall.MS_BIND | syscall.MS_REMOUNT | syscall.MS_RDONLY | syscall.MS_NOSUID | syscall.MS_NODEV | syscall.MS_REC)
	if calls[1].flags != wantRemountFlags {
		t.Fatalf("unexpected second mount flags: got=%#x want=%#x", calls[1].flags, wantRemountFlags)
	}
}

func TestBindMountReadOnlyReturnsFirstError(t *testing.T) {
	wantErr := errors.New("bind failed")
	restore := executor.SetMountFuncsForTest(
		func(source, target, fstype string, flags uintptr, data string) error {
			return wantErr
		},
		nil,
	)
	t.Cleanup(restore)

	err := executor.BindMountReadOnly("/src", "/dst")
	if !errors.Is(err, wantErr) {
		t.Fatalf("unexpected error: got=%v want=%v", err, wantErr)
	}
}

func TestBindMountReadOnlyRollsBackAfterRemountFailure(t *testing.T) {
	remountErr := errors.New("remount failed")
	var unmounted string
	mountCalls := 0
	restore := executor.SetMountFuncsForTest(
		func(source, target, fstype string, flags uintptr, data string) error {
			mountCalls++
			if mountCalls == 2 {
				return remountErr
			}
			return nil
		},
		func(target string, flags int) error {
			unmounted = target
			return nil
		},
	)
	t.Cleanup(restore)

	err := executor.BindMountReadOnly("/src", "/dst")
	if !errors.Is(err, remountErr) {
		t.Fatalf("unexpected error: got=%v want=%v", err, remountErr)
	}
	if unmounted != "/dst" {
		t.Fatalf("failed remount left mount active: unmounted=%q", unmounted)
	}
}

func TestPrepareRuntimeMountsMarksRootPrivateFirst(t *testing.T) {
	tempDir := t.TempDir()
	source := t.TempDir()

	type mountCall struct {
		source string
		target string
		flags  uintptr
	}
	calls := make([]mountCall, 0, 3)

	restore := executor.SetMountFuncsForTest(
		func(source, target, fstype string, flags uintptr, data string) error {
			calls = append(calls, mountCall{source: source, target: target, flags: flags})
			return nil
		},
		nil,
	)
	t.Cleanup(restore)

	ws := executor.SandboxWorkspace{
		Dir:           tempDir,
		UseChroot:     true,
		RuntimeMounts: []string{source},
	}

	if err := executor.PrepareRuntimeMounts(&ws); err != nil {
		t.Fatalf("PrepareRuntimeMounts returned error: %v", err)
	}
	if len(calls) < 1 {
		t.Fatal("expected at least one mount call")
	}

	if calls[0].source != "" || calls[0].target != "/" {
		t.Fatalf("first mount must mark root private: %+v", calls[0])
	}
	wantFlags := uintptr(syscall.MS_REC | syscall.MS_PRIVATE)
	if calls[0].flags != wantFlags {
		t.Fatalf("unexpected private-root mount flags: got=%#x want=%#x", calls[0].flags, wantFlags)
	}
}
