package tests

import (
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

// isolationScenario is one language-agnostic isolation check. Each scenario is
// executed once per supported language by runIsolationSuite.
type isolationScenario struct {
	name string
	run  func(t *testing.T, h integrationHarness, lang language)
}

// isolationScenarios is the single source of truth for the isolation suite.
// Adding a language only requires adding a language descriptor, not a new set
// of test functions.
var isolationScenarios = []isolationScenario{
	{name: "file privacy across request IDs", run: scenarioFilesystemIsolation},
	{name: "disk spammer is terminated and data is reclaimed", run: scenarioDiskCleanup},
	{name: "fork bomb does not poison subsequent requests", run: scenarioForkBombContainment},
	{name: "network namespace blocks localhost bridge", run: scenarioNetworkIsolation},
	{name: "memory hard limit triggers oom kill", run: scenarioMemoryHardLimit},
	{name: "io flood is bounded and returns before timeout", run: scenarioIOFloodResilience},
	{name: "signal trap cannot survive forced timeout", run: scenarioSignalTrapTimeout},
	{name: "orphan grandchild is reaped after request exits", run: scenarioOrphanReaping},
	{name: "inode bomb does not poison host temp filesystem", run: scenarioInodeExhaustion},
	{name: "privileged reboot syscall is denied", run: scenarioPrivilegedSyscallDenied},
}

func scenarioFilesystemIsolation(t *testing.T, h integrationHarness, lang language) {
	writeCode := lang.load(t, "file_privacy_write_read", nil)
	writeResp := callSimpleExecute(t, h.baseURL, buildLanguageRequest(lang.Name, writeCode, 4, 32768))
	if !strings.Contains(writeResp.Output, "SecretData123") {
		t.Fatalf("step A did not return secret in stdout; stdout=%q stderr=%q", writeResp.Output, writeResp.Error)
	}

	readCode := lang.load(t, "file_privacy_read_only", nil)
	readResp := callSimpleExecute(t, h.baseURL, buildLanguageRequest(lang.Name, readCode, 4, 32768))
	if !strings.Contains(strings.ToLower(readResp.Error), "no such file or directory") {
		t.Fatalf("step B unexpectedly accessed file from another sandbox; stdout=%q stderr=%q", readResp.Output, readResp.Error)
	}
}

func scenarioDiskCleanup(t *testing.T, h integrationHarness, lang language) {
	beforeFree := mustFreeBytes(t, os.TempDir())
	beforeCount := countSandboxTempDirs(t)

	code := lang.load(t, "disk_spammer", nil)
	resp := callSimpleExecute(t, h.baseURL, buildLanguageRequest(lang.Name, code, 2, 32768))
	stderrLower := strings.ToLower(resp.Error)
	if !containsAny(stderrLower, []string{"execution timed out", "memory limit exceeded"}) {
		t.Fatalf("disk spammer was not terminated as expected; stdout=%q stderr=%q", resp.Output, resp.Error)
	}

	time.Sleep(400 * time.Millisecond)
	assertDiskReclaimed(t, beforeFree, mustFreeBytes(t, os.TempDir()))
	assertNoSandboxLeak(t, beforeCount, countSandboxTempDirs(t))
}

func scenarioForkBombContainment(t *testing.T, h integrationHarness, lang language) {
	bombCode := lang.load(t, "fork_bomb", nil)
	_ = callSimpleExecute(t, h.baseURL, buildLanguageRequest(lang.Name, bombCode, 2, 32768))

	helloCode := lang.load(t, "hello_world", nil)
	helloResp := callSimpleExecute(t, h.baseURL, buildLanguageRequest(lang.Name, helloCode, 4, 32768))
	if !strings.Contains(helloResp.Output, "Hello World") {
		t.Fatalf("follow-up request failed after fork bomb; stdout=%q stderr=%q", helloResp.Output, helloResp.Error)
	}
	if strings.Contains(strings.ToLower(helloResp.Error), "resource temporarily unavailable") {
		t.Fatalf("follow-up request indicates host PID exhaustion; stderr=%q", helloResp.Error)
	}
}

func scenarioNetworkIsolation(t *testing.T, h integrationHarness, lang language) {
	replacements := map[string]string{"__API_PORT__": strconv.Itoa(h.apiPort)}
	code := lang.load(t, "network_localhost_bridge", replacements)
	resp := callSimpleExecute(t, h.baseURL, buildLanguageRequest(lang.Name, code, 3, 32768))

	combined := strings.ToLower(resp.Output + "\n" + resp.Error)
	if strings.Contains(combined, "connected") {
		t.Fatalf("localhost bridge unexpectedly succeeded; stdout=%q stderr=%q", resp.Output, resp.Error)
	}
	if !containsAny(combined, []string{"connection refused", "network is unreachable", "no route to host"}) {
		t.Fatalf("network isolation did not produce expected connect error; stdout=%q stderr=%q", resp.Output, resp.Error)
	}
}

func scenarioMemoryHardLimit(t *testing.T, h integrationHarness, lang language) {
	code := lang.load(t, "memory_bomb", nil)
	resp := callSimpleExecute(t, h.baseURL, buildLanguageRequest(lang.Name, code, 3, 16384))
	if !containsAny(strings.ToLower(resp.Error), []string{"memory limit exceeded", "killed", "execution timed out"}) {
		t.Fatalf("memory bomb did not terminate as expected; stdout=%q stderr=%q", resp.Output, resp.Error)
	}

	execDur, err := time.ParseDuration(resp.CPUTime)
	if err != nil {
		t.Fatalf("failed to parse cpu_time %q: %v", resp.CPUTime, err)
	}
	if execDur > 500*time.Millisecond {
		t.Fatalf("memory enforcement was too slow: cpu_time=%s stderr=%q", resp.CPUTime, resp.Error)
	}
}

func scenarioIOFloodResilience(t *testing.T, h integrationHarness, lang language) {
	code := lang.load(t, "io_spam", nil)
	resp := callSimpleExecute(t, h.baseURL, buildLanguageRequest(lang.Name, code, 1, 32768))

	stderrLower := strings.ToLower(resp.Error)
	if !containsAny(stderrLower, []string{"execution timed out", "killed", "memory limit exceeded"}) {
		t.Fatalf("io flood did not terminate with expected error; stdout=%q stderr=%q", resp.Output, resp.Error)
	}

	if strings.TrimSpace(resp.Error) == "" {
		t.Fatalf("io flood returned empty stderr; expected bounded but non-empty error output")
	}

	const maxErrorBytes = (1 << 20) + 4096
	if len(resp.Error) > maxErrorBytes {
		t.Fatalf("stderr exceeded resilience cap: got=%d bytes cap=%d", len(resp.Error), maxErrorBytes)
	}

	assertDurationNotExcessive(t, resp.CPUTime, maxIOFloodResponseDuration, "io flood request")
}

func scenarioSignalTrapTimeout(t *testing.T, h integrationHarness, lang language) {
	code := lang.load(t, "signal_trap", nil)
	resp := callSimpleExecute(t, h.baseURL, buildLanguageRequest(lang.Name, code, 1, 32768))

	if !strings.Contains(strings.ToLower(resp.Error), "execution timed out") {
		t.Fatalf("signal trap did not timeout as expected; stdout=%q stderr=%q", resp.Output, resp.Error)
	}

	assertDurationWindow(t, resp.CPUTime, 900*time.Millisecond, 2500*time.Millisecond, "signal trap timeout")
}

func scenarioOrphanReaping(t *testing.T, h integrationHarness, lang language) {
	code := lang.load(t, "orphan_maker", nil)
	resp := callSimpleExecute(t, h.baseURL, buildLanguageRequest(lang.Name, code, 2, 32768))
	assertDurationNotExcessive(t, resp.CPUTime, maxOrphanReapResponseDuration, "orphan maker request")

	time.Sleep(400 * time.Millisecond)
	if cnt := countProcessesByComm(t, lang.OrphanName); cnt > 0 {
		t.Fatalf("orphan grandchild leaked to host after request completion: count=%d stderr=%q", cnt, resp.Error)
	}
}

func scenarioInodeExhaustion(t *testing.T, h integrationHarness, lang language) {
	beforeFree := mustFreeBytes(t, os.TempDir())
	beforeCount := countSandboxTempDirs(t)

	code := lang.load(t, "inode_bomb", nil)
	resp := callSimpleExecute(t, h.baseURL, buildLanguageRequest(lang.Name, code, 4, 32768))

	combinedLower := strings.ToLower(resp.Output + "\n" + resp.Error)
	if !containsAny(combinedLower, []string{"inode bomb completed", "no space left", "disk quota exceeded"}) {
		t.Fatalf("inode exhaustion test produced unexpected result; stdout=%q stderr=%q", resp.Output, resp.Error)
	}

	time.Sleep(400 * time.Millisecond)
	assertDiskReclaimed(t, beforeFree, mustFreeBytes(t, os.TempDir()))
	assertNoSandboxLeak(t, beforeCount, countSandboxTempDirs(t))
	mustCreateAndDeleteTempFile(t)
}

func scenarioPrivilegedSyscallDenied(t *testing.T, h integrationHarness, lang language) {
	code := lang.load(t, "try_reboot", nil)
	resp := callSimpleExecute(t, h.baseURL, buildLanguageRequest(lang.Name, code, 3, 32768))
	assertDurationNotExcessive(t, resp.CPUTime, maxPrivilegedSyscallResponseDuration, "privileged reboot syscall")

	combinedLower := strings.ToLower(resp.Output + "\n" + resp.Error)
	if strings.Contains(combinedLower, "reboot succeeded unexpectedly") {
		t.Fatalf("privileged reboot syscall unexpectedly succeeded; stdout=%q stderr=%q", resp.Output, resp.Error)
	}
	if !containsAny(combinedLower, []string{"operation not permitted", "permission denied", "bad system call", "killed", "hangup"}) {
		t.Fatalf("expected privileged syscall denial signal was not observed; stdout=%q stderr=%q", resp.Output, resp.Error)
	}
}
