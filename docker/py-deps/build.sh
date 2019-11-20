#!/bin/sh
set -e

PRE_PWD=$(pwd)
WORKDIR=$(dirname "$(readlink -f ${0})")
cd $WORKDIR

REPO_PY_DEPS=${REPO_PY_DEPS:-goloop/py-deps}

./update.sh "${REPO_PY_DEPS}" ../..

if [ "${TAG_PY_DEPS}" != "" ] && [ "${TAG_PY_DEPS}" != "latest" ];then
  TAG_SLUG=${TAG_PY_DEPS//\//__}
  echo "Tag image ${REPO_PY_DEPS} to ${REPO_PY_DEPS}:${TAG_SLUG} for TAG_PY_DEPS:${TAG_PY_DEPS}"
  docker tag ${REPO_PY_DEPS} ${REPO_PY_DEPS}:${TAG_SLUG}
fi

cd $PRE_PWD
