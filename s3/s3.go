package s3

import (
	"fmt"
	"net/url"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// Client is a S3 uploader client
type Client struct {
	sess     *session.Session
	uploader *s3manager.Uploader
}

// Logger defines the interface for an S3 logger
type Logger interface {
	Debugf(format string, v ...interface{})
	Debugln(v ...interface{})
	DebugEnabled() bool
}

// DefaultRetries is the number of retries for a S3 upload until an error is
// raised
const DefaultRetries = 3

// NewClient returns a new S3 Client, configuration is read from env variables
// or configuration files,
// see https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html
func NewClient(logger Logger) (*Client, error) {
	loglvl := aws.LogLevel(aws.LogOff)
	if logger.DebugEnabled() {
		loglvl = aws.LogLevel(aws.LogDebug)
	}

	cfg := aws.Config{
		Logger:           aws.LoggerFunc(logger.Debugln),
		LogLevel:         loglvl,
		MaxRetries:       aws.Int(DefaultRetries),
		S3ForcePathStyle: aws.Bool(true),
	}

	sess, err := session.NewSession(&cfg)
	if err != nil {
		return nil, err
	}

	return &Client{sess: sess,
		uploader: s3manager.NewUploader(sess),
	}, nil
}

func bucketFromURL(u *url.URL) string {
	return u.Host
}

func fileFromURL(u *url.URL) string {
	return u.Path
}

func verifyURL(u *url.URL) error {
	if u.Scheme != "s3" {
		return fmt.Errorf("unsupported URL scheme '%s'", u.Scheme)
	}

	if len(u.Host) == 0 {
		return fmt.Errorf("bucket missing in url '%s'", u)
	}

	if len(u.Path) == 0 {
		return fmt.Errorf("filename missing in url '%s'", u)
	}

	return nil
}

// Upload uploads a file to an s3 bucket, On success it returns the URL to the
// file.
func (c *Client) Upload(file string, dest string) (string, error) {
	url, err := url.Parse(dest)
	if err != nil {
		return "", err
	}

	if err := verifyURL(url); err != nil {
		return "", err
	}

	f, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer f.Close()

	res, err := c.uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucketFromURL(url)),
		Key:    aws.String(fileFromURL(url)),
		Body:   f,
	})
	if err != nil {
		return "", err
	}

	return res.Location, err
}
