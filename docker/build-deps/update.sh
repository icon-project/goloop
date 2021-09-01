#!/bin/sh

BASE_DIR=$(dirname $0)
. ${BASE_DIR}/../version.sh
. ${BASE_DIR}/../function.sh

get_label() {
    local LABEL
    case $1 in
    go)
        LABEL="GOLOOP_GOMOD_SHA"
    ;;
    py)
        LABEL="GOLOOP_PYDEP_SHA"
    ;;
    java)
        LABEL="GOLOOP_JADEP_SHA"
    ;;
    rocksdb)
        LABEL="GOLOOP_ROCKSDBDEP_SHA"
    ;;
    build)
        LABEL="GOLOOP_BUILDDEP_SHA"
    ;;
    *)
    ;;
    esac
    echo $LABEL
}

get_dockerfile() {
  local TARGET=$1
  case $TARGET in
  build)
    echo Dockerfile
  ;;
  *)
    echo ${TARGET}.Dockerfile
  ;;
  esac
}

get_hash_of_dir() {
    local TARGET=$1
    local SRC_DIR=$2
    local DOCKERFILE=${SRC_DIR}/docker/build-deps/$(get_dockerfile ${TARGET})
    local SUM
    local HASH_OF_DIR
    case $TARGET in
    go)
        SUM=$(get_hash_of_files \
          "${SRC_DIR}/go.mod" "${SRC_DIR}/go.sum" \
          "${DOCKERFILE}" )
        HASH_OF_DIR="${GOLANG_VERSION}-alpine${ALPINE_VERSION}-${SUM}"
    ;;
    py)
        SUM=$(get_hash_of_files \
          "${SRC_DIR}/pyee/requirements.txt" \
          "${DOCKERFILE}" )
        HASH_OF_DIR="${PYTHON_VERSION}-alpine${ALPINE_VERSION}-${SUM}"
    ;;
    java)
        SUM=$(get_hash_of_files \
          "${DOCKERFILE}" \
          "${SRC_DIR}/javaee/gradle/wrapper/gradle-wrapper.properties")
        HASH_OF_DIR="${JAVA_VERSION}-alpine${ALPINE_VERSION}-${SUM}"
    ;;
    rocksdb)
        SUM=$(get_hash_of_files \
          "${DOCKERFILE}")
        HASH_OF_DIR="${ROCKSDB_VERSION}-alpine${ALPINE_VERSION}-${SUM}"
    ;;
    build)
        SUM=$(get_hash_of_files \
          "${DOCKERFILE}") \
        HASH_OF_DIR="go${GOLANG_VERSION}-rocksdb${ROCKSDB_VERSION}-alpine${ALPINE_VERSION}-${SUM}"
    ;;
    *)
    ;;
    esac

    echo "${HASH_OF_DIR}"
}

extra_files() {
    local TARGET=$1
    local SRC_DIR=$2
    local EXTRA_FILES
    case $TARGET in
    go)
        EXTRA_FILES=("${SRC_DIR}/go.sum" "${SRC_DIR}/go.mod")
    ;;
    py)
        EXTRA_FILES=("${SRC_DIR}/pyee/requirements.txt")
    ;;
    java)
        EXTRA_FILES=("${SRC_DIR}/javaee/gradle*")
    ;;
    rocksdb)
    ;;
    build)
    ;;
    *)
    ;;
    esac
    echo ${EXTRA_FILES[@]}
}

update_image() {
    if [ $# -lt 1 ] ; then
        echo "Usage: $0 <target> [<image_name>] [<src_dir>] [<build_dir>]"
        echo "\t <target>:  build, go, py, java, rocksdb"
        return 1
    fi

    local TARGET=${1}
    local LABEL=$(get_label ${TARGET})
    if [ -z "${LABEL}" ] ; then
        echo "invalid target ${TARGET}"
        return 1
    fi
    echo "TARGET=${TARGET} LABEL=${LABEL}"

    local TARGET_IMAGE=${2:-goloop/${TARGET}-deps:latest}
    local TARGET_REPO=${TARGET_IMAGE%%:*}
    local SRC_DIR=$3
    if [ -z "${SRC_DIR}" ] ; then
        SRC_DIR="."
    fi
    local BUILD_DIR=$4

    local HASH_OF_DIR=$(get_hash_of_dir ${TARGET} ${SRC_DIR})
    # get label check from target image => if hash !=

    local HASH_OF_IMAGE=$(get_label_of_image ${LABEL} ${TARGET_IMAGE})
    echo "HASH_OF_DIR=${HASH_OF_DIR} HASH_OF_IMAGE=${HASH_OF_IMAGE}"
    if [ "${HASH_OF_DIR}" != "${HASH_OF_IMAGE}" ] ; then
        local IMAGE_ID=$(get_id_with_hash ${TARGET_REPO} ${LABEL} ${HASH_OF_DIR})
        if [ "${IMAGE_ID}" != "" ] ; then
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

        EXTRA_FILES=$(extra_files ${TARGET} ${SRC_DIR})
        if [ ! -z "${EXTRA_FILES}" ] ; then
          CP_CMD="cp -r ${EXTRA_FILES} ${BUILD_DIR}/"
          echo ${CP_CMD}
          ${CP_CMD}
        fi

        CDIR=$(pwd)
        cd ${BUILD_DIR}

        local DOCKERFILE=$(get_dockerfile ${TARGET})
        echo "Building image ${TARGET_IMAGE} for ${HASH_OF_DIR}"
        docker build \
            --build-arg ${LABEL}=${HASH_OF_DIR} \
            --build-arg GOLANG_VERSION=${GOLANG_VERSION} \
            --build-arg PYTHON_VERSION=${PYTHON_VERSION} \
            --build-arg JAVA_VERSION=${JAVA_VERSION} \
            --build-arg ROCKSDB_VERSION=${ROCKSDB_VERSION} \
            --build-arg ALPINE_VERSION=${ALPINE_VERSION} \
            --build-arg IMAGE_ROCKSDB_DEPS=${IMAGE_ROCKSDB_DEPS} \
            --build-arg IMAGE_GO_DEPS=${IMAGE_GO_DEPS} \
            --tag ${TARGET_IMAGE} \
            --file ${DOCKERFILE} \
            .
        local result=$?

        if [ ! -z "${EXTRA_FILES}" ] ; then
          RM_CMD="rm -rf "
          EXTRA_FILES=($EXTRA_FILES)
          for EXTRA_FILE in "${EXTRA_FILES[@]}"; do
              echo "EXTRA_FILE=${EXTRA_FILE} STRIP=${EXTRA_FILE##*/}"
              RM_CMD="${RM_CMD} ${EXTRA_FILE##*/}"
          done

          echo ${RM_CMD}
          ${RM_CMD}
        fi

        cd ${CDIR}
        return $result
    else
        echo "Reuse image ${TARGET_IMAGE} for ${HASH_OF_DIR}"
        return 0
    fi
    return 0
}

update_image "$@"
