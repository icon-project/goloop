#!/bin/sh
set -e

PRE_PWD=$(pwd)
WORKDIR=$(dirname "$(readlink -f ${0})")
cd $WORKDIR

IMAGE_GO_DEPS=${IMAGE_GO_DEPS:-goloop/go-deps:latest}

./update.sh "${IMAGE_GO_DEPS}" ../..

cd $PRE_PWD
