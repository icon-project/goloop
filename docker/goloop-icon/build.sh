#!/bin/sh

# Copyright 2021 ICON Foundation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

PRE_PWD=$(pwd)
WORKDIR=$(dirname "$(readlink -f ${0})")
cd $WORKDIR

export IMAGE_BASE=${IMAGE_BASE:-goloop/base-all:latest}

export GOLOOP_ICON_VERSION=${GOLOOP_ICON_VERSION:-$(git describe --always --tags --dirty)}
IMAGE_GOLOOP_ICON=${IMAGE_GOLOOP_ICON:-goloop-icon:latest}

./update.sh "${IMAGE_GOLOOP_ICON}" ../..

cd $PRE_PWD
