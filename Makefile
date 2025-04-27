export GO = go
export GOOS ?= $(shell $(GO) env GOOS)
export GOARCH ?= $(shell $(GO) env GOARCH)

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
export EARTHLY ?= $(shell command -v earthly 2> /dev/null)

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
ANALYTICS_ENDPOINT ?= ""

PRIVATE_KEY_FILE := /tmp/private.pem
PUBLIC_KEY_FILE := /tmp/public.pem

export MAKE := $(shell command -v make 2> /dev/null)

define BUILD_FLAGS
-X github.com/bacalhau-project/bacalhau/pkg/version.GITVERSION=$(TAG) \
-X github.com/bacalhau-project/bacalhau/pkg/analytics.Endpoint=$(ANALYTICS_ENDPOINT)
endef

# pypi version scheme (https://peps.python.org/pep-0440/) does not accept
# versions with dashes (e.g. 0.3.24-build-testing-01), so we replace them a valid suffix
GIT_VERSION := $(shell git describe --tags --dirty)
PYPI_VERSION ?= $(shell python3 scripts/convert_git_version_to_pep440_compatible.py $(GIT_VERSION))

export PYPI_VERSION

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
	$(error "Earthly is not installed. Please go to https://earthly.dev/get-earthly install it.")
endif

## Run all pre-commit hooks
################################################################################
# Target: precommit
#
# Temporarily creates a build directory so that go vet does not complain that
# it is missing.
################################################################################
.PHONY: precommit
precommit: check-precommit
	${PRECOMMIT} run --all
	cd python && ${MAKE} pre-commit

# Check if pre-commit is installed only for precommit target
.PHONY: check-precommit
check-precommit:
PRECOMMIT_HOOKS_INSTALLED ?= $(shell grep -R "pre-commit.com" .git/hooks)
ifeq ($(PRECOMMIT_HOOKS_INSTALLED),)
$(warning "Pre-commit is not installed in .git/hooks/pre-commit. Please run 'make install-pre-commit' to install it.")
endif

################################################################################
# Target: build-python-apiclient
################################################################################
.PHONY: build-python-apiclient
build-python-apiclient: resolve-earthly
	cd clients && ${MAKE} clean all
	@echo "Python API client built."

################################################################################
# Target: build-python-sdk
################################################################################
.PHONY: build-python-sdk
build-python-sdk:
	cd python && ${EARTHLY} --push +build --PYPI_VERSION=${PYPI_VERSION}
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
	cd integration/flyte && ${MAKE} all
	@echo "Python bacalhau-flyte built."

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
	cd python && ${MAKE} publish
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
	cd integration/flyte && ${MAKE} release
	@echo "Python flyteplugins-bacalhau pushed to PyPi."

################################################################################
# Target: build
################################################################################
.PHONY: build
build: resolve-earthly build-bacalhau

.PHONY: build-ci
build-ci: build-bacalhau

.PHONY: build-dev
build-dev: build-ci
	sudo cp ${BINARY_PATH} /usr/local/bin

################################################################################
# Target: build-webui
################################################################################
WEB_GO_FILES := $(shell find webui -name '*.go')
WEB_SRC_FILES := $(shell find webui -not -path 'webui/build/*' -not -path 'webui/build' -not -path 'webui/node_modules/*' -not -name '*.go')

.PHONY: build-webui
build-webui: resolve-earthly
	cd webui && ${EARTHLY} +all


################################################################################
# Target: build-bacalhau
################################################################################
${BINARY_PATH}: build-bacalhau

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
	docker buildx build --load \
		--platform linux/amd64,linux/arm64 \
		-t ${HTTP_GATEWAY_IMAGE}:${HTTP_GATEWAY_TAG} \
		pkg/executor/docker/gateway

.PHONY: push-http-gateway-image
push-http-gateway-image:
	docker buildx build --push \
		--platform linux/amd64,linux/arm64 \
		-t ${HTTP_GATEWAY_IMAGE}:${HTTP_GATEWAY_TAG} \
		pkg/executor/docker/gateway

