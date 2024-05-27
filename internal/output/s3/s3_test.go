package s3

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseURL(t *testing.T) {
	type tc struct {
		url            string
		expectedBucket string
		expectedKey    string
	}

	tcs := []*tc{
		{
			url:            "s3://bucket/123",
			expectedBucket: "bucket",
			expectedKey:    "123",
		},
		{
			url:            "s3://bucket/a/b/c/d",
			expectedBucket: "bucket",
			expectedKey:    "a/b/c/d",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.url, func(t *testing.T) {
			bucket, key, err := ParseURL(tc.url)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedBucket, bucket)
			assert.Equal(t, tc.expectedKey, key)
		})
	}
}
