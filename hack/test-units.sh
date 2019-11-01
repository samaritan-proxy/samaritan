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

# for codecov.io, refer to https://github.com/codecov/example-go
echo "" > coverage.txt

# TODO(kirk91): add more options such as GOLFAGS, TEST PACKAGE
pkgs=$(go list ./... | grep -E -v '/test/|/pb/')
for pkg in $pkgs; do
    dir=${pkg/$REPO_URI\//}
    # TODO: enable race detector
    go test -coverprofile "$dir/cover.out" -covermode=atomic "$pkg"
    if [ -f "$dir/cover.out" ]; then
        cat "$dir/cover.out" >> coverage.txt
    fi
done
