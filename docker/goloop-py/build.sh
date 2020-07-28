#!/bin/sh
set -e

PRE_PWD=$(pwd)
WORKDIR=$(dirname "$(readlink -f ${0})")
cd $WORKDIR

export GOLOOP_VERSION=${GOLOOP_VERSION:-$(git describe --always --tags --dirty)}
export IMAGE_PY_DEPS=${IMAGE_PY_DEPS:-goloop/py-deps:latest}
IMAGE_GOLOOP_PY=${IMAGE_GOLOOP_PY:-goloop-py:latest}

./update.sh "${IMAGE_GOLOOP_PY}" ../..

cd $PRE_PWD
