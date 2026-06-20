#!/usr/bin/env bash
# Fail if any production (non-test) function exceeds cognitive complexity 7.
set -euo pipefail

cd "$(dirname "$0")/.."

# --cached --others --exclude-standard lists tracked and untracked-but-not-ignored
# files, so the gate works pre-commit and excludes anything in .gitignore.
findings="$(git ls-files --cached --others --exclude-standard '*.go' | grep -v '_test\.go$' | xargs go tool gocognit -over 7)"
if [[ -n "${findings}" ]]; then
  echo "gocognit: functions exceeding cognitive complexity 7:" >&2
  echo "${findings}" >&2
  exit 1
fi
