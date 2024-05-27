package s3test

import "testing"

func SetupEnv(t *testing.T) {
	t.Setenv("AWS_ENDPOINT_URL", "http://localhost:9090")

	// the AWS credentials  and region must be set to something when
	// uploading to S3Mock via the aws sdk, otherwise the upload will fail,
	// the actual values are arbitrary, they
	t.Setenv("AWS_REGION", "eu-central-1")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "123")
	t.Setenv("AWS_ACCESS_KEY_ID", "123")
}
