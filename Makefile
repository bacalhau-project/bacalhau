RUSTFLAGS="-C target-feature=+crt-static"

IPFS_FUSE_IMAGE ?= "binocarlos/bacalhau-ipfs-sidecar-image"
IPFS_FUSE_TAG ?= "v1"

ifeq ($(BUILD_SIDECAR), 1)
	$(MAKE) build-ipfs-sidecar-image
endif

# Detect OS
OS := $(shell uname | tr "[:upper:]" "[:lower:]")
ARCH := $(shell uname -m | tr "[:upper:]" "[:lower:]")
GOPATH ?= $(shell go env GOPATH)
GOFLAGS ?= $(GOFLAGS:)

ifeq ($(GOOS),)
GOOS = $(shell $(GO) version | cut -c 14- | cut -d' ' -f2 | cut -d'/' -f1 | tr "[:upper:]" "[:lower:]")
endif

ifeq ($(GOARCH),)
GOARCH = $(shell $(GO) version | cut -c 14- | cut -d' ' -f2 | cut -d'/' -f2 | tr "[:upper:]" "[:lower:]")
endif

# Env Variables
export GO111MODULE = on
export GO = go
export CGO = 0
export PYTHON = python3
export PRECOMMIT = poetry run pre-commit

BUILD_DIR = bacalhau
BINARY_NAME = bacalhau

ifeq ($(GOOS),windows)
BINARY_NAME := ${BINARY_NAME}.exe
endif

BINARY_PATH = bin/${GOOS}_${GOARCH}/${BINARY_NAME}

