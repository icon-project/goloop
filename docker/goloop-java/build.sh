#!/bin/sh
set -e

PRE_PWD=$(pwd)
WORKDIR=$(dirname "$(readlink -f ${0})")
cd $WORKDIR

export GOLOOP_VERSION=${GOLOOP_VERSION:-$(git describe --always --tags --dirty)}
IMAGE_GOLOOP_JAVA=${IMAGE_GOLOOP_JAVA:-goloop-java:latest}

./update.sh "${IMAGE_GOLOOP_JAVA}" ../..

cd $PRE_PWD
