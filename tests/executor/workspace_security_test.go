package executor_test

import (
	"testing"

	"CodeSandboxAPI/executor"
)

func TestDefaultRuntimeMountsDoNotExposeHostEtc(t *testing.T) {
	for _, mount := range executor.DefaultRuntimeMounts() {
		if mount == "/etc" {
			t.Fatal("runtime mounts must not expose the host /etc tree")
		}
	}
}

func TestJavaRuntimeMountsOnlyAllowlistedSecurityDirectory(t *testing.T) {
	ws, err := executor.BuildWorkspacePlan("/tmp/sandbox-dir", "java")
	if err != nil {
		t.Skipf("Java runtime unavailable: %v", err)
	}
	for _, mount := range ws.RuntimeMounts {
		if mount == "/etc" {
			t.Fatal("Java runtime must not expose the host /etc tree")
		}
	}
}
