package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testHostileBoundaryAttackChain(t *testing.T, h integrationHarness) {
	t.Helper()

	sentinelDir := t.TempDir()
	sentinelPath := filepath.Join(sentinelDir, "host-sentinel")
	const sentinel = "host-secret-must-remain-unchanged"
	if err := os.WriteFile(sentinelPath, []byte(sentinel), 0o600); err != nil {
		t.Fatalf("failed to create host sentinel: %v", err)
	}
	before := mustFileSHA256(t, sentinelPath)
	t.Setenv("SANDBOX_ATTACK_CANARY", "host-secret-env-canary")

	code := fmt.Sprintf(`#define _GNU_SOURCE
#include <errno.h>
#include <fcntl.h>
#include <linux/capability.h>
#include <sched.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <sys/mount.h>
#include <sys/socket.h>
#include <sys/stat.h>
#include <sys/sysmacros.h>
#include <sys/syscall.h>
#include <unistd.h>

static void check_open(const char *label, const char *path) {
	int fd = open(path, O_RDONLY);
	printf("%%s:%%s\n", label, fd >= 0 ? "opened" : strerror(errno));
	if (fd >= 0) close(fd);
}

int main(void) {
	char linkpath[] = "/escape-link";
	unlink(linkpath);
	symlink(%q, linkpath);
	check_open("absolute-symlink", linkpath);
	check_open("traversal", "/workspace/../../%s");
	check_open("shadow", "/etc/shadow");
	check_open("passwd", "/etc/passwd");
	check_open("proc-root", "/proc/self/root%s");
	check_open("proc-fd", "/proc/self/fd/0");
	int fd = open(%q, O_WRONLY | O_APPEND);
	printf("sentinel-write:%%s\n", fd >= 0 ? "opened" : strerror(errno));
	if (fd >= 0) { write(fd, "tampered", 8); close(fd); }
	printf("mount:%%s\n", mount("/", "/escape-mount", NULL, MS_BIND, NULL) == 0 ? "succeeded" : strerror(errno));
	printf("unshare:%%s\n", unshare(CLONE_NEWNS) == 0 ? "succeeded" : strerror(errno));
	printf("mknod:%%s\n", mknod("/escape-device", S_IFCHR | 0600, makedev(1, 1)) == 0 ? "succeeded" : strerror(errno));
	FILE *status = fopen("/proc/self/status", "r");
	char line[256];
	while (status && fgets(line, sizeof(line), status)) {
		if (strncmp(line, "CapEff:", 7) == 0 || strncmp(line, "CapBnd:", 7) == 0) fputs(line, stdout);
	}
	if (status) fclose(status);
	const char *canary = getenv("SANDBOX_ATTACK_CANARY");
	printf("env:%%s\n", canary ? canary : "absent");
	int sock = socket(AF_INET, SOCK_STREAM, 0);
	printf("socket:%%s\n", sock >= 0 ? "created" : strerror(errno));
	if (sock >= 0) close(sock);
	return 0;
}
`, sentinelPath, sentinelPath, sentinelPath, sentinelPath)

	resp := callSimpleExecute(t, h.baseURL, buildCRequest(code, 4, 32768))
	combined := strings.ToLower(resp.Output + "\n" + resp.Error)
	for _, marker := range []string{
		"absolute-symlink:opened",
		"traversal:opened",
		"shadow:opened",
		"passwd:opened",
		"proc-root:opened",
		"sentinel-write:opened",
		"mount:succeeded",
		"unshare:succeeded",
		"mknod:succeeded",
		"env:host-secret-env-canary",
	} {
		if strings.Contains(combined, marker) {
			t.Fatalf("hostile attack chain crossed a sandbox boundary (%s); stdout=%q stderr=%q", marker, resp.Output, resp.Error)
		}
	}
	if strings.Contains(combined, "host-secret-env-canary") {
		t.Fatalf("hostile attack chain crossed a sandbox boundary; stdout=%q stderr=%q", resp.Output, resp.Error)
	}
	if strings.Contains(combined, "cap_eff:") && !strings.Contains(combined, "cap_eff:\t0000000000000000") {
		t.Fatalf("sandbox retained effective capabilities; stdout=%q stderr=%q", resp.Output, resp.Error)
	}
	if got := mustFileSHA256(t, sentinelPath); got != before {
		t.Fatalf("host sentinel changed after hostile workload: before=%s after=%s", before, got)
	}
	content, err := os.ReadFile(sentinelPath)
	if err != nil || string(content) != sentinel {
		t.Fatalf("host sentinel was removed or modified: content=%q err=%v", content, err)
	}
}
