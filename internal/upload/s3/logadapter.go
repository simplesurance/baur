package s3

import "github.com/aws/smithy-go/logging"

const logPrefix = "aws-sdk-go-v2: "

type s3Logger struct {
	logger Logger
}

func (l *s3Logger) Logf(_ logging.Classification, format string, v ...interface{}) {
	l.logger.Debugf(logPrefix+format, v...)
}
