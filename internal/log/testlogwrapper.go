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

func (l *TestLogOutput) Printf(format string, v ...any) {
	l.t.Logf(format, v...)
}

func (l *TestLogOutput) Println(v ...any) {
	l.t.Log(v...)
}
