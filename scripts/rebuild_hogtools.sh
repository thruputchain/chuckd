#!/bin/sh
set -ex
SCRIPT_DIR=$(dirname $(readlink -f "$0"))
DIR=$SCRIPT_DIR/..
BUILD_DIR=/tmp/hogtools-build-${USER}
rm -rf ${BUILD_DIR}
mkdir -p ${BUILD_DIR}
cp $DIR/tools/PKGBUILD ${BUILD_DIR}/
cd ${BUILD_DIR}
makepkg -si
cd ~dm/ansible_hog
sudo ansible-playbook misc-config.yml
