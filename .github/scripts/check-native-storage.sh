#!/bin/sh
# Native integration prerequisite for the macOS release; synthetic credentials only.
set -eu
if [ "$(uname -s)" != Darwin ]; then
  echo "Native Keychain validation requires macOS; Linux substitutes are not equivalent." >&2
  exit 2
fi
go test -race -count=1 -timeout=60s ./internal/persistence ./internal/config ./internal/history ./internal/secrets
