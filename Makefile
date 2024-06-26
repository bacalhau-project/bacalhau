export GO = go
export GOOS ?= $(shell $(GO) env GOOS)
export GOARCH ?= $(shell $(GO) env GOARCH)
export FLOXREPOSITORY = aronchick

UNAME_S := $(shell uname -s)

# Detect if gsed is installed - sed default on MacOS is very old
ifeq ($(UNAME_S),Darwin)
SED := $(shell command -v gsed 2> /dev/null)
ifeq ($(SED),)
$(warning gsed is not installed. Please run 'brew install gsed' to install it. You may have issues with the Makefile. Falling back to default sed.)
export SED = sed
else
export SED = gsed
endif
endif

ifeq ($(UNAME_S),Linux)
export SED = sed
endif

ifeq ($(GOARCH),armv6)
export GOARCH = arm
export GOARM = 6
endif

ifeq ($(GOARCH),armv7)
export GOARCH = arm
export GOARM = 7
endif

# Env Variables
export GO111MODULE = on
export CGO_ENABLED = 0
export PRECOMMIT = poetry run pre-commit
export EARTHLY ?= $(shell command -v earthly --push 2> /dev/null)

BUILD_DIR = bacalhau
BINARY_NAME = bacalhau

ifeq ($(GOOS),windows)
BINARY_NAME := ${BINARY_NAME}.exe
CC = gcc.exe
endif

BINARY_PATH = bin/${GOOS}/${GOARCH}${GOARM}/${BINARY_NAME}