BACALHAU_IMAGE ?= ghcr.io/bacalhau-project/bacalhau
BACALHAU_TAG ?= ${TAG}
BUILD_TYPE ?= default  # Options: main, nightly, release, pre-release

# Only add latest tags if the release tag is a semver tag (e.g. v0.3.12)
# and not a commit hash or a release candidate (e.g. v0.3.12-rc1)
ifeq ($(shell echo ${BACALHAU_TAG} | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$$'), ${BACALHAU_TAG})
	BASE_TAGS := --tag ${BACALHAU_IMAGE}:${BACALHAU_TAG} \
		--tag ${BACALHAU_IMAGE}:latest
	DIND_TAGS := --tag ${BACALHAU_IMAGE}:${BACALHAU_TAG}-dind \
		--tag ${BACALHAU_IMAGE}:latest-dind
else
	BASE_TAGS := --tag ${BACALHAU_IMAGE}:${BACALHAU_TAG}
	DIND_TAGS := --tag ${BACALHAU_IMAGE}:${BACALHAU_TAG}-dind
endif

BACALHAU_IMAGE_FLAGS := \
	--progress=plain \
	--label org.opencontainers.artifact.created=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ") \
	--label org.opencontainers.image.version=${BACALHAU_TAG}

.PHONY: build-bacalhau-base-image
build-bacalhau-base-image:
	docker buildx build --load ${BACALHAU_IMAGE_FLAGS} \
		${BASE_TAGS} \
		--cache-from=type=registry,ref=${BACALHAU_IMAGE}:latest \
		--file docker/bacalhau-base/Dockerfile \
		.

.PHONY: build-bacalhau-dind-image
build-bacalhau-dind-image:
	docker buildx build --load ${BACALHAU_IMAGE_FLAGS} \
		${DIND_TAGS} \
		--cache-from=type=registry,ref=${BACALHAU_IMAGE}:latest-dind \
		--file docker/bacalhau-dind/Dockerfile \
		.

# Push targets (multi-platform)
.PHONY: push-bacalhau-base-image
push-bacalhau-base-image:
	docker buildx build --push ${BACALHAU_IMAGE_FLAGS} \
		--platform linux/amd64,linux/arm64 \
		${BASE_TAGS} \
		--cache-from=type=registry,ref=${BACALHAU_IMAGE}:latest \
		--file docker/bacalhau-base/Dockerfile \
		.

.PHONY: push-bacalhau-dind-image
push-bacalhau-dind-image:
	docker buildx build --push ${BACALHAU_IMAGE_FLAGS} \
		--platform linux/amd64,linux/arm64 \
		${DIND_TAGS} \
		--cache-from=type=registry,ref=${BACALHAU_IMAGE}:latest-dind \
		--file docker/bacalhau-dind/Dockerfile \
		.

# Combined targets for building and pushing all images
.PHONY: build-bacalhau-images
build-bacalhau-images: build-bacalhau-base-image build-bacalhau-dind-image

.PHONY: push-bacalhau-images
push-bacalhau-images: push-bacalhau-base-image push-bacalhau-dind-image

.PHONY: build-docker-images
build-docker-images: build-http-gateway-image

.PHONY: push-docker-images
push-docker-images: push-http-gateway-image

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
clean:
	${GO} clean
	${RM} -r bin/*
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
test-python-sdk: resolve-earthly
# sdk tests
	cd python && ${MAKE} test

.PHONY: integration-test
integration-test:
# integration tests parallelize less well (hence -p 1)
	go test ./... -v --tags=integration -p 1

.PHONY: bash-test
bash-test:
	${BINARY_PATH}
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
	golangci-lint -v run --timeout 10m

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
		--junitfile unittests.xml \
		--format testname \
		-- \
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

.PHONY: spellcheck-code
spellcheck-code:
	cspell lint  -c cspell.yaml --quiet "**/*.{go,js,ts,jsx,tsx,md,yml,yaml,json}"

.PHONY: generate-swagger
generate-swagger:
	./scripts/generate_swagger.sh