#!/usr/bin/env sh
set -eu

mkdir -p dist

build() {
  goos="$1"
  goarch="$2"
  ext="${3:-}"
  out="dist/psstd-${goos}-${goarch}${ext}"
  echo "building ${out}"
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" go build -trimpath -ldflags="-s -w" -o "$out" .
}

build linux amd64
build linux arm64
build darwin amd64
build darwin arm64
build windows amd64 .exe
build windows arm64 .exe
