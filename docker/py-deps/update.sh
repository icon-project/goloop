#!/bin/sh

BASE_DIR=$(dirname $0)
. ${BASE_DIR}/../version.sh
. ${BASE_DIR}/../function.sh

LABEL="GOLOOP_PYDEP_SHA"

get_hash_of_dir() {
    local SRC_DIR=$1
    local SUM=$(get_hash_of_files \
        "${SRC_DIR}/pyee/requirements.txt" \
        "${SRC_DIR}/docker/py-deps/Dockerfile")
    echo "${PYTHON_VERSION}-alpine${ALPINE_VERSION}-${SUM}"
}

get_hash_of_image() {
    local IMAGE=$1
    docker image inspect -f "{{.Config.Labels.${LABEL}}}" ${IMAGE} 2> /dev/null || echo 'none'
}

update_image() {
    if [ $# -lt 1 ] ; then
        echo "Usage: $0 <image_name> [<src_dir>] [<build_dir>]"
        return 1
    fi

    local TARGET_IMAGE=$1
    local TARGET_REPO=${TARGET_IMAGE%%:*}
    local SRC_DIR=$2
    if [ -z "${SRC_DIR}" ] ; then
        SRC_DIR="."
    fi
    local BUILD_DIR=$3

    local HASH_OF_DIR=$(get_hash_of_dir ${SRC_DIR})
    local HASH_OF_IMAGE=$(get_label_of_image ${LABEL} ${TARGET_IMAGE})

    if [ "${HASH_OF_DIR}" != "${HASH_OF_IMAGE}" ] ; then
        local IMAGE_ID=$(get_id_with_hash ${TARGET_REPO} ${LABEL} ${HASH_OF_DIR})
        if [ "${IMAGE_ID}" != "" ]; then
            echo "Tagging image ${IMAGE_ID} as ${TARGET_IMAGE}"
            docker tag ${IMAGE_ID} ${TARGET_IMAGE}
            return $?
        fi

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

        echo "Building image ${TARGET_IMAGE} for ${HASH_OF_DIR}"
        docker build \
            --build-arg ${LABEL}=${HASH_OF_DIR} \
            --build-arg PYTHON_VERSION=${PYTHON_VERSION} \
            --build-arg ALPINE_VERSION=${ALPINE_VERSION} \
            --tag ${TARGET_IMAGE} .
        local result=$?

        rm -f requirements.txt
        cd ${CDIR}
        return $result
    else
        echo "Reuse image ${TARGET_IMAGE} for ${HASH_OF_DIR}"
        return 0
    fi
    return 0
}

update_image "$@"
