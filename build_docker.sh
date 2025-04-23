#!/bin/sh
set -e

REPOSITORY_NAME=groundhogchain
TAG=chuckd.latest

#   --secret id=github_token,env=GITHUB_TOKEN \
docker build --progress=plain \
  --build-arg GITHUB_TOKEN=$GITHUB_TOKEN \
  -t $REPOSITORY_NAME:$TAG .
