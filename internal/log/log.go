package log

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/fatih/color"
)

var errorPrefix = color.New(color.FgRed).Sprint("ERROR: ")

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
	Fatalf(format string, v ...interface{})
	Fatalln(v ...interface{})
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

	l.getOutput().Println(v...)
}

// Debugf logs a debug message to stdout.
// It's only shown if debugging is enabled.
func (l *Logger) Debugf(format string, v ...interface{}) {
	if !l.debugEnabled {
		return
	}

	l.getOutput().Printf(format, v...)
}

// Fatalln logs a message to stderr and terminates the application with an error
func (l *Logger) Fatalln(v ...interface{}) {
	if len(v) != 0 {
		v[0] = fmt.Sprintf("%s%s", errorPrefix, v[0])
	}

	l.getOutput().Fatalln(v...)
}

// Fatalf logs a message to stderr and terminates the application with an error
func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.getOutput().Fatalf(errorPrefix+format, v...)
}

// Errorln logs a message to stderr
func (l *Logger) Errorln(v ...interface{}) {
	if len(v) != 0 {
		v[0] = fmt.Sprintf("%s %s", errorPrefix, v[0])
	}

	l.getOutput().Println(v...)
}

// Errorf logs a message to stderr
func (l *Logger) Errorf(format string, v ...interface{}) {
	l.getOutput().Printf(errorPrefix+" "+format, v...)
}

func (l *Logger) getOutput() Output {
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

// Fatalln logs a message to stderr and terminates the application with an error
func Fatalln(v ...interface{}) {
	StdLogger.Fatalln(v...)
}

// Fatalf logs a message to stderr and terminates the application with an error
func Fatalf(format string, v ...interface{}) {
	StdLogger.Fatalf(format, v...)
}

// Errorln logs a message to stderr
func Errorln(v ...interface{}) {
	StdLogger.Errorln(v...)
}

// Errorf logs a message to stderr
func Errorf(format string, v ...interface{}) {
	StdLogger.Errorf(format, v...)
}
