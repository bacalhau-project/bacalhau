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
ARCH ?= $(shell go env GOOS)_$(shell go env GOARCH)

# Temp dirs
TMPRELEASEWORKINGDIR := $(shell mktemp -d -t bacalhau-release-dir.XXXXXXX)
TMPARTIFACTDIR := $(shell mktemp -d -t bacalhau-artifact-dir.XXXXXXX)
PACKAGE := $(shell echo "same_$(TAG)_$(ARCH)")

all: go-arch-alignment build
.PHONY: all

build:
	go build
.PHONY: build

go-arch-alignment:
mismatch = 
ifeq ($(OS), darwin)
ifneq ($(ARCH), $(GO_ARCH))
mismatch = yes
endif
endif

ifdef mismatch
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
# Target: build-bacalhau                                                          #
################################################################################
.PHONY: build-bacalhau
build-bacalhau: fmt vet
	CGO_ENABLED=0 GOOS=${GO_OS} GOARCH=${GO_ARCH} ${GO} build -gcflags '-N -l' -ldflags "-X main.VERSION=$(TAG)" -o bin/$(ARCH)/bacalhau main.go
	cp bin/$(ARCH)/bacalhau bin/bacalhau

# Release tarballs suitable for upload to GitHub release pages
################################################################################
# Target: build-bacalhau-tgz                                                       #
################################################################################
.PHONY: build-bacalhau-tgz
build-bacalhau-tgz: build-bacalhau
	@echo "CWD: $(shell pwd)"
	@echo "RELEASE DIR: $(TMPRELEASEWORKINGDIR)"
	@echo "ARTIFACT DIR: $(TMPARTIFACTDIR)"
	mkdir $(TMPARTIFACTDIR)/$(PACKAGE)
	cp bin/$(ARCH)/bacalhau $(TMPARTIFACTDIR)/$(PACKAGE)/bacalhau
	cd $(TMPRELEASEWORKINGDIR)
	@echo "tar cvzf $(TMPARTIFACTDIR)/$(PACKAGE).tar.gz -C $(TMPARTIFACTDIR)/$(PACKAGE) $(PACKAGE)"
	tar cvzf $(TMPARTIFACTDIR)/$(PACKAGE).tar.gz -C $(TMPARTIFACTDIR)/$(PACKAGE) .
	@echo "BINARY_TARBALL=$(TMPARTIFACTDIR)/$(PACKAGE).tar.gz" >> $(GITHUB_ENV)
	@echo "BINARY_TARBALL_NAME=$(PACKAGE).tar.gz" >> $(GITHUB_ENV)

################################################################################
# Target: clean					                                               #
################################################################################
.PHONY: clean
clean:
	go clean


################################################################################
# Target: test					                                               #
################################################################################
.PHONY: test
test: build-bacalhau
	go test ./... -v

################################################################################
# Target: lint					                                               #
################################################################################
.PHONY: lint
lint: build-bacalhau
	golangci-lint run --timeout 10m

# Run the unittests and output a junit report for use with prow
################################################################################
# Target: test-junit			                                               #
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
