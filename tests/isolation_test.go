package tests

import "testing"

// runIsolationSuite executes every isolation scenario for one language.
func runIsolationSuite(t *testing.T, h integrationHarness, lang language) {
	t.Helper()
	for _, scenario := range isolationScenarios {
		scenario := scenario
		t.Run(scenario.name, func(t *testing.T) {
			scenario.run(t, h, lang)
		})
	}
}

func TestSandboxIsolationC(t *testing.T) {
	runIsolationSuite(t, newHarness(t), langC)
}

func TestSandboxIsolationPython3(t *testing.T) {
	runIsolationSuite(t, newHarness(t), langPython)
}

func TestSandboxIsolationCpp(t *testing.T) {
	runIsolationSuite(t, newHarness(t), langCpp)
}

func TestSandboxIsolationJava(t *testing.T) {
	runIsolationSuite(t, newHarness(t), langJava)
}
