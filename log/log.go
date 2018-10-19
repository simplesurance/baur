package log

import (
	"fmt"
	"log"
	"os"
)

const errorPrefix = "ERROR"

// Logger logs messages
type Logger struct {
	debugEnabled bool
	logger       *log.Logger
}

// StdLogger is the logger that is used from the log functions in this package
var StdLogger = New(false)

// New returns a new Logger that logs to Stderr.
// Debug messages are only printed if debugEnabled is true
func New(debugEnabled bool) *Logger {
	return &Logger{
		debugEnabled: debugEnabled,
		logger:       log.New(os.Stderr, "", 0),
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

	l.logger.Println(v...)
}

// Debugf logs a debug message to stdout.
// It's only shown if debugging is enabled.
func (l *Logger) Debugf(format string, v ...interface{}) {
	if !l.debugEnabled {
		return
	}

	l.logger.Printf(format, v...)
}

// Fatalln logs a message to stderr and terminates the application with an error
func (l *Logger) Fatalln(v ...interface{}) {
	if len(v) != 0 {
		v[0] = fmt.Sprintf("%s: %s", errorPrefix, v[0])
	}

	l.logger.Fatalln(v...)
}

// Fatalf logs a message to stderr and terminates the application with an error
func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.logger.Fatalf(errorPrefix+": "+format, v...)
}

// Errorln logs a message to stderr
func (l *Logger) Errorln(v ...interface{}) {
	if len(v) != 0 {
		v[0] = fmt.Sprintf("%s: %s", errorPrefix, v[0])
	}

	l.logger.Println(v...)
}

// Errorf logs a message to stderr
func (l *Logger) Errorf(format string, v ...interface{}) {
	l.logger.Printf(errorPrefix+": "+format, v...)
}

// Infoln logs a message to stdout
/*
func (l *Logger) Infoln(v ...interface{}) {
	l.logger.Println(v...)
}
*/

// Infof logs a message to stdout
/*
func (l *Logger) Infof(format string, v ...interface{}) {
	l.logger.Printf(format, v...)
}
*/

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

// Infoln logs a message to stdout
/*
func Infoln(v ...interface{}) {
	StdLogger.Infoln(v...)
}
*/

// Infof logs a message to stdout
/*
func Infof(format string, v ...interface{}) {
	StdLogger.Infof(format, v...)
}
*/
