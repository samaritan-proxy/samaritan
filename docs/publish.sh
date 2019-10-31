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

rm -rf website
git clone git@github.com:samaritan-proxy/samaritan-proxy.github.io.git website
rsync -avz docs/site/ website/docs/
cd website
git config user.name "sam-ci-bot"
git config user.email sam-ci-bot@users.noreply.github.com
git add .
git commit -m "Update docs"
git push
