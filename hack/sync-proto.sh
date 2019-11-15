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
readonly GO_SOURCE_DIR="pb"
readonly GO_TARGET_DIR="$DIR/go"

# clone repo, copy the latest proto and go generated files.
rm -rf $DIR
git clone $GIT_REPO $DIR
rsync -avz --include "*.proto" --include="*/" --exclude "*" $PROTO_SOURCE_DIR/ $PROTO_TARGET_DIR/
rsync -avz --include "*.go" --include="*/" --exclude "*" $GO_SOURCE_DIR/ $GO_TARGET_DIR/
cp -f go.mod $GO_TARGET_DIR

# fix go_package option in proto files.
find $PROTO_TARGET_DIR -name "*.proto" -exec sed -i 's|samaritan-proxy/samaritan/pb|samaritan-proxy/samaritan-api/go|g' {} \;
# fix pakcage import path in go files.
find $GO_TARGET_DIR -name "*.go" -exec sed -i 's|samaritan-proxy/samaritan/pb|samaritan-proxy/samaritan-api/go|g' {} \;
# fix go module name
sed -i 's|samaritan-proxy/samaritan|samaritan-proxy/samaritan-api/go|g' $GO_TARGET_DIR/go.mod
pushd $GO_TARGET_DIR && go mod tidy && popd

# push to git repo
pushd $DIR
git config user.name "sam-ci-bot"
git config user.email sam-ci-bot@users.noreply.github.com
diff=$(git --no-pager diff)
if [ -z "$diff" ]; then
    echo "No files change"
    exit 0
fi
git add .
git commit -m "Update proto definitions"
git push
