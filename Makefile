RUSTFLAGS="-C target-feature=+crt-static"

# Detect OS
OS := $(shell uname | tr "[:upper:]" "[:lower:]")
ARCH := $(shell uname -m | tr "[:upper:]" "[:lower:]")
GOPATH ?= $(shell go env GOPATH)
GOFLAGS ?= $(GOFLAGS:)
GO=go
GO_MAJOR_VERSION = $(shell $(GO) version | cut -c 14- | cut -d' ' -f1 | cut -d'.' -f1)
GO_MINOR_VERSION = $(shell $(GO) version | cut -c 14- | cut -d' ' -f1 | cut -d'.' -f2)
GO_OS = $(shell $(GO) version | cut -c 14- | cut -d' ' -f2 | cut -d'/' -f1 | tr "[:upper:]" "[:lower:]")
GO_ARCH = $(shell $(GO) version | cut -c 14- | cut -d' ' -f2 | cut -d'/' -f2 | tr "[:upper:]" "[:lower:]")

IPFS_FUSE_IMAGE ?= "binocarlos/bacalhau-ipfs-sidecar-image"
IPFS_FUSE_TAG ?= "v1"

ifeq ($(GO_ARCH), x86_64)
GO_ARCH = "amd64"
endif

# Assuming match
MISMATCH=

# use docker runtime rather than ignite, meaning we run basically everywhere (no need for hardware virtualization support)
export BACALHAU_RUNTIME = docker

define GO_MISMATCH_ERROR

Your go binary does not match your architecture.
	Go binary:    $(GO_OS) - $(GO_ARCH)
	Environment:  $(OS) - $(ARCH)
	GOPATH:       $(GOPATH)

endef
export GO_MISMATCH_ERROR

# Env Variables
export GO111MODULE = on
export GO = go
export PYTHON = python3
export PRECOMMIT = poetry run pre-commit

BUILD_DIR = bacalhau

TAG ?= $(eval TAG := $(shell git describe --tags --always))$(TAG)
REPO ?= $(shell echo $$(cd ../${BUILD_DIR} && git config --get remote.origin.url) | sed 's/git@\(.*\):\(.*\).git$$/https:\/\/\1\/\2/')
BRANCH ?= $(shell cd ../${BUILD_DIR} && git branch | grep '^*' | awk '{print $$2}')

# Temp dirs
TMPRELEASEWORKINGDIR := $(shell mktemp -d -t bacalhau-release-dir.XXXXXXX)
TMPARTIFACTDIR := $(shell mktemp -d -t bacalhau-artifact-dir.XXXXXXX)
PACKAGE := $(shell echo "bacalhau_$(TAG)_${GO_OS}_$(GO_ARCH)")

PRIVATE_KEY_FILE := /tmp/private.pem
PUBLIC_KEY_FILE := /tmp/public.pem

all: go-arch-alignment build
.PHONY: all

build: go-arch-alignment
	go build
.PHONY: build

set-go-arch:
ifeq ($(OS), darwin)
ifneq ($(ARCH), $(GO_ARCH))
MISMATCH=yes
endif
endif
.PHONY: set-go-arch

go-arch-alignment: set-go-arch
ifdef $(MISMATCH)
$(info $(GO_MISMATCH_ERROR))
$(error Please change your go binary)
endif
.PHONY: go-arch-alignment

all: build

# Run go fmt against code
fmt:
	@${GO} fmt ./cmd/... 


# Run go vet against code
vet:
	@${GO} vet ./cmd/...

## Run all pre-commit hooks
################################################################################
# Target: precommit                                                            #
################################################################################
.PHONY: precommit
precommit:
	${PRECOMMIT} run --all

################################################################################
# Target: build	                                                               #
################################################################################
.PHONY: build
build: build-bacalhau


################################################################################
# Target: build-bacalhau                                                       #
################################################################################
.PHONY: build-bacalhau
build-bacalhau: fmt vet
	CGO_ENABLED=0 GOOS=${GO_OS} GOARCH=${GO_ARCH} ${GO} build -gcflags '-N -l' -ldflags "-X main.VERSION=$(TAG)" -o bin/${GO_OS}_$(GO_ARCH)/bacalhau main.go

################################################################################
# Target: build-docker-images
################################################################################
.PHONY: build-ipfs-sidecar-image
build-ipfs-sidecar-image: 
	docker build -t $(IPFS_FUSE_IMAGE):$(IPFS_FUSE_TAG) docker/ipfs-sidecar-image


.PHONY: build-docker-images
build-docker-images: build-ipfs-sidecar-image
	@echo docker images built

