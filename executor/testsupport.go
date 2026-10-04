package executor

import (
	"os/exec"
)

// This file exposes a narrow, read-mostly surface of the executor internals so
// the black-box integration tests in the tests package can assert on the
// sandbox launch plan without duplicating production logic. Nothing here is
// used by the production code paths; the wrappers simply forward to the
// unexported implementations.

// BuildSandboxCommand returns the sandboxed command with cgroup fd attachment.
func BuildSandboxCommand(stdin string, ws SandboxWorkspace, cgroupFD int) *exec.Cmd {
	return buildSandboxCommand(stdin, ws, cgroupFD)
}

// BuildSandboxCommandWithoutCgroupFD returns the command built without cgroup
// fd attachment. It exists so tests can prove the launch path never uses it.
func BuildSandboxCommandWithoutCgroupFD(stdin string, ws SandboxWorkspace) *exec.Cmd {
	return buildSandboxCommandWithoutCgroupFD(stdin, ws)
}

// AppendExitDiagnostic forwards to the stderr diagnostic helper.
func AppendExitDiagnostic(stderr string, err error) string {
	return appendExitDiagnostic(stderr, err)
}

// BindMountReadOnly forwards to the read-only bind mount helper.
func BindMountReadOnly(source, target string) error {
	return bindMountReadOnly(source, target)
}

// PrepareRuntimeMounts forwards to the runtime mount preparation helper.
func PrepareRuntimeMounts(ws *SandboxWorkspace) error {
	return prepareRuntimeMounts(ws)
}

// DefaultRuntimeMounts returns the runtime mount allowlist.
func DefaultRuntimeMounts() []string {
	return defaultRuntimeMounts()
}

// BuildWorkspacePlan returns the launch plan for a language without touching
// the filesystem.
func BuildWorkspacePlan(dir, language string) (SandboxWorkspace, error) {
	return buildWorkspacePlan(dir, language)
}

// SandboxInitSourceForTest returns the embedded sandbox init C source.
func SandboxInitSourceForTest() string {
	return sandboxInitSource
}

// SetMountFuncsForTest replaces the mount syscall hooks and returns a restore
// function. Tests that use it must not run in parallel with each other.
func SetMountFuncsForTest(mount func(source, target, fstype string, flags uintptr, data string) error, unmount func(target string, flags int) error) func() {
	originalMount := syscallMount
	originalUnmount := syscallUnmount
	syscallMount = mount
	syscallUnmount = unmount
	return func() {
		syscallMount = originalMount
		syscallUnmount = originalUnmount
	}
}
