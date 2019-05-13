#!/bin/sh
set -e

PRE_PWD=$(pwd)
WORKDIR=$(dirname "$(readlink -f ${0})")
cd $WORKDIR
PYTHON_VERSION=${PYTHON_VERSION:-3.7.3}
PYREQ_SHA=$(sha1sum ../../pyee/requirements.txt | cut -d ' ' -f 1)
PYREQ_SHA_SHORT=${PYREQ_SHA:0:12}
REPO_PY_DEPS=${REPO_PY_DEPS:-goloop/py-deps}
TAG_PY_DEPS=${TAG_PY_DEPS:-${PYREQ_SHA_SHORT}}
#$(docker image inspect ${REPO_PY_DEPS}:${PYREQ_SHA} &> /dev/null;echo $?)
PRE_PYREQ_SHA=$(docker image inspect -f '{{.Config.Labels.GOLOOP_PYREQ_SHA}}' ${REPO_PY_DEPS}:${TAG_PY_DEPS} || echo "none")
if [ "${PYREQ_SHA}" != "${PRE_PYREQ_SHA}" ]
then
  echo "Build image ${REPO_PY_DEPS}:${TAG_PY_DEPS} for ${PYREQ_SHA}"
  cp ../../pyee/requirements.txt ./
  docker build --build-arg GOLOOP_PYREQ_SHA=${PYREQ_SHA} --build-arg PYTHON_VERSION=${PYTHON_VERSION} --tag ${REPO_PY_DEPS}:${TAG_PY_DEPS} --tag ${REPO_PY_DEPS} .
  rm -f requirements.txt
else
  echo "Already exists image ${REPO_PY_DEPS}:${TAG_PY_DEPS} for ${PYREQ_SHA}"
fi
cd $PRE_PW
