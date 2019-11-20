#!/bin/sh

BASE_DIR=$(dirname $0)
GOLANG_VERSION=${GOLANG_VERSION:-1.12.4}

get_hash_of_dir() {
    local SRC_DIR=$1
    local SUM=$(sha1sum "${SRC_DIR}/go.sum" | cut -d ' ' -f 1)
    echo "${GOLANG_VERSION}-${SUM}"
}

get_hash_of_image() {
    local TAG=$1
    docker image inspect -f '{{.Config.Labels.GOLOOP_GOMOD_SHA}}' ${TAG} 2> /dev/null || echo 'none'
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

    local GOLOOP_GOMOD_SHA=$(get_hash_of_dir ${SRC_DIR})
    local IMAGE_GOMOD_SHA=$(get_hash_of_image ${TAG})


    if [ "${GOLOOP_GOMOD_SHA}" != "${IMAGE_GOMOD_SHA}" ] ; then
	# Prepare build directory if it's set
	if [ "${BUILD_DIR}" != "" ] ; then
	    rm -rf ${BUILD_DIR}
	    mkdir -p ${BUILD_DIR}
	    cp ${BASE_DIR}/* ${BUILD_DIR}
	else
	    BUILD_DIR=${BASE_DIR}
	fi

	cp ${SRC_DIR}/go.sum ${SRC_DIR}/go.mod ${BUILD_DIR}/

        CDIR=$(pwd)
	cd ${BUILD_DIR}

	echo "Building image ${TAG} for ${GOLOOP_GOMOD_SHA}"
	docker build \
	    --build-arg GOLOOP_GOMOD_SHA=${GOLOOP_GOMOD_SHA} \
	    --build-arg GOLANG_VERSION=${GOLANG_VERSION} \
	    --tag ${TAG} .
	local result=$?

	rm -f go.sum go.mod
	cd ${CDIR}

	return $result
    else
	echo "Already exist image ${TAG} for ${GOLOOP_GOMOD_SHA}"
	return 0
    fi
    return 0
}

update_image "$@"
