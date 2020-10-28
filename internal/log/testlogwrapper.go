package log

import "testing"

type TestLogOutput struct {
	t *testing.T
}

// NewTestLogOutput wraps the logger of testing.T to provide the Output
// interface.
func NewTestLogOutput(t *testing.T) *TestLogOutput {
	return &TestLogOutput{t: t}
}

func (l *TestLogOutput) Printf(format string, v ...interface{}) {
	l.t.Logf(format, v...)
}

func (l *TestLogOutput) Println(v ...interface{}) {
	l.t.Log(v...)
}
