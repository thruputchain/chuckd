#!/bin/sh
set -e

REPOSITORY_NAME=groundhogchain
TAG=chuckdbench.latest

docker build \
  --build-arg GITHUB_TOKEN=$GITHUB_TOKEN \
  -t $REPOSITORY_NAME:$TAG --file ./Dockerfile.bench ./..
