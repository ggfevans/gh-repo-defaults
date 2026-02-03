#!/bin/bash
set -e

# Called by cli/gh-extension-precompile@v2 with: $1 = release tag
# Must build all platform binaries into dist/

tag="${1:-dev}"
ldflags="-s -w -X github.com/ggfevans/gh-mint/cmd.Version=${tag}"

platforms=(
  linux/amd64 linux/arm64 linux/arm linux/386
  darwin/amd64 darwin/arm64
  windows/amd64 windows/arm64 windows/386
  freebsd/amd64 freebsd/arm64 freebsd/386
)

mkdir -p dist

for platform in "${platforms[@]}"; do
  goos="${platform%/*}"
  goarch="${platform#*/}"
  ext=""
  if [ "$goos" = "windows" ]; then
    ext=".exe"
  fi
  output="dist/${goos}-${goarch}${ext}"
  echo "Building ${output}..."
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" go build -trimpath -ldflags="$ldflags" -o "$output"
done
