#!/bin/sh
set -e

PRE_PWD=$(pwd)
WORKDIR=$(dirname "$(readlink -f ${0})")
cd $WORKDIR

REPO_GO_DEPS=${REPO_GO_DEPS:-goloop/go-deps}

./update.sh "${REPO_GO_DEPS}" ../..

if [ "${TAG_GO_DEPS}" != "" ] && [ "${TAG_GO_DEPS}" != "latest" ];then
  TAG_SLUG=${TAG_GO_DEPS//\//__}
  echo "Tag image ${REPO_GO_DEPS} to ${REPO_GO_DEPS}:${TAG_SLUG} for TAG_GO_DEPS:${TAG_GO_DEPS}"
  docker tag ${REPO_GO_DEPS} ${REPO_GO_DEPS}:${TAG_SLUG}
fi

cd $PRE_PWD
