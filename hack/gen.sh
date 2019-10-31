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
set -o nounset
set -o pipefail


main() {
    if [[ ! -e '/.dockerenv' ]]; then
        echo "Error: Gen can only be executed in docker." 2>&1
        exit 1
    fi

    # TODO: add options, such as clean and help
    ./hack/gen-proto.sh -f
    go generate ./...
}

main "$@"
