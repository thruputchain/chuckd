#!/bin/sh
set -e

REPOSITORY_NAME=groundhogchain
TAG=chuckd.latest

docker build --debug \
  --secret id=GITHUB_TOKEN,env=GITHUB_TOKEN \
  -t $REPOSITORY_NAME:$TAG --file ./Dockerfile ./..
