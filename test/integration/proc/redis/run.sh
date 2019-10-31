#!/bin/bash -eu
#
# Copyright 2019 Samaritan Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

readonly CUR_DIR=$(dirname "$0")
readonly DOCKER_REPO="samaritanproxy/sam-test"
readonly DOCKER_TAG="integration-$(basename $CUR_DIR)"

docker build -t "${DOCKER_REPO}:${DOCKER_TAG}" -f "${CUR_DIR}/Dockerfile" .
cmd="go test ${GOTEST_FLAGS:-""} ${CUR_DIR}"
docker run --rm \
    -e GOPROXY="${GOPROXY:-}" \
    -e GOFLAGS="${GOFLAGS:-}" \
    "${DOCKER_REPO}:${DOCKER_TAG}" bash -c "$cmd"
