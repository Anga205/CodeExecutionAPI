package tests

import "time"

// Timeout budgets for the integration suite.
//
// These values are deliberately generous: they exist to fail fast when a
// sandboxed process or the HTTP round-trip hangs, not to measure performance.
// The sandbox enforces the request's own timeout server-side, so a healthy
// request always returns shortly after that deadline plus teardown overhead.
// CI runners are slower and noisier than a developer machine, so every budget
// keeps headroom above the expected execution time.
const (
	// defaultRequestTimeout mirrors config.Limits.DefaultTimeout, used when a
	// request omits the timeout field.
	defaultRequestTimeout = 5 * time.Second

	// responseGrace is the extra time allowed beyond a request's own timeout
	// for process teardown, cgroup cleanup and HTTP transport overhead before
	// the client gives up. A request that exceeds this is considered hung.
	responseGrace = 10 * time.Second

	// maxIOFloodResponseDuration bounds wall-clock time for the I/O flood
	// scenario, whose request timeout is 1s.
	maxIOFloodResponseDuration = 3 * time.Second

	// maxOrphanReapResponseDuration bounds wall-clock time for the orphan
	// reaping scenario, whose request timeout is 2s.
	maxOrphanReapResponseDuration = 3 * time.Second

	// maxPrivilegedSyscallResponseDuration bounds wall-clock time for the
	// privileged syscall scenario. The request timeout is 3s; the extra
	// headroom covers sandbox teardown and CI scheduling jitter.
	maxPrivilegedSyscallResponseDuration = 5 * time.Second
)

// requestDeadline returns the client-side deadline for a request: the
// request's own timeout (or the default when unset) plus responseGrace.
func requestDeadline(req simpleExecuteRequest) time.Duration {
	timeout := time.Duration(req.Timeout) * time.Second
	if timeout <= 0 {
		timeout = defaultRequestTimeout
	}
	return timeout + responseGrace
}
