#!/bin/sh
set -e

PRE_PWD=$(pwd)
WORKDIR=$(dirname "$(readlink -f ${0})")
cd $WORKDIR

#refer from docker/py-deps/build.sh
PYREQ_SHA=$(sha1sum ../../pyee/requirements.txt | cut -d ' ' -f 1)
PYREQ_SHA_SHORT=${PYREQ_SHA:0:12}
REPO_PY_DEPS=${REPO_PY_DEPS:-goloop/py-deps}
TAG_PY_DEPS=${TAG_PY_DEPS:-${PYREQ_SHA_SHORT}}

REPO_GOCHAIN=${REPO_GOCHAIN:-goloop/gochain}
TAG_GOCHAIN=${TAG_GOCHAIN:-$(git describe --always --tags --dirty)}
if [ "$(docker image inspect ${REPO_GOCHAIN}:${TAG_GOCHAIN} &> /dev/null;echo $?)" != "0" ]
then
  echo "Build image ${REPO_GOCHAIN}:${TAG_GOCHAIN} using ${REPO_PY_DEPS}:${TAG_PY_DEPS}"
  mkdir dist
  cp ../../pyee/dist/pyexec-*.whl ./dist/
  cp ../../bin/gochain ./dist/
  #require update 'tar' for '--transform' argument
  #tar -C ../../testsuite --transform='flags=r;s|data|testsuite|' -z -cvf dist/testsuite.tar.gz data
  cp -r ../../testsuite/data ./testsuite
  tar -cvzf dist/testsuite.tar.gz testsuite
  rm -rf testsuite
  docker build \
    --build-arg REPO_PY_DEPS=${REPO_PY_DEPS} \
    --build-arg TAG_PY_DEPS=${TAG_PY_DEPS} \
    --build-arg GOCHAIN_VERSION=${TAG_GOCHAIN} \
    --tag ${REPO_GOCHAIN}:${TAG_GOCHAIN} --tag ${REPO_GOCHAIN} .
  rm -rf dist
else
  echo "Already exists image ${REPO_GOCHAIN}:${TAG_GOCHAIN}"
fi
cd $PRE_PW
