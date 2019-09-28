#!/usr/bin/env sh
set -x

PKG="github.com/relvacode/fifo"

VERSION=`git tag | tail -n 1`
COMMIT=`git rev-parse HEAD | tail -n 1`

CGO_ENABLED=0 go build -ldflags "-X ${PKG}/build.Version=${VERSION} -X ${PKG}/build.Commit=${COMMIT} -X ${PKG}/build.Build=docker" $@ "${PKG}/cmd/fifo"
