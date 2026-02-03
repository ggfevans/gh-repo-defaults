#!/bin/bash
set -e

GOOS="${1}" GOARCH="${2}" CGO_ENABLED=0 go build -trimpath \
  -ldflags="-s -w -X github.com/ggfevans/gh-mint/cmd.Version=${GH_RELEASE_TAG:-dev}" \
  -o "${3}"
