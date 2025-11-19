#!/bin/sh
set -e

if [ $# -ne 2 ]; then
  echo "Usage: $0 <git_ssh_url> <key_file>"
  exit 1
fi

ssh-keygen -t ed25519 -C $1 -f ~/.ssh/$2
