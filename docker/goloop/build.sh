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

GOLOOP_VERSION=${GOLOOP_VERSION:-$(git describe --always --tags --dirty)}
REPO_GOLOOP=${REPO_GOLOOP:-goloop}
PRE_GOLOOP_VERSION=$(docker image inspect ${REPO_GOLOOP} -f "{{.Config.Labels.GOLOOP_VERSION}}" || echo "none")
if [ "${GOLOOP_VERSION}" != "${PRE_GOLOOP_VERSION}" ]
then
  echo "Build image ${REPO_GOLOOP} using ${REPO_PY_DEPS} with TAG_PY_DEPS:${TAG_PY_DEPS}"
  mkdir -p dist/pyee dist/bin
  cp ../../pyee/dist/* ./dist/pyee/
  cp ../../bin/* ./dist/bin/
  docker build \
    --build-arg REPO_PY_DEPS=${REPO_PY_DEPS} \
    ${BUILD_ARG_TAG_PY_DEPS} \
    --build-arg GOLOOP_VERSION=${GOLOOP_VERSION} \
    --tag ${REPO_GOLOOP} .
  rm -rf dist
else
  echo "Already exists image ${REPO_GOLOOP}"
fi

if [ "${TAG_GOCHAIN}" != "" ] && [ "${TAG_GOCHAIN}" != "latest" ];then
  TAG_SLUG=${TAG_GOCHAIN//\//__}
  echo "Tag image ${REPO_GOCHAIN} to ${REPO_GOCHAIN}:${TAG_SLUG} for TAG_GOCHAIN:${TAG_GOCHAIN}"
  docker tag ${REPO_GOCHAIN} ${REPO_GOCHAIN}:${TAG_SLUG}
fi

cd $PRE_PW
