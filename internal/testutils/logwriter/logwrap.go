package logwriter

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
)

type Logger struct {
	t   *testing.T
	w   io.Writer
	buf bytes.Buffer
	mu  sync.Mutex
}

// New returns an io.writer compatible writer that writes everything to w and
// to t.Logf.
// It also registers Logger.Flush() as cleanup method for testing.T.
func New(t *testing.T, w io.Writer) *Logger {
	l := Logger{
		t: t,
		w: w,
	}

	t.Cleanup(func() { _ = l.Flush() })

	return &l
}

func (l *Logger) Write(p []byte) (int, error) {
	l.t.Helper()
	l.mu.Lock()
	defer l.mu.Unlock()

	n, writerErr := l.w.Write(p)
	_, _ = l.buf.Write(p[0:n]) // returns always a nil error

	if !bytes.ContainsRune(p, '\n') {
		return n, writerErr
	}

	for {
		line, err := l.buf.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				panic(fmt.Sprintf("logwrap: reading from buffer failed: %s", err))
			}
			// add chunk without line-ending back to buffer
			_, _ = l.buf.WriteString(line) // returns always a nil error
			break
		}

		l.t.Log(strings.TrimRight(line, "\n"))
	}

	return n, writerErr
}

// Flush writes all bytes in the buffer to the test logger and the io.writer.
// It always returns a nil error and panics on errors.
func (l *Logger) Flush() error {
	l.t.Helper()

	l.mu.Lock()
	defer l.mu.Unlock()

	b := l.buf.Bytes()

	if len(b) == 0 {
		return nil
	}

	_, err := l.w.Write(b)
	if err != nil {
		panic(fmt.Sprintf("logwrap: writing to w failed: %s", err))
	}

	l.t.Log(string(b))

	return nil
}
