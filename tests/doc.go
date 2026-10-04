// Package tests contains the black-box integration and security test suite for
// the sandbox API.
//
// The suite is split by concern:
//
//   - main_test.go / harness_test.go / assertions_test.go: shared plumbing.
//   - languages_test.go: language descriptors used to parameterize scenarios.
//   - isolation_test.go / scenarios_isolation_test.go: resource and filesystem
//     isolation scenarios, run once per supported language.
//   - hardening_test.go / hostile_chain_test.go: sandbox hardening scenarios.
//   - security_boundary_test.go: namespace, IPC, UTS, procfs and descriptor
//     boundary probe.
//   - overload_test.go: admission-control behaviour under load.
//
// The executor subpackage holds unit tests for the executor internals and does
// not require root:
//
//   - executor/*_test.go: command construction, mount helpers, diagnostics.
//
// Most tests in this package require root and a writable cgroup v2 hierarchy;
// see the README security verification section.
package tests
