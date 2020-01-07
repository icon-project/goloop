#!/bin/sh
set -e

PRE_PWD=$(pwd)
WORKDIR=$(dirname "$(readlink -f ${0})")
cd $WORKDIR

REPO_PYJAVA_DEPS=${REPO_PYJAVA_DEPS:-goloop/pyjava-deps}

./update.sh "${REPO_PYJAVA_DEPS}" ../..

if [ "${TAG_PYJAVA_DEPS}" != "" ] && [ "${TAG_PYJAVA_DEPS}" != "latest" ]; then
  TAG_SLUG=${TAG_PYJAVA_DEPS//\//__}
  echo "Tag image ${REPO_PYJAVA_DEPS} to ${REPO_PYJAVA_DEPS}:${TAG_SLUG} for TAG_PYJAVA_DEPS:${TAG_PYJAVA_DEPS}"
  docker tag ${REPO_PYJAVA_DEPS} ${REPO_PYJAVA_DEPS}:${TAG_SLUG}
fi

cd $PRE_PWD
