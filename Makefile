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
GOOS = $(shell $(GO) env GOOS)
endif

ifeq ($(GOARCH),)
GOARCH = $(shell $(GO) env GOARCH)
endif

# Env Variables
export GO111MODULE = on
export GO = go
export CGO_ENABLED = 0
export PYTHON = python3ƒ
export PRECOMMIT = poetry run pre-commit

BUILD_DIR = bacalhau
BINARY_NAME = bacalhau

ifeq ($(GOOS),windows)
BINARY_NAME := ${BINARY_NAME}.exe
CC = gcc.exe
endif

BINARY_PATH = bin/${GOOS}_${GOARCH}/${BINARY_NAME}

TAG ?= $(eval TAG := $(shell git describe --tags --always))$(TAG)
COMMIT ?= $(eval COMMIT := $(shell git rev-parse HEAD))$(COMMIT)
REPO ?= $(shell echo $$(cd ../${BUILD_DIR} && git config --get remote.origin.url) | sed 's/git@\(.*\):\(.*\).git$$/https:\/\/\1\/\2/')
BRANCH ?= $(shell cd ../${BUILD_DIR} && git branch | grep '^*' | awk '{print $$2}')
BUILDDATE ?= $(eval BUILDDATE := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ'))$(BUILDDATE)
PACKAGE := $(shell echo "bacalhau_$(TAG)_${GOOS}_$(GOARCH)")
PRECOMMIT_HOOKS_INSTALLED ?= $(shell grep -R "pre-commit.com" .git/hooks)
TEST_BUILD_TAGS ?= unit,integration

PRIVATE_KEY_FILE := /tmp/private.pem
PUBLIC_KEY_FILE := /tmp/public.pem

define BUILD_FLAGS
-X github.com/filecoin-project/bacalhau/pkg/version.GITVERSION=$(TAG)
endef

all: build

# Run init repo after cloning it
.PHONY: init
init:
	@ops/repo_init.sh 1>/dev/null
	@echo "Build environment initialized."

# Run install pre-commit
.PHONY: install-pre-commit
install-pre-commit:
	@ops/install_pre_commit.sh 1>/dev/null
	@echo "Pre-commit installed."

## Run all pre-commit hooks
################################################################################
# Target: precommit
################################################################################
.PHONY: precommit
precommit: buildenvcorrect
	${PRECOMMIT} run --all

.PHONY: buildenvcorrect
buildenvcorrect:
	@echo "Checking build environment..."
# Checking GO
# @echo "Checking for go..."
# @which go
# @echo "Checking for go version..."
# @go version
# @echo "Checking for go env..."
# @go env
# @echo "Checking for go env GOOS..."
# @go env GOOS
# @echo "Checking for go env GOARCH..."
# @go env GOARCH
# @echo "Checking for go env GO111MODULE..."
# @go env GO111MODULE
# @echo "Checking for go env GOPATH..."
# @go env GOPATH
# @echo "Checking for go env GOCACHE..."
# @go env GOCACHE
# ===============
# Ensure that "pre-commit.com" is in .git/hooks/pre-commit to run all pre-commit hooks
# before each commit.
# Error if it's empty or not found.
ifeq ($(PRECOMMIT_HOOKS_INSTALLED),)
	@echo "Pre-commit is not installed in .git/hooks/pre-commit. Please run 'make install-pre-commit' to install it."
	@exit 1
endif
	@echo "Build environment correct."


################################################################################
# Target: build
################################################################################
.PHONY: build
build: buildenvcorrect build-bacalhau

.PHONY: build-ci
build-ci: build-bacalhau

.PHONY: build-dev
build-dev: build-ci
	sudo cp ${BINARY_PATH} /usr/local/bin

################################################################################
# Target: build-bacalhau
################################################################################
.PHONY: build-bacalhau
build-bacalhau: ${BINARY_PATH}

${BINARY_PATH}: $(shell git ls-files cmd) $(shell git ls-files pkg)
	${GO} build -gcflags '-N -l' -ldflags "${BUILD_FLAGS}" -trimpath -o ${BINARY_PATH} .

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
build-bacalhau-tgz: dist/${PACKAGE}.tar.gz dist/${PACKAGE}.tar.gz.signature.sha256

dist/${PACKAGE}.tar.gz: ${BINARY_PATH}
	tar cvzf $@ -C $(dir $(BINARY_PATH)) $(notdir ${BINARY_PATH})

dist/${PACKAGE}.tar.gz.signature.sha256: dist/${PACKAGE}.tar.gz
	openssl dgst -sha256 -sign $(PRIVATE_KEY_FILE) -passin pass:"$(PRIVATE_KEY_PASSPHRASE)" $^ | openssl base64 -out $@

################################################################################
# Target: clean
################################################################################
.PHONY: clean
clean:
	${GO} clean
	${RM} -r bin/*
	${RM} dist/bacalhau_*


################################################################################
# Target: test
################################################################################
.PHONY: test
test:
# unittests parallelize well (default go test behavior is to parallelize)
	go test ./... -v

.PHONY: integration-test
integration-test:
# integration tests parallelize less well (hence -p 1)
	go test ./... -v --tags=integration -p 1

.PHONY: grc-test
grc-test:
	grc go test ./... -v -p 4
.PHONY: grc-test-short
grc-test-short:
	grc go test ./... -test.short -v

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
test-and-report: unittests.xml coverage/coverage.html

coverage.out unittests.xml ${TEST_OUTPUT_FILE_PREFIX}_unit.json: ${BINARY_PATH}
	gotestsum \
		--jsonfile ${TEST_OUTPUT_FILE_PREFIX}_unit.json \
		--junitfile unittests.xml \
		--format standard-quiet \
		-- \
			-p 1 \
			./pkg/... ./cmd/... \
			-coverpkg=./... -coverprofile=coverage.out \
			--tags=${TEST_BUILD_TAGS}

coverage/coverage.html: coverage.out coverage/ 
	go tool cover -html=$< -o $@

coverage/:
	mkdir -p $@

.PHONY: generate
generate:
	${GO} generate -gcflags '-N -l' -ldflags "-X main.VERSION=$(TAG)" ./...
	echo "[OK] Files added to pipeline template directory!"

.PHONY: security
security:
	gosec -exclude=G204,G304 -exclude-dir=test ./...
	echo "[OK] Go security check was completed!"

release: build-bacalhau
	cp bin/bacalhau .
