#!/usr/bin/env bash

set -eu -o pipefail

PSQL_PORT=5434

docker run \
	-d \
	--rm \
	-p 127.0.0.1:$PSQL_PORT:5432 \
	-e POSTGRES_HOST_AUTH_METHOD=trust \
	-e POSTGRES_DB=baur \
	postgres:14

echo "Started docker PostgreSQL container on port $PSQL_PORT"
