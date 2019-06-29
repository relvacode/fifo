#!/usr/bin/env sh
set -x

PKG="github.com/relvacode/fifo"

VERSION=`git tag`
COMMIT=`git rev-parse HEAD`

CGO_ENABLED=0 go build -ldflags "-X ${PKG}/build.Version=${VERSION} -X ${PKG}/build.Commit=${COMMIT}" $@ "${PKG}/cmd/fifo"