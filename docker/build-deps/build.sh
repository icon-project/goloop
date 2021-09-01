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
case $TARGET in
go)
    IMAGE_DEPS=${IMAGE_GO_DEPS}
;;
py)
    IMAGE_DEPS=${IMAGE_PY_DEPS}
;;
java)
    IMAGE_DEPS=${IMAGE_JAVA_DEPS}
;;
rocksdb)
    IMAGE_DEPS=${IMAGE_ROCKSDB_DEPS}
;;
build)
    IMAGE_DEPS=${IMAGE_BUILD_DEPS}
;;
*)
;;
esac
if [ -z "${IMAGE_DEPS}" ]; then
  IMAGE_DEPS=goloop/${TARGET}-deps:latest
fi

./update.sh ${TARGET} "${IMAGE_DEPS}" ../..

cd $PRE_PWD
