#!/bin/sh
set -e

PRE_PWD=$(pwd)
WORKDIR=$(dirname "$(readlink -f ${0})")
cd $WORKDIR

GOLANG_VERSION=${GOLANG_VERSION:-1.12.4}
GOMOD_SHA=$(sha1sum ../../go.sum | cut -d ' ' -f 1)
GOMOD_SHA_SHORT=${GOMOD_SHA:0:12}
REPO_GO_DEPS=${REPO_GO_DEPS:-goloop/go-deps}
PRE_GOMOD_SHA=$(docker image inspect -f '{{.Config.Labels.GOLOOP_GOMOD_SHA}}' ${REPO_GO_DEPS} || echo "none")
if [ "${GOMOD_SHA}" != "${PRE_GOMOD_SHA}" ]
then
  echo "Build image ${REPO_GO_DEPS} for ${GOMOD_SHA}"
  cp ../../go.mod ../../go.sum ./
  docker build \
    --build-arg GOLOOP_GOMOD_SHA=${GOMOD_SHA} \
    --build-arg GOLANG_VERSION=${GOLANG_VERSION} \
    --tag ${REPO_GO_DEPS} .
  rm -f go.mod go.sum
else
  echo "Already exists image ${REPO_GO_DEPS} for ${GOMOD_SHA}"
fi

if [ "${TAG_GO_DEPS}" != "" ] && [ "${TAG_GO_DEPS}" != "latest" ];then
  TAG_SLUG=${TAG_GO_DEPS//\//__}
  echo "Tag image ${REPO_GO_DEPS} to ${REPO_GO_DEPS}:${TAG_SLUG} for TAG_GO_DEPS:${TAG_GO_DEPS}"
  docker tag ${REPO_GO_DEPS} ${REPO_GO_DEPS}:${TAG_SLUG}
fi

cd $PRE_PWD
