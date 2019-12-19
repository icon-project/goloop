#!/bin/sh
set -e

PRE_PWD=$(pwd)
WORKDIR=$(dirname "$(readlink -f ${0})")
cd $WORKDIR

REPO_JAVA_DEPS=${REPO_JAVA_DEPS:-goloop/java-deps}

./update.sh "${REPO_JAVA_DEPS}" ../..

if [ "${TAG_JAVA_DEPS}" != "" ] && [ "${TAG_JAVA_DEPS}" != "latest" ]; then
  TAG_SLUG=${TAG_JAVA_DEPS//\//__}
  echo "Tag image ${REPO_JAVA_DEPS} to ${REPO_JAVA_DEPS}:${TAG_SLUG} for TAG_JAVA_DEPS:${TAG_JAVA_DEPS}"
  docker tag ${REPO_JAVA_DEPS} ${REPO_JAVA_DEPS}:${TAG_SLUG}
fi

cd $PRE_PWD
