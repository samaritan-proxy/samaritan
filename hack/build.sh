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

#!/bin/bash

set -o errexit
set -o pipefail
set -o nounset

build() {
    now=$(date '+%Y-%m-%d_%H:%M:%S')
    rev=${rev:-$(git rev-parse HEAD)}
    ld_flags="-X ${REPO_URI}/consts.Build=${now}@${rev}"

    target=$1
    bin_name=$1
    output_dir="bin/$GOOS-$GOARCH"
    output="${output_dir}/${bin_name}"
    go build \
        -gcflags=${GOFLAGS:-""} \
        -o "$output" \
        -ldflags="${ld_flags}" \
        "./cmd/${target}/"

    # link bin/${bin_name} to the binary when not compiling across platforms
    hostos=$(go env GOHOSTOS)
    hostarch=$(go env GOHOSTARCH)
    if [ "$GOOS" == "$hostos" ] && [ "$GOARCH" == "$hostarch" ]; then
        cp "$output" "bin/${bin_name}"
    fi
}

main() {
    build $1
}

main $@
