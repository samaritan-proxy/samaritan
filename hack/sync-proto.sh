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

readonly GIT_REPO="git@github.com:samaritan-proxy/samaritan-api"
readonly DIR=".ci/samaritan-api"
readonly PROTO_SOURCE_DIR="pb"
readonly PROTO_TARGET_DIR="$DIR/proto"

# clone repo and copy the latest proto files
rm -rf $DIR
git clone $GIT_REPO $DIR
rsync -avz --exclude "*.go" $PROTO_SOURCE_DIR/ $PROTO_TARGET_DIR/
pushd $DIR

# remove customizes options
find . -name "*.proto" -exec sed -i '/option go_package =.*/d' {} \;

# push to git repo
git config user.name "sam-ci-bot"
git config user.email sam-ci-bot@users.noreply.github.com
git add .
git commit -m "Update proto definitions"
git push