TAG ?= $(eval TAG := $(shell git describe --tags --always))$(TAG)
COMMIT ?= $(eval COMMIT := $(shell git rev-parse HEAD))$(COMMIT)
REPO ?= $(shell echo $$(cd ../${BUILD_DIR} && git config --get remote.origin.url) | $(SED) 's/git@\(.*\):\(.*\).git$$/https:\/\/\1\/\2/')
BRANCH ?= $(shell cd ../${BUILD_DIR} && git branch | grep '^*' | awk '{print $$2}')
BUILDDATE ?= $(eval BUILDDATE := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ'))$(BUILDDATE)
PACKAGE := $(shell echo "bacalhau_$(TAG)_${GOOS}_$(GOARCH)${GOARM}")
TEST_BUILD_TAGS ?= unit,integration
TEST_PARALLEL_PACKAGES ?= 1

PRIVATE_KEY_FILE := /tmp/private.pem
PUBLIC_KEY_FILE := /tmp/public.pem

export MAKE := $(shell command -v make 2> /dev/null)
export JUST := $(shell command -v just 2> /dev/null)

define BUILD_FLAGS
-X github.com/bacalhau-project/bacalhau/pkg/version.GITVERSION=$(TAG)
endef

# pypi version scheme (https://peps.python.org/pep-0440/) does not accept
# versions with dashes (e.g. 0.3.24-build-testing-01), so we replace them a valid suffix
export GIT_VERSION := $(shell git describe --tags --dirty)
export PYPI_VERSION ?= $(shell python3 scripts/convert_git_version_to_pep440_compatible.py $(GIT_VERSION))

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

.PHONY: resolve-earthly
resolve-earthly:
	@echo "Resolved Earthly path - ${EARTHLY}"
ifeq ($(EARTHLY),)
	$(error "Earthly is not installed. Please go to https://earthly.dev/get-earthly to install it.")
endif

.PHONY: resolve-just
resolve-just:
	@echo "Resolved Just path - ${JUST}"
ifeq ($(JUST),)
	$(error "Just is not installed. Please go to https://github.com/casey/just to install it.")
endif


## Run all pre-commit hooks
################################################################################
# Target: precommit
#
# Temporarily creates a build directory so that go vet does not complain that
# it is missing.
################################################################################
.PHONY: precommit
precommit:
	@mkdir -p webui/build && touch webui/build/stub
	${PRECOMMIT} run --all
	@rm webui/build/stub
	cd python && ${MAKE} pre-commit

PRECOMMIT_HOOKS_INSTALLED ?= $(shell grep -R "pre-commit.com" .git/hooks)
ifeq ($(PRECOMMIT_HOOKS_INSTALLED),)
$(warning "Pre-commit is not installed in .git/hooks/pre-commit. Please run 'make install-pre-commit' to install it.")
endif

################################################################################
# Target: build-python-apiclient
################################################################################
.PHONY: build-python-apiclient
build-python-apiclient:
	cd clients && ${JUST} clean all
	@echo "Python API client built."

################################################################################
# Target: build-python-sdk
################################################################################
.PHONY: build-python-sdk
build-python-sdk:
	cd python && ${JUST} build
	@echo "Python SDK built."

################################################################################
# Target: build-bacalhau-airflow
################################################################################
.PHONY: build-bacalhau-airflow
build-bacalhau-airflow: resolve-earthly
	cd integration/airflow && ${MAKE} clean all
	@echo "Python bacalhau-airflow built."

################################################################################
# Target: build-bacalhau-flyte
################################################################################
.PHONY: build-bacalhau-flyte
build-bacalhau-flyte:
	$(error "Flyte Plugins NOT built - the libaries are out of date.")

# Builds all python packages
################################################################################
# Target: build-python
################################################################################
.PHONY: build-python
build-python: build-python-apiclient build-python-sdk build-bacalhau-airflow

################################################################################
# Target: release-python-apiclient
################################################################################
.PHONY: release-python-apiclient
release-python-apiclient: resolve-earthly
	cd clients && ${MAKE} release
	@echo "Python API client pushed to PyPi."

################################################################################
# Target: release-python-sdk
################################################################################
.PHONY: release-python-sdk
release-python-sdk: build-python-sdk
	cd python && ${EARTHLY} --push +publish --PYPI_TOKEN=${PYPI_TOKEN}
	@echo "Python SDK pushed to PyPi."

################################################################################
# Target: release-bacalhau-airflow
################################################################################
.PHONY: release-bacalhau-airflow
release-bacalhau-airflow: resolve-earthly
	cd integration/airflow && ${MAKE} release
	@echo "Python bacalhau-airflow pushed to PyPi."

################################################################################
# Target: release-bacalhau-flyte
################################################################################
.PHONY: release-bacalhau-flyte
release-bacalhau-flyte: resolve-earthly
	$(error "Flyte Plugins NOT released - the libaries are out of date.")

################################################################################
# Target: build
################################################################################
.PHONY: build
build: resolve-earthly build-bacalhau build-plugins

.PHONY: build-ci
build-ci: build-bacalhau install-plugins

.PHONY: build-dev
build-dev: build-ci
	sudo cp ${BINARY_PATH} /usr/local/bin

################################################################################
# Target: build-webui
################################################################################
WEB_GO_FILES := $(shell find webui -name '*.go')
WEB_SRC_FILES := $(shell find webui -not -path 'webui/build/*' -not -path 'webui/build' -not -path 'webui/node_modules/*' -not -name '*.go')

.PHONY: build-webui
build-webui:
	cd webui && flox activate -r "${FLOXREPOSITORY}/webui" -- just all


################################################################################
# Target: build-bacalhau
################################################################################
${BINARY_PATH}: build-bacalhau build-plugins

.PHONY: build-bacalhau
build-bacalhau: binary-web binary

CMD_FILES := $(shell bash -c 'comm -23 <(git ls-files cmd | sort) <(git ls-files cmd --deleted | sort)')
PKG_FILES := $(shell bash -c 'comm -23 <(git ls-files pkg | sort) <(git ls-files pkg --deleted | sort)')

.PHONY: binary

binary: ${CMD_FILES} ${PKG_FILES} main.go
	${GO} build -ldflags "${BUILD_FLAGS}" -trimpath -o ${BINARY_PATH} .

binary-web: build-webui ${WEB_GO_FILES}

################################################################################
# Target: build-docker-images
################################################################################
HTTP_GATEWAY_IMAGE ?= "ghcr.io/bacalhau-project/http-gateway"
HTTP_GATEWAY_TAG ?= ${TAG}
.PHONY: build-http-gateway-image
build-http-gateway-image:
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		-t ${HTTP_GATEWAY_IMAGE}:${HTTP_GATEWAY_TAG} \
		pkg/executor/docker/gateway

BACALHAU_IMAGE ?= ghcr.io/bacalhau-project/bacalhau
BACALHAU_TAG ?= ${TAG}

# Only tag images with :latest if the release tag is a semver tag (e.g. v0.3.12)
# and not a commit hash or a release candidate (e.g. v0.3.12-rc1)
LATEST_TAG :=
ifeq ($(shell echo ${BACALHAU_TAG} | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$$'), ${BACALHAU_TAG})
	LATEST_TAG := --tag ${BACALHAU_IMAGE}:latest
endif

BACALHAU_IMAGE_FLAGS := \
	--progress=plain \
	--platform linux/amd64,linux/arm64 \
	--tag ${BACALHAU_IMAGE}:${BACALHAU_TAG} \
	${LATEST_TAG} \
	--label org.opencontainers.artifact.created=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ") \
	--label org.opencontainers.image.version=${BACALHAU_TAG} \
	--cache-from=type=registry,ref=${BACALHAU_IMAGE}:latest \
	--file docker/bacalhau-image/Dockerfile \
	.

.PHONY: build-bacalhau-image
build-bacalhau-image:
	docker buildx build ${BACALHAU_IMAGE_FLAGS}

.PHONY: push-bacalhau-image
push-bacalhau-image:
	docker buildx build --push ${BACALHAU_IMAGE_FLAGS}

.PHONY: build-docker-images
build-docker-images: build-http-gateway-image

.PHONY: push-docker-images
push-docker-images: build-http-gateway-image

# Release tarballs suitable for upload to GitHub release pages
################################################################################
# Target: build-bacalhau-tgz
################################################################################
.PHONY: build-bacalhau-tgz
build-bacalhau-tgz: dist/${PACKAGE}.tar.gz dist/${PACKAGE}.tar.gz.signature.sha256

dist/:
	mkdir -p $@

dist/${PACKAGE}.tar.gz: ${BINARY_PATH} | dist/
	tar cvzf $@ -C $(dir $(BINARY_PATH)) $(notdir ${BINARY_PATH})

dist/${PACKAGE}.tar.gz.signature.sha256: dist/${PACKAGE}.tar.gz | dist/
	openssl dgst -sha256 -sign $(PRIVATE_KEY_FILE) -passin pass:"$(PRIVATE_KEY_PASSPHRASE)" $^ | openssl base64 -out $@

################################################################################
# Target: images
################################################################################
IMAGE_REGEX := 'Image ?(:|=)\s*"[^"]+"'
FILES_WITH_IMAGES := $(shell grep -Erl ${IMAGE_REGEX} pkg cmd)

docker/.images: ${FILES_WITH_IMAGES}
	grep -Eroh ${IMAGE_REGEX} $^ | cut -d'"' -f2 | sort | uniq > $@

docker/.pulled: docker/.images
	- cat $^ | xargs -n1 docker pull
	touch $@

.PHONY: images
images: docker/.pulled

################################################################################
# Target: clean
################################################################################
.PHONY: clean
clean: clean-plugins
	${GO} clean
	${RM} -r bin/*
	${RM} -r webui/build/*
	${RM} -r webui/node_modules
	${RM} dist/bacalhau_*
	${RM} docker/.images
	${RM} docker/.pulled


################################################################################
# Target: test
################################################################################
.PHONY: test
test: unit-test bash-test

.PHONY: unit-test
unit-test:
# unittests parallelize well (default go test behavior is to parallelize)
	go test ./... -v --tags=unit

.PHONY: test-python-sdk
test-python-sdk:
# sdk tests
	cd python && ${JUST} test

.PHONY: integration-test
integration-test:
# integration tests parallelize less well (hence -p 1)
	go test ./... -v --tags=integration -p 1

.PHONY: bash-test
bash-test: ${BINARY_PATH}
	cd test && bin/bashtub *.sh

.PHONY: test-debug
test-debug:
	LOG_LEVEL=debug go test ./... -v

.PHONY: test-one
test-one:
	go test -v -count 1 -timeout 3000s -run ^$(TEST)$$ github.com/bacalhau-project/bacalhau/cmd/bacalhau/

.PHONY: test-devstack
test-devstack:
	go test -v -count 1 -timeout 3000s -run '^Test\w+Suite$$' github.com/bacalhau-project/bacalhau/pkg/test/devstack/

.PHONY: test-commands
test-commands:
	go test -v -count 1 -timeout 3000s -run '^Test\w+Suite$$' github.com/bacalhau-project/bacalhau/cmd/bacalhau/

.PHONY: test-all
test-all: test test-python-sdk
	cd webui && yarn run build && yarn run lint && yarn run test

################################################################################
# Target: devstack
################################################################################
.PHONY: devstack
devstack:
	go run . devstack

.PHONY: devstack-one
devstack-one:
	IGNORE_PID_AND_PORT_FILES=true PREDICTABLE_API_PORT=1 go run . devstack --requester-nodes 0 --compute-nodes 0 --hybrid-nodes 1

.PHONY: devstack-100
devstack-100:
	go run . devstack --compute-nodes 100

.PHONY: devstack-250
devstack-250:
	go run . devstack --compute-nodes 250

.PHONY: devstack-20
devstack-20:
	go run . devstack --compute-nodes 20

.PHONY: devstack-noop
devstack-noop:
	go run . devstack --noop

.PHONY: devstack-noop-100
devstack-noop-100:
	go run . devstack --noop --compute-nodes 100

.PHONY: devstack-race
devstack-race:
	go run -race . devstack

.PHONY: devstack-badactor
devstack-badactor:
	go run . devstack --bad-compute-actors 1

################################################################################
# Target: lint
################################################################################
.PHONY: lint
lint:
	golangci-lint run --timeout 10m

.PHONY: lint-fix
lint-fix:
	golangci-lint run --timeout 10m

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
COMMA = ,
COVER_FILE := coverage/${PACKAGE}_$(subst ${COMMA},_,${TEST_BUILD_TAGS}).coverage

.PHONY: test-and-report
test-and-report: unittests.xml ${COVER_FILE}

${COVER_FILE} unittests.xml ${TEST_OUTPUT_FILE_PREFIX}_unit.json &: ${CMD_FILES} ${PKG_FILES} $(dir ${COVER_FILE})
	gotestsum \
		--jsonfile ${TEST_OUTPUT_FILE_PREFIX}_unit.json \
		--junitfile unittests.xml \
		--format testname \
		-- \
			-p ${TEST_PARALLEL_PACKAGES} \
			./pkg/... ./cmd/... \
			-coverpkg=./... -coverprofile=${COVER_FILE} \
			--tags=${TEST_BUILD_TAGS}

################################################################################
# Target: coverage-report
################################################################################
.PHONY:
coverage-report: coverage/coverage.html

coverage/coverage.out: $(wildcard coverage/*.coverage)
	gocovmerge $^ > $@

coverage/coverage.html: coverage/coverage.out coverage/
	go tool cover -html=$< -o $@

coverage/:
	mkdir -p $@

.PHONY: generate
generate: generate-tools
	${GO} generate -gcflags '-N -l' -ldflags "-X main.VERSION=$(TAG)" ./...
	@echo "[OK] Files added to pipeline template directory!"

.PHONY: generate-tools
generate-tools:
	@which mockgen > /dev/null || echo "Installing 'mockgen'" && ${GO} install go.uber.org/mock/mockgen@latest
	@which stringer > /dev/null  || echo "Installing 'stringer'" && ${GO} install golang.org/x/tools/cmd/stringer


.PHONY: security
security:
	gosec -exclude=G204,G304 -exclude-dir=test ./...
	echo "[OK] Go security check was completed!"

release: build-bacalhau
	cp bin/bacalhau .

ifeq ($(OS),Windows_NT)
    detected_OS := Windows
else
    detected_OS := $(shell sh -c 'uname 2>/dev/null || echo Unknown')
endif

# TODO make the plugin path configurable instead of using the bacalhau config path.
BACALHAU_CONFIG_PATH := $(shell echo $$BACALHAU_PATH)
INSTALL_PLUGINS_DEST := $(if $(BACALHAU_CONFIG_PATH),$(BACALHAU_CONFIG_PATH)plugins/,~/.bacalhau/plugins/)

EXECUTOR_PLUGINS := $(wildcard ./pkg/executor/plugins/executors/*/.)

# TODO fix install on windows
ifeq ($(detected_OS),Windows)
    build-plugins clean-plugins install-plugins:
	@echo "Skipping executor plugins on Windows"
else
    build-plugins: plugins-build
    clean-plugins: plugins-clean
    install-plugins: plugins-install

    .PHONY: plugins-build $(EXECUTOR_PLUGINS)

    plugins-build: $(EXECUTOR_PLUGINS)
	@echo "Building executor plugins..."
	@$(foreach plugin,$(EXECUTOR_PLUGINS),$(MAKE) --no-print-directory -C $(plugin) &&) true

    .PHONY: plugins-clean $(addsuffix .clean,$(EXECUTOR_PLUGINS))

    plugins-clean: $(addsuffix .clean,$(EXECUTOR_PLUGINS))
	@echo "Cleaning executor plugins..."
	@$(foreach plugin,$(addsuffix .clean,$(EXECUTOR_PLUGINS)),$(MAKE) --no-print-directory -C $(basename $(plugin)) clean &&) true

    .PHONY: plugins-install $(addsuffix .install,$(EXECUTOR_PLUGINS))

    plugins-install: plugins-build $(addsuffix .install,$(EXECUTOR_PLUGINS))
	@echo "Installing executor plugins..."
	@$(foreach plugin,$(addsuffix .install,$(EXECUTOR_PLUGINS)),mkdir -p $(INSTALL_PLUGINS_DEST) && cp $(basename $(plugin))/bin/* $(INSTALL_PLUGINS_DEST) &&) true
endif

.PHONY: spellcheck-code
spellcheck-code:  ## Runs a spellchecker over all code - MVP just does one file
	cspell -c .cspell-code.json lint ./pkg/authn/**

.PHONY: spellcheck-docs
spellcheck-docs:  ## Runs a spellchecker over all documentation - MVP just does one directory
	cspell -c .cspell-docs.json lint ./docs/docs/dev/**
