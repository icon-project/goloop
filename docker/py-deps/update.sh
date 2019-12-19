#!/bin/sh

BASE_DIR=$(dirname $0)
PYTHON_VERSION=${PYTHON_VERSION:-3.7.5}

get_hash_of_dir() {
    local SRC_DIR=$1
    local SUM=$(sha1sum "${SRC_DIR}/pyee/requirements.txt" | cut -d ' ' -f 1)
    echo "${PYTHON_VERSION}-${SUM}"
}

get_hash_of_image() {
    local TAG=$1
    docker image inspect -f '{{.Config.Labels.GOLOOP_PYREQ_SHA}}' ${TAG} 2> /dev/null || echo 'none'
}

update_image() {
    if [ $# -lt 1 ] ; then
	echo "Usage: $0 <image_name> [<src_dir>] [<build_dir>]"
	return 1
    fi

    local TAG=$1
    local SRC_DIR=$2
    if [ "${SRC_DIR}" == "" ] ; then
	SRC_DIR="."
    fi
    local BUILD_DIR=$3

    local GOLOOP_PYREQ_SHA=$(get_hash_of_dir ${SRC_DIR})
    local IMAGE_PYREQ_SHA=$(get_hash_of_image ${TAG})


    if [ "${GOLOOP_PYREQ_SHA}" != "${IMAGE_PYREQ_SHA}" ] ; then
	# Prepare build directory if it's set
	if [ "${BUILD_DIR}" != "" ] ; then
	    rm -rf ${BUILD_DIR}
	    mkdir -p ${BUILD_DIR}
	    cp ${BASE_DIR}/* ${BUILD_DIR}
	else
	    BUILD_DIR=${BASE_DIR}
	fi

	cp ${SRC_DIR}/pyee/requirements.txt ${BUILD_DIR}/

        CDIR=$(pwd)
	cd ${BUILD_DIR}

	echo "Building image ${TAG} for ${GOLOOP_PYREQ_SHA}"
	docker build \
	    --build-arg GOLOOP_PYREQ_SHA=${GOLOOP_PYREQ_SHA} \
	    --build-arg PYTHON_VERSION=${PYTHON_VERSION} \
	    --tag ${TAG} .
	local result=$?

	rm -f requirements.txt
	cd ${CDIR}

	return $result
    else
	echo "Already exist image ${TAG} for ${GOLOOP_PYREQ_SHA}"
	return 0
    fi
    return 0
}

update_image "$@"
