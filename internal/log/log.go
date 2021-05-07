package log

import (
	"log"
	"os"
	"sync"
)

// Logger logs messages
type Logger struct {
	debugEnabled bool

	output     Output
	outputLock sync.Mutex
}

// Output defines the output channel of a logger to that all log messages are
// written.
type Output interface {
	Printf(format string, v ...interface{})
	Println(v ...interface{})
}

// StdLogger is the logger that is used from the log functions in this package
var StdLogger = New(false)

// New returns a new Logger that logs to Stderr.
// Debug messages are only printed if debugEnabled is true
func New(debugEnabled bool) *Logger {
	return &Logger{
		debugEnabled: debugEnabled,
		output:       log.New(os.Stderr, "", 0),
	}
}

// EnableDebug enables/disables logging debug messages
func (l *Logger) EnableDebug(enabled bool) {
	l.debugEnabled = enabled
}

// DebugEnabled returns true if logging debug messages is enabled
func (l *Logger) DebugEnabled() bool {
	return l.debugEnabled
}

// Debugln logs a debug message to stdout.
// It's only shown if debugging is enabled.
func (l *Logger) Debugln(v ...interface{}) {
	if !l.debugEnabled {
		return
	}

	l.GetOutput().Println(v...)
}

// Debugf logs a debug message to stdout.
// It's only shown if debugging is enabled.
func (l *Logger) Debugf(format string, v ...interface{}) {
	if !l.debugEnabled {
		return
	}

	l.GetOutput().Printf(format, v...)
}

// GetOutput returns the output to which log messages are written.
func (l *Logger) GetOutput() Output {
	l.outputLock.Lock()
	defer l.outputLock.Unlock()

	return l.output
}

// SetOutput changes the output of the logger
func (l *Logger) SetOutput(o Output) {
	l.outputLock.Lock()
	defer l.outputLock.Unlock()

	l.output = o
}

// DebugEnabled returns true if the Stdlogger logs debug messages
func DebugEnabled() bool {
	return StdLogger.DebugEnabled()
}

// Debugln logs a debug message to stdout.
// It's only shown if debugging is enabled.
func Debugln(v ...interface{}) {
	StdLogger.Debugln(v...)
}

// Debugf logs a debug message to stdout.
// It's only shown if debugging is enabled.
func Debugf(format string, v ...interface{}) {
	StdLogger.Debugf(format, v...)
}
