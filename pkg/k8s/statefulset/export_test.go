package statefulset

import "time"

// SetNotReadyLogInterval overrides the ReadyReplicasWatch "not ready yet" log
// cadence so tests can exercise the ticker branch without a multi-second wait.
// It returns a function that restores the previous value.
func SetNotReadyLogInterval(d time.Duration) (restore func()) {
	old := notReadyLogInterval
	notReadyLogInterval = d
	return func() { notReadyLogInterval = old }
}
