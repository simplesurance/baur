package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4"
)

type pgxLogger struct {
	logger Logger
}

func (l *pgxLogger) Log(_ context.Context, level pgx.LogLevel, msg string, data map[string]any) {
	logArgs := make([]any, 2, 2+len(data))
	logArgs[0] = level
	logArgs[1] = msg

	for k, v := range data {
		logArgs = append(logArgs, fmt.Sprintf("%s=%v", k, v))
	}

	l.logger.Debugln(logArgs...)
}
