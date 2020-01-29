#!/bin/sh
set -e

PRE_PWD=$(pwd)
WORKDIR=$(dirname "$(readlink -f ${0})")
cd $WORKDIR

GOLOOP_VERSION=${GOLOOP_VERSION:-$(git describe --always --tags --dirty)}
REPO_GOLOOP_JAVA=${REPO_GOLOOP_JAVA:-goloop-java}
PRE_GOLOOP_VERSION=$(docker image inspect ${REPO_GOLOOP_JAVA} -f "{{.Config.Labels.GOLOOP_VERSION}}" || echo "none")
if [ "${GOLOOP_VERSION}" != "${PRE_GOLOOP_VERSION}" ]
then
  echo "Build image ${REPO_GOLOOP_JAVA} for ${GOLOOP_VERSION}"
  mkdir -p dist/bin
  cp ../../bin/* ./dist/bin/
  cp ../../javaee/app/execman/build/distributions/execman.zip ./dist/
  docker build \
    --build-arg GOLOOP_VERSION=${GOLOOP_VERSION} \
    --tag ${REPO_GOLOOP_JAVA} .
  rm -rf dist
else
  echo "Already exists image ${REPO_GOLOOP_JAVA}:${GOLOOP_VERSION}"
fi

cd $PRE_PWD
