#!/bin/sh
set -e

PRE_PWD=$(pwd)
WORKDIR=$(dirname "$(readlink -f ${0})")
cd $WORKDIR

#refer ../py-deps/build.sh
PYREQ_SHA=$(sha1sum ../../pyee/requirements.txt | cut -d ' ' -f 1)
REPO_PY_DEPS=${REPO_PY_DEPS:-goloop/py-deps}
TAG_PY_DEPS=${TAG_PY_DEPS:-$(docker images --filter="reference=$REPO_PY_DEPS" --filter="label=GOLOOP_PYREQ_SHA=${PYREQ_SHA}" --format="{{.Tag}}" | head -n 1)}
if [ "${TAG_PY_DEPS}" != "" ] ;then
  TAG_SLUG=${TAG_PY_DEPS//\//__}
  BUILD_ARG_TAG_PY_DEPS="--build-arg=TAG_PY_DEPS=${TAG_SLUG} "
fi

GOCHAIN_VERSION=${GOCHAIN_VERSION:-$(git describe --always --tags --dirty)}
REPO_GOCHAIN=${REPO_GOCHAIN:-goloop/gochain}
PRE_GOCHAIN_VERSION=$(docker image inspect ${REPO_GOCHAIN} -f "{{.Config.Labels.GOCHAIN_VERSION}}" || echo "none")
if [ "${GOCHAIN_VERSION}" != "${PRE_GOCHAIN_VERSION}" ]
then
  echo "Build image ${REPO_GOCHAIN} using ${REPO_PY_DEPS} with TAG_PY_DEPS:${TAG_PY_DEPS}"
  mkdir dist
  cp ../../pyee/dist/pyexec-*.whl ./dist/
  cp ../../bin/gochain ./dist/
  cp ../../javaee/app/exectest/build/distributions/exectest.zip ./dist/
  docker build \
    --build-arg REPO_PY_DEPS=${REPO_PY_DEPS} \
    ${BUILD_ARG_TAG_PY_DEPS} \
    --build-arg GOCHAIN_VERSION=${GOCHAIN_VERSION} \
    --tag ${REPO_GOCHAIN} .
  rm -rf dist
else
  echo "Already exists image ${REPO_GOCHAIN}"
fi

if [ "${TAG_GOCHAIN}" != "" ] && [ "${TAG_GOCHAIN}" != "latest" ];then
  TAG_SLUG=${TAG_GOCHAIN//\//__}
  echo "Tag image ${REPO_GOCHAIN} to ${REPO_GOCHAIN}:${TAG_SLUG} for TAG_GOCHAIN:${TAG_GOCHAIN}"
  docker tag ${REPO_GOCHAIN} ${REPO_GOCHAIN}:${TAG_SLUG}
fi

cd $PRE_PWD
