#!/bin/sh
set -e

PRE_PWD=$(pwd)
WORKDIR=$(dirname "$(readlink -f ${0})")
cd $WORKDIR

export IMAGE_GO_DEPS=${IMAGE_GO_DEPS:-goloop/go-deps:latest}
export IMAGE_PY_DEPS=${IMAGE_PY_DEPS:-goloop/py-deps:latest}
export IMAGE_JAVA_DEPS=${IMAGE_JAVA_DEPS:-goloop/java-deps:latest}
export IMAGE_ROCKSDB_DEPS=${IMAGE_ROCKSDB_DEPS:-goloop/rocksdb-deps:latest}

if [ $# -lt 1 ] ; then
    echo "Usage: $0 <target>"
    echo "\t <target>:  build, go, py, java, rocksdb"
    return 1
fi
TARGET=${1}
IMAGE_BUILD_DEPS=${IMAGE_BUILD_DEPS:-goloop/${TARGET}-deps:latest}

./update.sh ${TARGET} "${IMAGE_BUILD_DEPS}" ../..

cd $PRE_PWD
