#!/usr/bin/env bash

fatal() {
	echo "ERROR: $*" >&2
	exit 1
}

set -euo pipefail

head_tag="$(git tag --points-at)"

if [ -z "$head_tag" ]; then
	fatal "git tag is missing for head commit"
fi

ver="$(<ver)"
if [ -z "$ver" ]; then
	fatal "ver file is empty"
fi

if [ "${head_tag#v}" != "$ver" ]; then
	fatal "git tag (without 'v' postfix) and string in ver file do not match: '${head_tag}' vs. '${ver}'"
fi

echo "success: git tag exists and matches with string in ver file: $ver"
