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
# Usage: ./hack/bump-version.sh 1.1.0 [anything is ok if you want to commit your own message with the tag]

set -o errexit
set -o nounset
set -o pipefail

readonly GIT_REMOTE="git@github.com:samaritan-proxy/samaritan.git"

sam_version_file_path='./consts/consts.go'

get_sam_version() {
    echo `sed -n '/Version = ".*"/p' ${sam_version_file_path} | awk -F '"' '{print $2}'`
}

modify_sam_version() {
    local new_version=$1
    sed -i '' 's/Version = ".*"/Version = "'$new_version'"/g' ${sam_version_file_path}
}

save_latest_commit() {
    local new_version=$1
    git add $sam_version_file_path
    git commit -m "bump version to $new_version"
}

save_latest_update() {
    local version=$1
    git add $sam_version_file_path
    git commit -m "bump version to $version"
    local tag=v$version
    local tag_msg=$2
    git tag -a $tag -m "$tag_msg"
}

push_update_to_upstream() {
    current_branch=`git rev-parse --abbrev-ref HEAD`
    git push ${GIT_REMOTE} $current_branch --follow-tags
}

# Test case here: https://regex101.com/r/8ZWg7c/1
parse_semver() {
    local RE="^(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)(\\-[0-9A-Za-z-]+(\\.[0-9A-Za-z-]+)*)?(\\+[0-9A-Za-z-]+(\\.[0-9A-Za-z-]+)*)?$"
    #MAJOR
    eval $2=""
    #MINOR
    eval $3=""
    #PATCH
    eval $4=""
    #PRERELEASE
    eval $5=""
    #BUILDMETADATA
    eval $6=""
    if [[ $1 =~ $RE ]]; then
        eval $2=${BASH_REMATCH[1]}
        eval $3=${BASH_REMATCH[2]}
        eval $4=${BASH_REMATCH[3]}
        eval $5=${BASH_REMATCH[4]}
        eval $6=${BASH_REMATCH[6]}
    fi
}

parse_semver_to_list() {
    local ver=$1
    parse_semver $ver MAJOR MINOR PATCH PRERELEASE BUILDMETADATA
    eval "$2=(\"$MAJOR\" \"$MINOR\" \"$PATCH\" \"$PRERELEASE\" \"$BUILDMETADATA\")"
}

gen_semver() {
    local MAJOR=$1
    local MINOR=$2
    local PATCH=$3
    local PRERELEASE=$4
    local BUILDMETADATA=$5
    local version=$MAJOR.$MINOR.$PATCH
    if [[ ! -z "${PRERELEASE// }" ]]; then
        version=$version$PRERELEASE
    fi
    if [[ ! -z "${BUILDMETADATA// }" ]]; then
        version=$version$BUILDMETADATA
    fi
    eval $6=$version
}

validate_version() {
    local version=$1
    parse_semver $version MAJOR MINOR PATCH PRERELEASE BUILDMETADATA
    if [[ -z $MAJOR ]] || [[ -z $MINOR ]] || [[ -z $PATCH ]]; then
        eval $2=1
        return
    fi
    gen_semver $MAJOR $MINOR $PATCH ${PRERELEASE:-""} ${BUILDMETADATA:-""} parsed_version
    if [ $parsed_version != $version ]; then
        eval $2=1
    else
        eval $2=0
    fi
}

is_prerelease_version() {
    local version=$1
    parse_semver $version MAJOR MINOR PATCH PRERELEASE BUILDMETADATA
    if [ ! -z "$PRERELEASE" ]; then
        eval $2=1
    else
        eval $2=0
    fi
}

compare_version() {
    parse_semver_to_list $1 old_ver
    parse_semver_to_list $2 new_ver
    for i in 0 1 2; do
        local diff=$((${old_ver[$i]} - ${new_ver[$i]}))
        if [[ $diff -lt 0 ]]; then
            eval $3=-1; return 0
        elif [[ $diff -gt 0 ]]; then
            eval $3=1; return 0
        fi
    done

    # PREREL should compare with the ASCII order.
    if [[ -z "${old_ver[3]}" ]] && [[ -n "${new_ver[3]}" ]]; then
        eval $3=1; return 0;
    elif [[ -n "${old_ver[3]}" ]] && [[ -z "${new_ver[3]}" ]]; then
        eval $3=-1; return 0;
    elif [[ -n "${old_ver[3]}" ]] && [[ -n "${new_ver[3]}" ]]; then
        if [[ "${old_ver[3]}" > "${new_ver[3]}" ]]; then
        eval $3=1; return 0;
        elif [[ "${old_ver[3]}" < "${new_ver[3]}" ]]; then
        eval $3=-1; return 0;
        fi
    fi

    eval $3=0
}

bump_version() {
    local version=$1
    parse_semver $version MAJOR MINOR PATCH PRERELEASE BUILDMETADATA
    PATCH=$(($PATCH+1))
    eval $2=$MAJOR.$MINOR.$PATCH
}

main(){
    # check user input
    read -t 10 -p "Enter 12345 to bump version and press [ENTER]: " password
    if [ "$password" != "12345" ]; then
        echo -e "Input isn't equal to 12345. Bye..."
        exit 1
    fi

    version=${1:-"empty"}
    validate_version $version is_semver
    if [ $is_semver -eq 1 ]; then
        echo $version' is not semantic version. Bye...'
        exit 1
    fi

    old_ver=$( get_sam_version )
    compare_version $old_ver $version newer_or_not
    if [ $newer_or_not -gt -1 ];then
        echo "$version is not newer than current version"
        exit 1
    fi

    modify_sam_version $version
    echo "Start push version $version & tag v$version to upstream"
    msg=${2:-""}
    if [ ! -z "$msg" ]; then
        read -t 10 -p "Enter the tag message and press [ENTER]: " msg
    fi
    save_latest_update $version "${msg:-"bump version to $version"}"
    echo 'Bump version success'

    # exit for pre-release version
    is_prerelease_version $version is_prerelease
    if [ $is_prerelease -eq 1 ]; then
        push_update_to_upstream
        echo 'Bye...'
        exit 0
    fi

    # modify sam version to $new_version_dev
    bump_version $version new_version
    new_version_dev=${new_version}-dev
    echo "Start push version $new_version_dev to upstream"
    modify_sam_version $new_version_dev
    save_latest_commit $new_version_dev
    push_update_to_upstream
    echo 'Push dev version success'
    echo 'Bye...'
}

main "$@"
