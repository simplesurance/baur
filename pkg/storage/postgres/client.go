// Package postgres is a baur-storage implementation storing data in postgresql.
package postgres

import (
	"context"
	"net/url"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

// Client is a postgres storage client
type Client struct {
	db   dbConn
	pool *pgxpool.Pool
}

// Logger is an interface for logging debug informations
type Logger interface {
	Debugln(v ...any)
}

func redactedURL(strURL string) (string, error) {
	url, err := url.Parse(strURL)
	if err != nil {
		return "", err
	}
	return url.Redacted(), nil
}

// New returns a new postgres client.
// If logger is nil, logging is disabled.
func New(ctx context.Context, url string, logger Logger) (*Client, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}

	if logger != nil {
		cfg.ConnConfig.Logger = &pgxLogger{logger: logger}
		cfg.ConnConfig.LogLevel = pgx.LogLevelInfo

		if rURL, err := redactedURL(url); err == nil {
			logger.Debugln("postgres: establishing connection to", rURL)
		}

	}

	con, err := pgxpool.ConnectConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}

	return &Client{
		pool: con,
		db:   con,
	}, nil
}

// Close closes the connections of the client.
// It always returns a nil error.
func (c *Client) Close() error {
	c.pool.Close()

	return nil
}

type dbConn interface {
	BeginFunc(context.Context, func(pgx.Tx) error) error
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Begin(context.Context) (pgx.Tx, error)
}