# Release tarballs suitable for upload to GitHub release pages
################################################################################
# Target: build-bacalhau-tgz                                                   #
################################################################################
.PHONY: build-bacalhau-tgz
build-bacalhau-tgz: 
	@echo "CWD: $(shell pwd)"
	@echo "RELEASE DIR: $(TMPRELEASEWORKINGDIR)"
	@echo "ARTIFACT DIR: $(TMPARTIFACTDIR)"
	mkdir $(TMPARTIFACTDIR)/$(PACKAGE)
	cp bin/$(GO_OS)_$(GO_ARCH)/bacalhau $(TMPARTIFACTDIR)/$(PACKAGE)/bacalhau
	cd $(TMPRELEASEWORKINGDIR)
	@echo "tar cvzf $(TMPARTIFACTDIR)/$(PACKAGE).tar.gz -C $(TMPARTIFACTDIR)/$(PACKAGE) $(PACKAGE)"
	tar cvzf $(TMPARTIFACTDIR)/$(PACKAGE).tar.gz -C $(TMPARTIFACTDIR)/$(PACKAGE) .
	openssl dgst -sha256 -sign $(PRIVATE_KEY_FILE)  -passin pass:"$(PRIVATE_KEY_PASSPHRASE)" -out $(TMPRELEASEWORKINGDIR)/tarsign.sha256 $(TMPARTIFACTDIR)/$(PACKAGE).tar.gz
	openssl base64 -in $(TMPRELEASEWORKINGDIR)/tarsign.sha256 -out $(TMPARTIFACTDIR)/$(PACKAGE).tar.gz.signature.sha256
	@echo "BINARY_TARBALL=$(TMPARTIFACTDIR)/$(PACKAGE).tar.gz" >> $(GITHUB_ENV)
	@echo "BINARY_TARBALL_NAME=$(PACKAGE).tar.gz" >> $(GITHUB_ENV)
	@echo "BINARY_TARBALL_SIGNATURE=$(TMPARTIFACTDIR)/$(PACKAGE).tar.gz.signature.sha256" >> $(GITHUB_ENV)
	@echo "BINARY_TARBALL_SIGNATURE_NAME=$(PACKAGE).tar.gz.signature.sha256" >> $(GITHUB_ENV)

################################################################################
# Target: clean					                               #
################################################################################
.PHONY: clean
clean:
	go clean


################################################################################
# Target: test					                               #
################################################################################
.PHONY: test
test: build-ipfs-sidecar-image
	go test ./... -v

.PHONY: test-debug
test-debug: build-ipfs-sidecar-image
	LOG_LEVEL=debug go test ./... -v

.PHONY: test-one
test-one:
	BACALHAU_RUNTIME=docker go test -v -count 1 -timeout 3000s -run ^$(TEST)$$ github.com/filecoin-project/bacalhau/cmd/bacalhau/

.PHONY: test-devstack
test-devstack:
	TEST=TestDevStack make test-one

.PHONY: test-commands
test-commands:
	TEST=TestCommands make test-one

.PHONY: test-badactors
test-badactors:
	TEST=TestCatchBadActors make test-one

################################################################################
# Target: devstack
################################################################################
.PHONY: devstack
devstack:
	BACALHAU_RUNTIME=docker go run . devstack

.PHONY: devstack-badactor
devstack-badactor:
	BACALHAU_RUNTIME=docker go run . devstack --bad-actors 1

################################################################################
# Target: lint					                               #
################################################################################
.PHONY: lint
lint: build-bacalhau
	golangci-lint run --timeout 10m

################################################################################
# Target: modtidy                                                              #
################################################################################
.PHONY: modtidy
modtidy:
	go mod tidy

################################################################################
# Target: check-diff                                                           #
################################################################################
.PHONY: check-diff
check-diff:
	git diff --exit-code ./go.mod # check no changes
	git diff --exit-code ./go.sum # check no changes

# Run the unittests and output a junit report for use with prow
################################################################################
# Target: test-junit			                                       #
################################################################################
.PHONY: test-junit
test-junit: build-bacalhau
	echo Running tests ... junit_file=$(JUNIT_FILE)
	go test ./... -v 2>&1 | go-junit-report > $(JUNIT_FILE) --set-exit-code

.PHONY: generate
generate:
	CGO_ENABLED=0 GOARCH=$(shell go env GOARCH) ${GO} generate -gcflags '-N -l' -ldflags "-X main.VERSION=$(TAG)" ./...
	echo "[OK] Files added to pipeline template directory!"

.PHONY: security
security:
	gosec -exclude=G204,G304 -exclude-dir=test ./... 
	echo "[OK] Go security check was completed!"

release: build-bacalhau
	cp bin/bacalhau .
