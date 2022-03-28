package s3

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Client is a S3 uploader client
type Client struct {
	uploader *manager.Uploader
}

// Logger defines the interface for an S3 logger
type Logger interface {
	Debugf(format string, v ...interface{})
}

// DefaultRetries is the number of retries for a S3 upload until an error is
// raised
const DefaultRetries = 3

// NewClient returns a new S3 Client, configuration is read from env variables
// or configuration files,
// see https://docs.aws.amazon.com/sdkref/latest/guide/creds-config-files.html
func NewClient(ctx context.Context, logger Logger) (*Client, error) {
	s3Logger := &s3Logger{logger: logger}

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRetryMaxAttempts(DefaultRetries),
		config.WithLogger(s3Logger),
		config.WithLogConfigurationWarnings(true),
	)
	if err != nil {
		return nil, err
	}

	clt := s3.NewFromConfig(
		cfg,
		func(o *s3.Options) {
			o.UsePathStyle = true
			o.Logger = s3Logger
			o.ClientLogMode = aws.LogRetries | aws.LogRequest | aws.LogResponse
		},
	)

	return &Client{
		uploader: manager.NewUploader(clt),
	}, nil
}

// Upload uploads a file to an s3 bucket, on success it returns the URL to the
// file.
func (c *Client) Upload(filepath, bucket, key string) (string, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	res, err := c.uploader.Upload(context.TODO(),
		&s3.PutObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
			Body:   f,
		},
	)
	if err != nil {
		return "", err
	}

	return res.Location, err
}
