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

.EXPORT_ALL_VARIABLES:

REPO_URI = github.com/samaritan-proxy/samaritan
BUILD_IN_DOCKER ?= 0

GO_VERSION ?= 1.13
GOOS ?= $(shell go env GOHOSTOS)
GOARCH ?= $(shell go env GOHOSTARCH)
GOFLAGS ?=
GOPROXY ?= $(shell go env GOPROXY)

.PHONY: build, all
build all:
	@for target in $(notdir $(abspath $(wildcard cmd/*/))); do \
		make $$target; \
	done

# Add rules for all directories in cmd/
#
# Example:
#	make samaritan
.PHONY: $(notdir $(abspath $(wildcard cmd/*/)))
ifeq ($(BUILD_IN_DOCKER), 1)
$(notdir $(abspath $(wildcard cmd/*/))):
	$(info building $@ (GOOS: $(GOOS), GOARCH: $(GOARCH), GO_VERSION: $(GO_VERSION), BUILD_IN_DOCKER: $(BUILD_IN_DOCKER)))
	@docker run --rm \
		-e GOOS=$(GOOS) \
		-e GOARCH=$(GOARCH) \
		-e CGO_ENABLED=0 \
		-e GOFLAGS=$(GOFLAGS) \
		-e GOPROXY=$(GOPROXY) \
		-e REPO_URI=$(REPO_URI) \
		--mount type=bind,source=`pwd`,destination=/samaritan \
		-w '/samaritan' \
		golang:$(GO_VERSION) \
		./hack/build.sh $@
else
local_go_version := $(shell go version | cut -d' ' -f3 | sed -e 's/go//g')
$(notdir $(abspath $(wildcard cmd/*/))):
	$(info building $@ (GOOS: $(GOOS), GOARCH: $(GOARCH), GO_VERSION: $(local_go_version), BUILD_IN_DOCKER: $(BUILD_IN_DOCKER)))
	@./hack/build.sh $@
endif


.PHONY: docs
docs:
	@./docs/build.sh

.PHONY: pub-docs
pub-docs: docs
	@./docs/publish.sh


.PHONY: dev-image
dev-image:
	docker build -t samaritanproxy/sam-dev -f ./Dockerfile_dev .

.PHONY: pub-dev-image
pub-dev-image:
	docker push samaritanproxy/sam-dev

.PHONY: generate
generate:
	docker run --rm \
		-e CGO_ENABLED=0 \
		-e GOPROXY=$(GOPROXY) \
		-e REPO_URI=$(REPO_URI) \
		--mount type=bind,source=`pwd`,destination=/go/src/$(REPO_URI) \
		samaritanproxy/sam-dev \
		./hack/gen.sh

.PHONY: clean-proto
clean-proto:
	@./hack/gen-proto.sh -c


.PHONY: test
test:
	@./hack/test-units.sh

.PHONY: integration-test
integration-test:
	@./hack/test-integration.sh

.PHONY: ci
ci: build test integration-test


.PHONY: release
release:
	./hack/release.sh


.PHONY: clean
clean:
	@rm -rf bin
	@rm -rf docs/site
	@rm -rf .release
	@find . -name "cover.out" -exec rm {} \;
