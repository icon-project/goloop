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

GOCHAIN_VERSION=${GOCHAIN_VERSION:-$(git describe --always --tags --dirty)}
REPO_GOCHAIN=${REPO_GOCHAIN:-goloop/gochain}
PRE_GOCHAIN_VERSION=$(docker image inspect ${REPO_GOCHAIN} -f "{{.Config.Labels.GOCHAIN_VERSION}}" || echo "none")
if [ "${GOCHAIN_VERSION}" != "${PRE_GOCHAIN_VERSION}" ]
then
  echo "Build image ${REPO_GOCHAIN} using ${REPO_PYJAVA_DEPS} with TAG_PYJAVA_DEPS:${TAG_PYJAVA_DEPS}"
  mkdir dist
  cp ../../pyee/dist/pyexec-*.whl ./dist/
  cp ../../bin/gochain ./dist/
  cp ../../javaee/app/exectest/build/distributions/exectest.zip ./dist/
  docker build \
    --build-arg REPO_PYJAVA_DEPS=${REPO_PYJAVA_DEPS} \
    ${BUILD_ARG_TAG_PYJAVA_DEPS} \
    --build-arg GOCHAIN_VERSION=${GOCHAIN_VERSION} \
    --tag ${REPO_GOCHAIN} .
  rm -rf dist
else
  echo "Already exists image ${REPO_GOCHAIN}"
fi

if [ "${TAG_GOCHAIN}" != "" ] && [ "${TAG_GOCHAIN}" != "latest" ]; then
  TAG_SLUG=${TAG_GOCHAIN//\//__}
  echo "Tag image ${REPO_GOCHAIN} to ${REPO_GOCHAIN}:${TAG_SLUG} for TAG_GOCHAIN:${TAG_GOCHAIN}"
  docker tag ${REPO_GOCHAIN} ${REPO_GOCHAIN}:${TAG_SLUG}
fi

cd $PRE_PWD
