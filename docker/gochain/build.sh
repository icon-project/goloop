#!/bin/sh
set -e

PRE_PWD=$(pwd)
WORKDIR=$(dirname "$(readlink -f ${0})")
cd $WORKDIR

export IMAGE_BASE=${IMAGE_BASE:-goloop/base-all:latest}

export GOCHAIN_VERSION=${GOCHAIN_VERSION:-$(git describe --always --tags --dirty)}
IMAGE_GOCHAIN=${IMAGE_GOCHAIN:-goloop/gochain:latest}

./update.sh "${IMAGE_GOCHAIN}" ../..

cd $PRE_PWD
