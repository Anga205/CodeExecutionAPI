package executor_test

import (
	"strings"
	"testing"

	"CodeSandboxAPI/executor"
)

func TestSandboxInitSourceMountsProcfsBeforeSeccomp(t *testing.T) {
	source := executor.SandboxInitSourceForTest()
	if !strings.Contains(source, "mount(\"proc\", \"/proc\", \"proc\"") {
		t.Fatal("sandbox init source must attempt to mount procfs before installing seccomp")
	}
	if !strings.Contains(source, "#include <sys/mount.h>") {
		t.Fatal("sandbox init source must include sys/mount.h for procfs mounting")
	}
	if strings.Contains(source, "mount_procfs_if_possible") {
		t.Fatal("procfs setup must not fail open")
	}
	if !strings.Contains(source, "drop_capabilities") || !strings.Contains(source, "SYS_capset") {
		t.Fatal("sandbox init must clear capabilities before executing user code")
	}
	if !strings.Contains(source, "__NR_bpf") || !strings.Contains(source, "__NR_unshare") {
		t.Fatal("seccomp policy must deny namespace and kernel attack syscalls")
	}
}
