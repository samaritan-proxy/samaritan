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

force=0
readonly path=./pb
readonly doc_path=./docs/src
readonly protoc_gen_gogo_repo="github.com/gogo/protobuf"
readonly protoc_gen_validate_repo="github.com/envoyproxy/protoc-gen-validate"

gen_proto() {
    pb_file=$1

    if no_need_gen "${pb_file}"; then
        return 0
    fi

    protoc \
        -I ${path} \
        -I "$GOPATH/src/$protoc_gen_gogo_repo" \
        -I "$GOPATH/src/$protoc_gen_validate_repo" \
        --gogofast_out=\
Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/duration.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/struct.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/wrappers.proto=github.com/gogo/protobuf/types,\
plugins=grpc:${GOPATH}/src \
        --validate_out="lang=gogo:${GOPATH}/src" \
        --jsonify_out="jsonpb=github.com/gogo/protobuf/jsonpb:${GOPATH}/src" \
        "${pb_file}"
}

gen_doc() {
    _path=$1
    mkdir -p ${doc_path}
    protoc \
        --plugin=protoc-gen-doc=${GOPATH}/bin/protoc-gen-doc \
        -I ${path} \
        -I "$GOPATH/src/$protoc_gen_gogo_repo" \
        -I "$GOPATH/src/$protoc_gen_validate_repo" \
        --doc_out=${doc_path} \
        --doc_opt=markdown,proto-ref.md \
        `find ${_path} -name '*.proto'`
}

modify_time() {
    stat -c %Y "$1"
}

# return 1 if need run protoc
no_need_gen() {
    if [[ ${force} == 1 ]]; then
        return 1
    fi

    pb_file=$1
    go_file=${pb_file/%.proto/.pb.go}
    if [[ ! -e ${go_file} ]]; then
        return 1
    fi
    if [[ $(modify_time "${go_file}") < $(modify_time "${pb_file}") ]]; then
        return 1
    fi
    return 0
}

walk() {
    _path=$1
    for file in $(find ${_path} -name '*.proto' -not -path "*/vendor/*"); do
        gen_proto "${file}"
    done
}

do_clean() {
    find ./ -name '*.pb.*' -not -path "*/vendor/*" -delete
    rm -rf ${doc_path}/proto-ref.md
}

usage() {
    echo "Usage: gen-proto.sh [ -h | -c | -f ]"
    echo ""
    echo "Options:"
    echo "  -h      : this help"
    echo "  -c      : clean all generated files"
    echo "  -f      : regenerated all files forcely"
}

main() {
    while [[ $# > 0 ]]; do
        key="$1"
        case ${key} in
        -c | --clean)
            do_clean "${path}"
            exit
            ;;
        -f | --force)
            force=1
            ;;
        -h | --help)
            usage
            exit
            ;;
        *)
            usage
            exit 1
            ;;
        esac
        shift
    done

    if [[ ! -e '/.dockerenv' ]]; then
        echo "Error: Gen can only be executed in docker." 2>&1 && exit 1
    fi

    oldifs=$IFS
    IFS=$'\n'
    walk "${path}"
    gen_doc "${path}"
    IFS=$oldifs
}

main $@
