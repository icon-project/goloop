#!/bin/sh
set -e

PRE_PWD=$(pwd)
WORKDIR=$(dirname "$(readlink -f ${0})")
cd $WORKDIR

PYTHON_VERSION=${PYTHON_VERSION:-3.7.5}
JAVA_VERSION=${JAVA_VERSION:-11.0.4}
SHASUM=$(cat ../../pyee/requirements.txt \
             ../../docker/pyjava-deps/Dockerfile \
         | sha1sum | cut -d ' ' -f 1)
PYJADEP_SHA=${PYTHON_VERSION}-${JAVA_VERSION}-${SHASUM}
REPO_PYJAVA_DEPS=${REPO_PYJAVA_DEPS:-goloop/pyjava-deps}
TAG_PYJAVA_DEPS=${TAG_PYJAVA_DEPS:-$(docker images --filter="reference=$REPO_PYJAVA_DEPS" --filter="label=GOLOOP_PYJADEP_SHA=${PYJADEP_SHA}" --format="{{.Tag}}" | head -n 1)}
if [ "${TAG_PYJAVA_DEPS}" != "" ]; then
  TAG_SLUG=${TAG_PYJAVA_DEPS//\//__}
  BUILD_ARG_TAG_PYJAVA_DEPS="--build-arg=TAG_PYJAVA_DEPS=${TAG_SLUG}"
fi

GOLOOP_VERSION=${GOLOOP_VERSION:-$(git describe --always --tags --dirty)}
REPO_GOLOOP=${REPO_GOLOOP:-goloop}
PRE_GOLOOP_VERSION=$(docker image inspect ${REPO_GOLOOP} -f "{{.Config.Labels.GOLOOP_VERSION}}" || echo "none")
if [ "${GOLOOP_VERSION}" != "${PRE_GOLOOP_VERSION}" ]
then
  echo "Build image ${REPO_GOLOOP} using ${REPO_PYJAVA_DEPS} with TAG_PYJAVA_DEPS:${TAG_PYJAVA_DEPS}"
  mkdir -p dist/pyee dist/bin
  cp ../../pyee/dist/* ./dist/pyee/
  cp ../../bin/* ./dist/bin/
  cp ../../javaee/app/exectest/build/distributions/exectest.zip ./dist/
  docker build \
    --build-arg REPO_PYJAVA_DEPS=${REPO_PYJAVA_DEPS} \
    ${BUILD_ARG_TAG_PYJAVA_DEPS} \
    --build-arg GOLOOP_VERSION=${GOLOOP_VERSION} \
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
