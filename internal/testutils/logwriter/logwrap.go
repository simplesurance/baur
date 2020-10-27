package logwriter

import (
	"io"
	"testing"
)

type Logger struct {
	t *testing.T
	w io.Writer
}

// New returns an io.writer compatible writer that writes everything to w and
// to t.Logf
func New(t *testing.T, w io.Writer) *Logger {
	return &Logger{
		t: t,
		w: w,
	}
}

func (l *Logger) Write(p []byte) (n int, err error) {
	n, err = l.w.Write(p)
	l.t.Logf("%x", p[0:n])

	return n, err
}
