#!/usr/bin/env bash
set -euox pipefail

cd chuckd

go test ./... -covermode=atomic -coverprofile=coverage.out
go tool cover -func=coverage.out
