#!/bin/sh

#
# Copyright 2021 ICON Foundation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

BASE_DIR=$(dirname $0)
. ${BASE_DIR}/../version.sh

build_image() {
    if [ $# -lt 1 ] ; then
        echo "Usage: $0 <image_name> [<src_dir>] [<build_dir>]"
        return 1
    fi

    local TAG=$1
    local SRC_DIR=$2
    if [ -z "${SRC_DIR}" ] ; then
        SRC_DIR="."
    fi
    local BUILD_DIR=$3

    # Prepare build directory if it's set
    if [ "${BUILD_DIR}" != "" ] ; then
        rm -rf ${BUILD_DIR}
        mkdir -p ${BUILD_DIR}
        cp ${BASE_DIR}/* ${BUILD_DIR}
    else
        BUILD_DIR=${BASE_DIR}
    fi

    JAVAEE_VERSION=$(grep "^VERSION=" ${SRC_DIR}/javaee/gradle.properties | cut -d= -f2)
    BIN_DIR=${BIN_DIR:-${SRC_DIR}/bin}
    if [ "${GOBUILD_TAGS}" != "" ] ; then
	    LCIMPORT_VERSION="${LCIMPORT_VERSION}-tags(${GOBUILD_TAGS})"
    fi

    # copy required files to ${BUILD_DIR}/dist
    rm -rf ${BUILD_DIR}/dist
    mkdir -p ${BUILD_DIR}/dist/bin
    cp ${BIN_DIR}/lcimport ${BUILD_DIR}/dist/bin/

    mkdir -p ${BUILD_DIR}/dist/pyee
    cp ${SRC_DIR}/build/iconee/dist/iconee-*.whl ${BUILD_DIR}/dist/pyee/
    # cp ${SRC_DIR}/javaee/app/execman/build/distributions/execman-${JAVAEE_VERSION}.zip ${BUILD_DIR}/dist/

    CDIR=$(pwd)
    cd ${BUILD_DIR}

    echo "Building image ${TAG}"
    docker build \
        --build-arg IMAGE_BASE="${IMAGE_BASE}" \
        --build-arg JAVAEE_VERSION="${JAVAEE_VERSION}" \
        --build-arg LCIMPORT_VERSION="${LCIMPORT_VERSION}" \
        --tag ${TAG} .
    local result=$?

    cd ${CDIR}
    rm -rf ${BUILD_DIR}/dist
    return $result
}

build_image "$@"
