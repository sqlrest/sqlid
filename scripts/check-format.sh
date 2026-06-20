#!/usr/bin/env bash
# Fail if any Go file is not gofumpt-formatted.
set -euo pipefail

cd "$(dirname "$0")/.."

unformatted="$(go tool gofumpt -l .)"
if [[ -n "${unformatted}" ]]; then
  echo "gofumpt: the following files are not formatted:" >&2
  echo "${unformatted}" >&2
  exit 1
fi
