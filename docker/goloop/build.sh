#!/bin/sh
set -e

PRE_PWD=$(pwd)
WORKDIR=$(dirname "$(readlink -f ${0})")
cd $WORKDIR

export GOLOOP_VERSION=${GOLOOP_VERSION:-$(git describe --always --tags --dirty)}
export IMAGE_PY_DEPS=${IMAGE_PY_DEPS:-goloop/py-deps:latest}
IMAGE_GOLOOP=${IMAGE_GOLOOP:-goloop:latest}

./update.sh "${IMAGE_GOLOOP}" ../..

cd $PRE_PWD
