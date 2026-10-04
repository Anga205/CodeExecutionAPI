package tests

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func mustFileSHA256(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}

	sum := sha256.Sum256(content)
	return fmt.Sprintf("%x", sum)
}

func mustFreeBytes(t *testing.T, path string) uint64 {
	t.Helper()
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		t.Fatalf("statfs failed for %s: %v", path, err)
	}
	return stat.Bavail * uint64(stat.Bsize)
}

func countSandboxTempDirs(t *testing.T) int {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(os.TempDir(), "codesandbox-c-*"))
	if err != nil {
		t.Fatalf("failed to glob sandbox temp dirs: %v", err)
	}
	return len(matches)
}

func countProcessesByComm(t *testing.T, expected string) int {
	t.Helper()
	entries, err := os.ReadDir("/proc")
	if err != nil {
		t.Fatalf("failed to read /proc: %v", err)
	}

	count := 0
	for _, entry := range entries {
		if !entry.IsDir() || !isNumeric(entry.Name()) {
			continue
		}
		if hasProcessComm(entry, expected) {
			count++
		}
	}
	return count
}

func hasProcessComm(entry fs.DirEntry, expected string) bool {
	commPath := filepath.Join("/proc", entry.Name(), "comm")
	content, err := os.ReadFile(commPath)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(content)) == expected
}

func isNumeric(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return value != ""
}

func assertDurationWindow(t *testing.T, durationRaw string, min, max time.Duration, label string) {
	t.Helper()
	d, err := time.ParseDuration(durationRaw)
	if err != nil {
		t.Fatalf("failed to parse cpu_time for %s (%q): %v", label, durationRaw, err)
	}
	if d < min || d > max {
		t.Fatalf("%s duration out of range: got=%s min=%s max=%s", label, d, min, max)
	}
}

func assertDurationNotExcessive(t *testing.T, durationRaw string, max time.Duration, label string) {
	t.Helper()
	d, err := time.ParseDuration(durationRaw)
	if err != nil {
		t.Fatalf("failed to parse cpu_time for %s (%q): %v", label, durationRaw, err)
	}
	if d > max {
		t.Fatalf("%s took too long: got=%s max=%s", label, d, max)
	}
}

func mustCreateAndDeleteTempFile(t *testing.T) {
	t.Helper()
	f, err := os.CreateTemp("", "inode-health-*")
	if err != nil {
		t.Fatalf("host temp filesystem unhealthy after inode test: %v", err)
	}
	name := f.Name()
	if err := f.Close(); err != nil {
		t.Fatalf("failed to close temp file %s: %v", name, err)
	}
	if err := os.Remove(name); err != nil {
		t.Fatalf("failed to remove temp file %s: %v", name, err)
	}
}

func containsAny(haystack string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(haystack, strings.ToLower(needle)) {
			return true
		}
	}
	return false
}

func assertDiskReclaimed(t *testing.T, beforeFree, afterFree uint64) {
	t.Helper()
	const maxResidual = uint64(20 * 1024 * 1024)
	if beforeFree > afterFree && (beforeFree-afterFree) > maxResidual {
		t.Fatalf("sandbox disk garbage appears to persist: free space dropped by %d bytes", beforeFree-afterFree)
	}
}

func assertNoSandboxLeak(t *testing.T, beforeCount, afterCount int) {
	t.Helper()
	if afterCount > beforeCount {
		t.Fatalf("sandbox temp directories leaked: before=%d after=%d", beforeCount, afterCount)
	}
}
