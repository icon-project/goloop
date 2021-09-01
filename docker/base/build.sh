#!/bin/sh
set -e

PRE_PWD=$(pwd)
WORKDIR=$(dirname "$(readlink -f ${0})")
cd $WORKDIR

export IMAGE_PY_DEPS=${IMAGE_PY_DEPS:-goloop/py-deps:latest}
export IMAGE_ROCKSDB_DEPS=${IMAGE_ROCKSDB_DEPS:-goloop/rocksdb-deps:latest}

ENGINE=${1}
IMAGE_SUFFIX=-${ENGINE}
IMAGE_BASE=${IMAGE_BASE:-goloop/base-${ENGINE}latest}

./update.sh "${ENGINE}" "${IMAGE_BASE}" ../..

cd $PRE_PWD
