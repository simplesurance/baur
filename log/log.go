package log

import (
	"fmt"
	"os"
)

var DebugEnabled bool

const ActionPrefix = "* "

// Action prints something with the ActionPrefix preprended
func Actionln(v ...interface{}) {
	v = append([]interface{}{ActionPrefix}, v)
	fmt.Println(v...)

}

// Actionf prints something with the ActionPrefix preprended
func Actionf(format string, v ...interface{}) {
	fmt.Printf(ActionPrefix+format, v...)
}

// Debug logs a debug message to stdout.
// It's only shown if debugging is enabled.
func Debugln(v ...interface{}) {
	if !DebugEnabled {
		return
	}

	fmt.Println(v...)
}

// Debugf logs a debug message to stdout.
// It's only shown if debugging is enabled.
func Debugf(format string, v ...interface{}) {
	if !DebugEnabled {
		return
	}

	fmt.Printf(format, v...)
}

// Fatal logs a message to stderr and terminates the application with an error
func Fatalln(v ...interface{}) {
	fmt.Fprintln(os.Stderr, v...)
	os.Exit(1)
}

// Fatalf logs a message to stderr and terminates the application with an error
func Fatalf(format string, v ...interface{}) {
	fmt.Fprintf(os.Stderr, format, v...)
	os.Exit(1)
}

// Error logs a message to stderr
func Errorln(v ...interface{}) {
	fmt.Fprintln(os.Stderr, v...)
}

// Errorf logs a message to stderr
func Errorf(format string, v ...interface{}) {
	fmt.Fprintf(os.Stderr, format, v...)
}

// Info logs a message to stdout
func Infoln(v ...interface{}) {
	fmt.Println(v...)
}

// Infof logs a message to stdout
func Infof(format string, v ...interface{}) {
	fmt.Printf(format, v...)
}
