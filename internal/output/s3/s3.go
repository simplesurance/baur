package s3

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Client downloads and uploads objects from/to S3.
type Client struct {
	uploader   *manager.Uploader
	downloader *manager.Downloader
}

// Logger defines the interface for an S3 logger
type Logger interface {
	Debugf(format string, v ...any)
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
		uploader:   manager.NewUploader(clt),
		downloader: manager.NewDownloader(clt),
	}, nil
}

// Upload uploads a file to an s3 bucket, on success it returns the s3:// URL
// of the object.
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

	url := url.URL{
		Scheme: "s3",
		Host:   bucket,
		Path:   *res.Key,
	}

	return url.String(), err
}

func (c *Client) Download(ctx context.Context, bucket, key, filepath string) error {
	f, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o640)
	if err != nil {
		return err
	}

	_, err = c.downloader.Download(ctx, f, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return err
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("writing to file failed: %w", err)
	}
	return nil
}

func ParseURL(u string) (bucket, key string, err error) {
	url, err := url.Parse(u)
	if err != nil {
		return "", "", err
	}

	if len(url.Scheme) > 0 && url.Scheme != "s3" {
		return "", "", fmt.Errorf("scheme is %s, expecting s3 or an empty one", url.Scheme)
	}
	if url.Host == "" {
		return "", "", errors.New("bucket part is missing")
	}
	if url.Path == "" {
		return "", "", errors.New("object key part is missing")
	}

	return url.Host, strings.TrimPrefix(url.Path, "/"), nil
}