TAG ?= $(eval TAG := $(shell git describe --tags --always))$(TAG)
COMMIT ?= $(eval COMMIT := $(shell git rev-parse HEAD))$(COMMIT)
REPO ?= $(shell echo $$(cd ../${BUILD_DIR} && git config --get remote.origin.url) | sed 's/git@\(.*\):\(.*\).git$$/https:\/\/\1\/\2/')
BRANCH ?= $(shell cd ../${BUILD_DIR} && git branch | grep '^*' | awk '{print $$2}')
BUILDDATE ?= $(eval BUILDDATE := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ'))$(BUILDDATE)

# Temp dirs
TMPRELEASEWORKINGDIR := $(shell mktemp -d -t bacalhau-release-dir.XXXXXXX)
TMPARTIFACTDIR := $(shell mktemp -d -t bacalhau-artifact-dir.XXXXXXX)
PACKAGE := $(shell echo "bacalhau_$(TAG)_${GOOS}_$(GOARCH)")

PRIVATE_KEY_FILE := /tmp/private.pem
PUBLIC_KEY_FILE := /tmp/public.pem

define BUILD_FLAGS
-X github.com/filecoin-project/bacalhau/pkg/version.GITVERSION=$(TAG) \
-X github.com/filecoin-project/bacalhau/pkg/version.GITCOMMIT=$(COMMIT) \
-X github.com/filecoin-project/bacalhau/pkg/version.BUILDDATE=$(BUILDDATE) \
-X github.com/filecoin-project/bacalhau/pkg/version.GOOS=$(GOOS) \
-X github.com/filecoin-project/bacalhau/pkg/version.GOARCH=$(GOARCH)
endef

all: build

# Run go fmt against code
fmt:
	@${GO} fmt ./cmd/...
	@${GO} fmt ./pkg/...


# Run go vet against code
vet:
	@${GO} vet ./cmd/...
	@${GO} vet ./pkg/...


## Run all pre-commit hooks
################################################################################
# Target: precommit
################################################################################
.PHONY: precommit
precommit:
	${PRECOMMIT} run --all

################################################################################
# Target: build
################################################################################
.PHONY: build
build: build-bacalhau

.PHONY: build-dev
build-dev: build
	sudo cp ${BINARY_PATH} /usr/local/bin

################################################################################
# Target: build-bacalhau
################################################################################
.PHONY: build-bacalhau
build-bacalhau: fmt vet
	CGO_ENABLED=${CGO} GOOS=${GOOS} GOARCH=${GOARCH} GO111MODULE=${GO111MODULE} ${GO} build -gcflags '-N -l' -ldflags "${BUILD_FLAGS}" -o ${BINARY_PATH} main.go

################################################################################
# Target: build-docker-images
################################################################################
.PHONY: build-ipfs-sidecar-image
build-ipfs-sidecar-image:
	docker build -t $(IPFS_FUSE_IMAGE):$(IPFS_FUSE_TAG) docker/ipfs-sidecar-image


.PHONY: build-docker-images
build-docker-images:
	@echo docker images built

# Release tarballs suitable for upload to GitHub release pages
################################################################################
# Target: build-bacalhau-tgz
################################################################################
.PHONY: build-bacalhau-tgz
build-bacalhau-tgz:
	@echo "CWD: $(shell pwd)"
	@echo "RELEASE DIR: $(TMPRELEASEWORKINGDIR)"
	@echo "ARTIFACT DIR: $(TMPARTIFACTDIR)"
	mkdir $(TMPARTIFACTDIR)/$(PACKAGE)
	cp ${BINARY_PATH} $(TMPARTIFACTDIR)/$(PACKAGE)/${BINARY_NAME}
	cd $(TMPRELEASEWORKINGDIR)
	@echo "tar cvzf $(TMPARTIFACTDIR)/$(PACKAGE).tar.gz -C $(TMPARTIFACTDIR)/$(PACKAGE) $(PACKAGE)"
	tar cvzf $(TMPARTIFACTDIR)/$(PACKAGE).tar.gz -C $(TMPARTIFACTDIR)/$(PACKAGE) .
	openssl dgst -sha256 -sign $(PRIVATE_KEY_FILE)  -passin pass:"$(PRIVATE_KEY_PASSPHRASE)" -out $(TMPRELEASEWORKINGDIR)/tarsign.sha256 $(TMPARTIFACTDIR)/$(PACKAGE).tar.gz
	openssl base64 -in $(TMPRELEASEWORKINGDIR)/tarsign.sha256 -out $(TMPARTIFACTDIR)/$(PACKAGE).tar.gz.signature.sha256
	@echo "export ARTIFACT_DIR=$(TMPARTIFACTDIR)" >> /tmp/packagevars
	@echo "export BINARY_TARBALL=$(TMPARTIFACTDIR)/$(PACKAGE).tar.gz" >> /tmp/packagevars
	@echo "export BINARY_TARBALL_NAME=$(PACKAGE).tar.gz" >> /tmp/packagevars
	@echo "export BINARY_TARBALL_SIGNATURE=$(TMPARTIFACTDIR)/$(PACKAGE).tar.gz.signature.sha256" >> /tmp/packagevars
	@echo "export BINARY_TARBALL_SIGNATURE_NAME=$(PACKAGE).tar.gz.signature.sha256" >> /tmp/packagevars

################################################################################
# Target: clean
################################################################################
.PHONY: clean
clean:
	go clean


################################################################################
# Target: test
################################################################################
.PHONY: test
test:
	go test ./... -v -p 4

.PHONY: grc-test
grc-test:
	grc go test ./... -v -p 4

.PHONY: test-debug
test-debug: 
	LOG_LEVEL=debug go test ./... -v -p 4

.PHONY: grc-test-debug
grc-test-debug:
	LOG_LEVEL=debug grc go test ./... -v -p 4

.PHONY: test-one
test-one:
	go test -v -count 1 -timeout 3000s -run ^$(TEST)$$ github.com/filecoin-project/bacalhau/cmd/bacalhau/

.PHONY: test-devstack
test-devstack:
	go test -v -count 1 -timeout 3000s -run '^Test\w+Suite$$' github.com/filecoin-project/bacalhau/pkg/test/devstack/

.PHONY: test-commands
test-commands:
	go test -v -count 1 -timeout 3000s -run '^Test\w+Suite$$' github.com/filecoin-project/bacalhau/cmd/bacalhau/

# .PHONY: test-badactors
# test-badactors:
# 	TEST=TestCatchBadActors make test-one

# .PHONY: test-pythonwasm
# test-pythonwasm:
# # TestSimplestPythonWasmDashC
# 	LOG_LEVEL=debug go test -v -count 1 -timeout 3000s -run ^TestSimplePythonWasm$$ github.com/filecoin-project/bacalhau/pkg/test/devstack/
# #	LOG_LEVEL=debug go test -v -count 1 -timeout 3000s -run ^TestSimplestPythonWasmDashC$$ github.com/filecoin-project/bacalhau/pkg/test/devstack/

################################################################################
# Target: devstack
################################################################################
.PHONY: devstack
devstack:
	go run . devstack

.PHONY: devstack-one
devstack-one:
	IGNORE_PORT_FILES=true PREDICTABLE_API_PORT=1 go run . devstack --nodes 1

.PHONY: devstack-100
devstack-100:
	go run . devstack --nodes 100

.PHONY: devstack-250
devstack-250:
	go run . devstack --nodes 250

.PHONY: devstack-20
devstack-20:
	go run . devstack --nodes 20

.PHONY: devstack-noop
devstack-noop:
	go run . devstack --noop

.PHONY: devstack-noop-100
devstack-noop-100:
	go run . devstack --noop --nodes 100

.PHONY: devstack-race
devstack-race:
	go run -race . devstack

.PHONY: devstack-badactor
devstack-badactor:
	go run . devstack --bad-actors 1

################################################################################
# Target: lint
################################################################################
.PHONY: lint
lint:
	golangci-lint run --timeout 10m

.PHONY: lint-fix
lint-fix:
	golangci-lint run --timeout 10m --fix

################################################################################
# Target: modtidy
################################################################################
.PHONY: modtidy
modtidy:
	go mod tidy

################################################################################
# Target: check-diff
################################################################################
.PHONY: check-diff
check-diff:
	git diff --exit-code ./go.mod # check no changes
	git diff --exit-code ./go.sum # check no changes

# Run the unittests and output results for recording
################################################################################
# Target: test-test-and-report
################################################################################
.PHONY: test-and-report
test-and-report: build-bacalhau
	CGO_ENABLED=${CGO} \
		gotestsum \
			--jsonfile ${TEST_OUTPUT_FILE_PREFIX}_unit.json \
			--junitfile unittests.xml \
			--format standard-quiet \
			-- \
				-p 1 \
				./pkg/... ./cmd/... \
				$(COVERAGE_OPTS) --tags=unit

.PHONY: generate
generate:
	CGO_ENABLED=0 GOARCH=$(shell go env GOARCH) GO111MODULE=${GO111MODULE} ${GO} generate -gcflags '-N -l' -ldflags "-X main.VERSION=$(TAG)" ./...
	echo "[OK] Files added to pipeline template directory!"

.PHONY: security
security:
	gosec -exclude=G204,G304 -exclude-dir=test ./...
	echo "[OK] Go security check was completed!"

release: build-bacalhau
	cp bin/bacalhau .
