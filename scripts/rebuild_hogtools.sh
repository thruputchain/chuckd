#!/bin/sh
set -ex
SCRIPT_DIR=$(dirname $(readlink -f "$0"))
DIR=$SCRIPT_DIR/..
rm -rf /tmp/p/*
cp $DIR/tools/PKGBUILD /tmp/p/
cd /tmp/p/
makepkg -si
cd ~dm/ansible_hog
sudo ansible-playbook misc-config.yml
