#!/bin/sh
set -e

PRE_PWD=$(pwd)
WORKDIR=$(dirname "$(readlink -f ${0})")
cd $WORKDIR

PYTHON_VERSION=${PYTHON_VERSION:-3.7.5}
SHASUM=$(cat ../../pyee/requirements.txt \
             ../../docker/py-deps/Dockerfile \
         | sha1sum | cut -d ' ' -f 1)
PYDEP_SHA=${PYTHON_VERSION}-${SHASUM}
REPO_PY_DEPS=${REPO_PY_DEPS:-goloop/py-deps}
TAG_PY_DEPS=${TAG_PY_DEPS:-$(docker images --filter="reference=$REPO_PY_DEPS" --filter="label=GOLOOP_PYDEP_SHA=${PYDEP_SHA}" --format="{{.Tag}}" | head -n 1)}
if [ "${TAG_PY_DEPS}" != "" ]; then
  TAG_SLUG=${TAG_PY_DEPS//\//__}
  BUILD_ARG_TAG_PY_DEPS="--build-arg=TAG_PY_DEPS=${TAG_SLUG}"
fi

GOLOOP_VERSION=${GOLOOP_VERSION:-$(git describe --always --tags --dirty)}
REPO_GOLOOP=${REPO_GOLOOP:-goloop}
PRE_GOLOOP_VERSION=$(docker image inspect ${REPO_GOLOOP} -f "{{.Config.Labels.GOLOOP_VERSION}}" || echo "none")
if [ "${GOLOOP_VERSION}" != "${PRE_GOLOOP_VERSION}" ]
then
  echo "Build image ${REPO_GOLOOP} using ${REPO_PY_DEPS} with TAG_PY_DEPS:${TAG_PY_DEPS}"
  JAVAEE_VERSION=$(grep "^VERSION=" ../../javaee/gradle.properties | cut -d= -f2)
  mkdir -p dist/pyee dist/bin
  cp ../../pyee/dist/* ./dist/pyee/
  cp ../../bin/* ./dist/bin/
  cp ../../javaee/app/execman/build/distributions/execman-*.zip ./dist/
  docker build \
    --build-arg REPO_PY_DEPS=${REPO_PY_DEPS} \
    ${BUILD_ARG_TAG_PY_DEPS} \
    --build-arg GOLOOP_VERSION=${GOLOOP_VERSION} \
    --build-arg JAVAEE_VERSION=${JAVAEE_VERSION} \
    --tag ${REPO_GOLOOP} .
  rm -rf dist
else
  echo "Already exists image ${REPO_GOLOOP}"
fi

if [ "${TAG_GOLOOP}" != "" ] && [ "${TAG_GOLOOP}" != "latest" ]; then
  TAG_SLUG=${TAG_GOLOOP//\//__}
  echo "Tag image ${REPO_GOLOOP} to ${REPO_GOLOOP}:${TAG_SLUG} for TAG_GOLOOP:${TAG_GOLOOP}"
  docker tag ${REPO_GOLOOP} ${REPO_GOLOOP}:${TAG_SLUG}
fi

cd $PRE_PWD
