#!/usr/bin/env sh
set -eu

npm --prefix web test -- --run
npm --prefix web run build
go test ./... -count=1
go vet ./...
npm --prefix web run e2e
