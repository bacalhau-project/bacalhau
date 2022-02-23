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
	CGO_ENABLED=0 GOOS=$(shell go env GOOS) GOARCH=$(shell go env GOARCH) ${GO} build -gcflags '-N -l' -ldflags "-X main.VERSION=$(TAG)" -o bin/$(ARCH)/bacalhau main.go
	cp bin/$(ARCH)/bacalhau bin/bacalhau


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
	go test ./test/... -v

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
