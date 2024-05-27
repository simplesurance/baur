#!/usr/bin/env bash

set -eu -o pipefail

HTTP_PORT=9090

docker run \
	-d \
	--rm \
	-e initialBuckets=mock \
	-p 127.0.0.1:$HTTP_PORT:9090 \
	adobe/s3mock:3.9.1

echo "Started S3 container on port $HTTP_PORT"
