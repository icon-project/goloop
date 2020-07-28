#!/bin/sh
set -e

PRE_PWD=$(pwd)
WORKDIR=$(dirname "$(readlink -f ${0})")
cd $WORKDIR

export GOCHAIN_VERSION=${GOCHAIN_VERSION:-$(git describe --always --tags --dirty)}
export IMAGE_PY_DEPS=${IMAGE_PY_DEPS:-goloop/py-deps:latest}
IMAGE_GOCHAIN=${IMAGE_GOCHAIN:-goloop/gochain:latest}

./update.sh "${IMAGE_GOCHAIN}" ../..

cd $PRE_PWD
