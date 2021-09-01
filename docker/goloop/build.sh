#!/bin/sh
set -e

PRE_PWD=$(pwd)
WORKDIR=$(dirname "$(readlink -f ${0})")
cd $WORKDIR

export IMAGE_BASE=${IMAGE_BASE:-goloop/base-all:latest}

export GOLOOP_VERSION=${GOLOOP_VERSION:-$(git describe --always --tags --dirty)}
IMAGE_GOLOOP=${IMAGE_GOLOOP:-goloop:latest}

./update.sh "${IMAGE_GOLOOP}" ../..

cd $PRE_PWD
