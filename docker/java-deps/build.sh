#!/bin/sh
set -e

PRE_PWD=$(pwd)
WORKDIR=$(dirname "$(readlink -f ${0})")
cd $WORKDIR

IMAGE_JAVA_DEPS=${IMAGE_JAVA_DEPS:-goloop/java-deps:latest}

./update.sh "${IMAGE_JAVA_DEPS}" ../..

cd $PRE_PWD
