package log

// S3Logger is a logger compatible with the S3 package
type S3Logger struct{}

// Log logs a debug message
func (l *S3Logger) Log(args ...interface{}) {
	Debugln(args)
}
