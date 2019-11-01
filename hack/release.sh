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

set -o errexit
set -o pipefail
set -o nounset

readonly SUPPORTED_PLATFORMS=(
    linux/amd64
    darwin/amd64
)
readonly RELEASE_DIR=".release"
readonly DOCKER_REPO="samaritanproxy/samaritan"
readonly DOCKER_TAG=$VERSION
readonly IMAGE_NAME="$DOCKER_REPO:$DOCKER_TAG"

for platform in "${SUPPORTED_PLATFORMS[@]}"; do
    export GOOS=${platform%/*}
    export GOARCH=${platform#*/}
    export CGO_ENABLED=0

    make build

    # package tarball
    target="samaritan$VERSION.${GOOS}-${GOARCH}"
    mkdir -p "$RELEASE_DIR/$target"
    cp ./bin/"$GOOS-$GOARCH"/* "$RELEASE_DIR/$target"

    pushd $RELEASE_DIR >/dev/null
    tar zcf "$target.tar.gz" "$target"
    echo "Wrote $RELEASE_DIR/$target.tar.gz"
    popd >/dev/null

    # build image, only support linux/amd64 currently
    if [ "$GOOS" == "linux" ] && [ "$GOARCH" == "amd64" ]; then
        docker build -t "$IMAGE_NAME" -f Dockerfile "./bin/$GOOS-$GOARCH"
        echo "Built image $IMAGE_NAME"
    fi
done
