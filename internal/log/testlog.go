package log

import "testing"

// RedirectToTestingLog redirects all log output while a testcase is executed
// to t.Log.
// When the testcase finished, the logger output and the debug log level is
// restored to the previous values.
func RedirectToTestingLog(t *testing.T) {
	oldLogOut := StdLogger.GetOutput()
	oldDebugEnabled := StdLogger.DebugEnabled()

	StdLogger.SetOutput(NewTestLogOutput(t))
	StdLogger.EnableDebug(true)

	t.Cleanup(func() {
		StdLogger.SetOutput(oldLogOut)
		StdLogger.EnableDebug(oldDebugEnabled)
	})
}
