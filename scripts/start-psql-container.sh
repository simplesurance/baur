#!/usr/bin/env bash

set -eu -o pipefail

PSQL_PORT=5434

if ! command -v; then
	echo "docker is not installed"
	exit 1
fi


docker run \
	-d \
	--rm \
	-p 127.0.0.1:$PSQL_PORT:5432 \
	-e POSTGRES_HOST_AUTH_METHOD=trust \
	-e POSTGRES_DB=baur \
	postgres:latest

echo "Started docker PostgreSQL container on port $PSQL_PORT"
