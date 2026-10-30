#!/bin/sh
set -e

REPOSITORY_NAME=thruputchain
TAG=chuckd.latest

docker build --debug \
  -t $REPOSITORY_NAME:$TAG --file ./Dockerfile ./..
