package log

import (
	"fmt"
	"os"
	"sync"
)

var lock = sync.Mutex{}

// DebugEnabled set to true to print debug message otherwise they are suppressed
var DebugEnabled bool

const actionPrefix = "* "
const errorPrefix = "ERROR: "

// Actionln prints something with the ActionPrefix preprended
func Actionln(v ...interface{}) {
	lock.Lock()
	defer lock.Unlock()

	v = append([]interface{}{actionPrefix}, v...)
	fmt.Println(v...)
}

// Actionf prints something with the ActionPrefix preprended
func Actionf(format string, v ...interface{}) {
	lock.Lock()
	defer lock.Unlock()

	fmt.Printf(actionPrefix+format, v...)
}

// Debugln logs a debug message to stdout.
// It's only shown if debugging is enabled.
func Debugln(v ...interface{}) {
	lock.Lock()
	defer lock.Unlock()

	if !DebugEnabled {
		return
	}

	fmt.Println(v...)
}

// Debugf logs a debug message to stdout.
// It's only shown if debugging is enabled.
func Debugf(format string, v ...interface{}) {
	lock.Lock()
	defer lock.Unlock()

	if !DebugEnabled {
		return
	}

	fmt.Printf(format, v...)
}

// Fatalln logs a message to stderr and terminates the application with an error
func Fatalln(v ...interface{}) {
	lock.Lock()
	defer lock.Unlock()

	v = append([]interface{}{errorPrefix}, v...)
	fmt.Fprintln(os.Stderr, v...)

	os.Exit(1)
}

// Fatalf logs a message to stderr and terminates the application with an error
func Fatalf(format string, v ...interface{}) {
	lock.Lock()
	defer lock.Unlock()

	fmt.Fprintf(os.Stderr, errorPrefix+format, v...)
	os.Exit(1)
}

// Errorln logs a message to stderr
func Errorln(v ...interface{}) {
	lock.Lock()
	defer lock.Unlock()

	v = append([]interface{}{errorPrefix}, v...)
	fmt.Fprintln(os.Stderr, v...)
}

// Errorf logs a message to stderr
func Errorf(format string, v ...interface{}) {
	lock.Lock()
	defer lock.Unlock()

	fmt.Fprintf(os.Stderr, errorPrefix+format, v...)
}

// Infoln logs a message to stdout
func Infoln(v ...interface{}) {
	lock.Lock()
	defer lock.Unlock()

	fmt.Println(v...)
}

// Infof logs a message to stdout
func Infof(format string, v ...interface{}) {
	lock.Lock()
	defer lock.Unlock()

	fmt.Printf(format, v...)
}
