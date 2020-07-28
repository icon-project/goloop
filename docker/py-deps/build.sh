#!/bin/sh
set -e

PRE_PWD=$(pwd)
WORKDIR=$(dirname "$(readlink -f ${0})")
cd $WORKDIR

IMAGE_PY_DEPS=${IMAGE_PY_DEPS:-goloop/py-deps:latest}

./update.sh "${IMAGE_PY_DEPS}" ../..

cd $PRE_PWD
