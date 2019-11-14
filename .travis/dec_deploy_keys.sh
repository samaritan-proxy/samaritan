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

pushd .travis
find . -name "*_deploy_key.enc" -exec sh -c 'openssl aes-256-cbc -K $encrypted_fa8540867d09_key -iv $encrypted_fa8540867d09_iv -in $0 -out ${0%.enc} -d' {} \;
find . -name "*_deploy_key" -exec chmod 600 {} \;
popd
