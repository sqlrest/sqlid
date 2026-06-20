#!/usr/bin/env bash
# Run the test suite and fail unless total statement coverage is exactly 100%.
set -euo pipefail

cd "$(dirname "$0")/.."

profile="$(mktemp)"
trap 'rm -f "${profile}"' EXIT

go test -covermode=atomic -coverprofile="${profile}" ./...

total="$(go tool cover -func="${profile}" | awk '/^total:/ {print $3}')"
if [[ "${total}" != "100.0%" ]]; then
  echo "coverage: total ${total}, want 100.0%" >&2
  exit 1
fi
echo "coverage: ${total}"
