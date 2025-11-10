#!/usr/bin/env bash
set -euox pipefail

GOPRIVATE=github.com/groundhogchain/*
GONOSUMDB=github.com/groundhogchain/*
GONOPROXY=github.com/groundhogchain/*

git config --global url."git@github.com:".insteadOf "https://github.com/"

git clone git@github.com:groundhogchain/chuckd.git

cd chuckd

go build ./...
go test ./... -covermode=atomic -coverprofile=coverage.out
go tool cover -func=coverage.out
