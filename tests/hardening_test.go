package tests

import (
	"strings"
	"testing"
)

// hardeningScenario is one sandbox-hardening check, parameterized by language.
type hardeningScenario struct {
	name string
	run  func(t *testing.T, h integrationHarness, lang language)
}

var hardeningScenarios = []hardeningScenario{
	{name: "sandbox cannot write to host /etc/passwd", run: scenarioCannotWriteHostEtcPasswd},
	{name: "remounting /etc read-write is denied", run: scenarioRemountEtcReadWriteDenied},
	{name: "sandbox process uid is not host root", run: scenarioProcessUIDIsNotHostRoot},
	{name: "dangerous syscalls are blocked by seccomp", run: scenarioDangerousSyscallsBlocked},
	{name: "sandbox environment does not expose host", run: scenarioEnvironmentDoesNotExposeHost},
}

func runHardeningSuite(t *testing.T, h integrationHarness, lang language) {
	t.Helper()
	for _, scenario := range hardeningScenarios {
		scenario := scenario
		t.Run(scenario.name, func(t *testing.T) {
			scenario.run(t, h, lang)
		})
	}
}

func TestSandboxHardeningC(t *testing.T) {
	h := newHarness(t)
	runHardeningSuite(t, h, langC)
	t.Run("hostile boundary attack chain cannot cross sandbox", func(t *testing.T) {
		testHostileBoundaryAttackChain(t, h)
	})
}

func TestSandboxHardeningPython3(t *testing.T) {
	runHardeningSuite(t, newHarness(t), langPython)
}

func scenarioCannotWriteHostEtcPasswd(t *testing.T, h integrationHarness, lang language) {
	beforeChecksum := mustFileSHA256(t, "/etc/passwd")

	code := lang.load(t, "etc_write_denied", nil)
	resp := callSimpleExecute(t, h.baseURL, buildLanguageRequest(lang.Name, code, 3, 32768))

	afterChecksum := mustFileSHA256(t, "/etc/passwd")
	if beforeChecksum != afterChecksum {
		t.Fatalf("host /etc/passwd checksum changed unexpectedly; before=%s after=%s", beforeChecksum, afterChecksum)
	}

	combinedLower := strings.ToLower(resp.Output + "\n" + resp.Error)
	if strings.Contains(combinedLower, "write succeeded") {
		t.Fatalf("sandbox unexpectedly wrote to /etc/passwd; stdout=%q stderr=%q", resp.Output, resp.Error)
	}
	if !strings.Contains(resp.Output, "write correctly denied") {
		t.Fatalf("expected write denial marker in stdout; stdout=%q stderr=%q", resp.Output, resp.Error)
	}
}

func scenarioRemountEtcReadWriteDenied(t *testing.T, h integrationHarness, lang language) {
	code := lang.load(t, "remount_etc_rw_denied", nil)
	resp := callSimpleExecute(t, h.baseURL, buildLanguageRequest(lang.Name, code, 3, 32768))

	combinedLower := strings.ToLower(resp.Output + "\n" + resp.Error)
	if strings.Contains(combinedLower, "remount succeeded") {
		t.Fatalf("sandbox unexpectedly remounted /etc read-write; stdout=%q stderr=%q", resp.Output, resp.Error)
	}
	if !strings.Contains(resp.Output, "remount correctly denied") {
		t.Fatalf("expected remount denial marker in stdout; stdout=%q stderr=%q", resp.Output, resp.Error)
	}
}

func scenarioProcessUIDIsNotHostRoot(t *testing.T, h integrationHarness, lang language) {
	code := lang.load(t, "not_host_root", nil)
	resp := callSimpleExecute(t, h.baseURL, buildLanguageRequest(lang.Name, code, 3, 32768))

	combinedLower := strings.ToLower(resp.Output + "\n" + resp.Error)
	if strings.Contains(combinedLower, "is host root: bad") {
		t.Fatalf("sandbox gained host-root-equivalent privileges; stdout=%q stderr=%q", resp.Output, resp.Error)
	}
	if !strings.Contains(resp.Output, "not host root: ok") {
		t.Fatalf("expected non-host-root marker in stdout; stdout=%q stderr=%q", resp.Output, resp.Error)
	}
}

func scenarioDangerousSyscallsBlocked(t *testing.T, h integrationHarness, lang language) {
	code := lang.load(t, "dangerous_syscalls_seccomp", nil)
	resp := callSimpleExecute(t, h.baseURL, buildLanguageRequest(lang.Name, code, 3, 32768))

	combinedLower := strings.ToLower(resp.Output + "\n" + resp.Error)
	if strings.Contains(combinedLower, "syscall succeeded unexpectedly") {
		t.Fatalf("dangerous syscall unexpectedly succeeded; stdout=%q stderr=%q", resp.Output, resp.Error)
	}
	if strings.Contains(resp.Output, "syscall denied: ok") {
		return
	}
	if containsAny(combinedLower, []string{"bad system call", "sigsys"}) {
		return
	}
	t.Fatalf("expected syscall denial signal was not observed; stdout=%q stderr=%q", resp.Output, resp.Error)
}

func scenarioEnvironmentDoesNotExposeHost(t *testing.T, h integrationHarness, lang language) {
	t.Setenv("SANDBOX_SECRET_CANARY", "super-secret")

	code := lang.load(t, "env_leak_probe", nil)
	resp := callSimpleExecute(t, h.baseURL, buildLanguageRequest(lang.Name, code, 3, 32768))

	stdoutLower := strings.ToLower(resp.Output)
	stderrLower := strings.ToLower(resp.Error)
	if strings.Contains(stdoutLower, "env_leak:sandbox_secret_canary") {
		t.Fatalf("sandbox leaked sentinel env var; stdout=%q stderr=%q", resp.Output, resp.Error)
	}
	if strings.Contains(stdoutLower, "super-secret") || strings.Contains(stderrLower, "super-secret") {
		t.Fatalf("sandbox leaked sentinel env var value; stdout=%q stderr=%q", resp.Output, resp.Error)
	}
}
