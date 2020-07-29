SHELL=/bin/sh
BINARY ?= eci
PACKAGE_NAME?=github.com/lf-edge/edge-containers
GIT_VERSION?=$(shell git log -1 --format="%h")
VERSION?=$(GIT_VERSION)
RELEASE_TAG ?= $(shell git tag --points-at HEAD)
ifneq (,$(RELEASE_TAG))
VERSION=$(RELEASE_TAG)-$(VERSION)
endif
GO_FILES := $(shell find . -type f -not -path './vendor/*' -name '*.go')
FROMTAG ?= latest
LDFLAGS ?= -ldflags '-extldflags "-static" -X "$(PACKAGE_NAME)/pkg/version.VERSION=$(VERSION)"'

# BUILDARCH is the host architecture
# ARCH is the target architecture
# we need to keep track of them separately
BUILDARCH ?= $(shell uname -m)
BUILDOS ?= $(shell uname -s | tr A-Z a-z)

# canonicalized names for host architecture
ifeq ($(BUILDARCH),aarch64)
BUILDARCH=arm64
endif
ifeq ($(BUILDARCH),x86_64)
BUILDARCH=amd64
endif

# unless otherwise set, I am building for my own architecture, i.e. not cross-compiling
ARCH ?= $(BUILDARCH)
OS ?= $(BUILDOS)

# canonicalized names for target architecture
ifeq ($(ARCH),aarch64)
        override ARCH=arm64
endif
ifeq ($(ARCH),x86_64)
    override ARCH=amd64
endif

# these macros create a list of valid architectures for pushing manifests
space :=
space +=
comma := ,
prefix_linux = $(addprefix linux/,$(strip $1))
join_platforms = $(subst $(space),$(comma),$(call prefix_linux,$(strip $1)))

export GO111MODULE=on
DIST_DIR=./dist/bin
DIST_BINARY = $(DIST_DIR)/$(BINARY)-$(OS)-$(ARCH)
BUILD_CMD = CGO_ENABLED=0 GOOS=$(OS) GOARCH=$(ARCH)
ifdef DOCKERBUILD
BUILD_CMD = docker run --rm \
                -e GOARCH=$(ARCH) \
                -e GOOS=$(OS) \
                -e CGO_ENABLED=0 \
                -v $(CURDIR):/go/src/$(PACKAGE_NAME) \
                -w /go/src/$(PACKAGE_NAME) \
		$(BUILDER_IMAGE)
endif

GOBIN ?= $(shell go env GOPATH)/bin
LINTER ?= $(GOBIN)/golangci-lint
MANIFEST_TOOL ?= $(GOBIN)/manifest-tool

pkgs:
ifndef PKG_LIST
	$(eval PKG_LIST := $(shell $(BUILD_CMD) go list ./... | grep -v vendor))
endif

.PHONY: fmt fmt-check lint test vet golint tag version

$(DIST_DIR):
	mkdir -p $@

## report the git tag that would be used for the images
tag:
	@echo $(GIT_VERSION)

## report the version that would be put in the binary
version:
	@echo $(VERSION)


## Check the file format
fmt-check: 
	@if [ -n "$(shell $(BUILD_CMD) gofmt -l ${GO_FILES})" ]; then \
	  $(BUILD_CMD) gofmt -s -e -d ${GO_FILES}; \
	  exit 1; \
	fi

## format files
fmt:
	gofmt -w -s $(GO_FILES)

golangci-lint: $(LINTER)
$(LINTER):
	go get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.17.1

golint:
ifeq (, $(shell which golint))
	go get -u golang.org/x/lint/golint
endif

## Lint the files
lint: pkgs golint golangci-lint
	@$(BUILD_CMD) $(LINTER) run --disable-all --enable=golint ./ pkg/... cmd/...

## Run unittests
test: pkgs
	@$(BUILD_CMD) go test -short ${PKG_LIST}

## Vet the files
vet: pkgs
	@$(BUILD_CMD) go vet ${PKG_LIST}

## Read about data race https://golang.org/doc/articles/race_detector.html
## to not test file for race use `// +build !race` at top
## Run data race detector
race: pkgs
	@$(BUILD_CMD) go test -race -short ${PKG_LIST}

## Display this help screen
help: 
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: build build-all image push deploy ci cd dep manifest-tool clean

## Build the binaries for all supported ARCH
build-all: $(addprefix sub-build-, $(ARCHES))
sub-build-%:
	@$(MAKE) ARCH=$* build

## Build the binary for a single ARCH
build: $(DIST_BINARY)
$(DIST_BINARY): $(DIST_DIR)
	$(BUILD_CMD) go build -v -o $@ $(LDFLAGS) $(PACKAGE_NAME)

## copy a binary to an install destination
install:
ifneq (,$(DESTDIR))
	mkdir -p $(DESTDIR)
	cp $(DIST_BINARY) $(DESTDIR)/$(shell basename $(DIST_BINARY))
endif

clean:
	rm -rf $(DIST_DIR)


###############################################################################
# CI/CD
###############################################################################
.PHONY: ci cd build deploy push release confirm pull-images
## Run what CI runs
# race has an issue with alpine, see https://github.com/golang/go/issues/14481
ci: build-all fmt-check lint test vet # race

