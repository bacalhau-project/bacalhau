export GO = go
export GOOS ?= $(shell $(GO) env GOOS)
export GOARCH ?= $(shell $(GO) env GOARCH)

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

BUILD_DIR = bacalhau
BINARY_NAME = bacalhau

ifeq ($(GOOS),windows)
BINARY_NAME := ${BINARY_NAME}.exe
CC = gcc.exe
endif

BINARY_PATH = bin/${GOOS}_${GOARCH}${GOARM}/${BINARY_NAME}

TAG ?= $(eval TAG := $(shell git describe --tags --always))$(TAG)
COMMIT ?= $(eval COMMIT := $(shell git rev-parse HEAD))$(COMMIT)
REPO ?= $(shell echo $$(cd ../${BUILD_DIR} && git config --get remote.origin.url) | sed 's/git@\(.*\):\(.*\).git$$/https:\/\/\1\/\2/')
BRANCH ?= $(shell cd ../${BUILD_DIR} && git branch | grep '^*' | awk '{print $$2}')
BUILDDATE ?= $(eval BUILDDATE := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ'))$(BUILDDATE)
PACKAGE := $(shell echo "bacalhau_$(TAG)_${GOOS}_$(GOARCH)${GOARM}")
PRECOMMIT_HOOKS_INSTALLED ?= $(shell grep -R "pre-commit.com" .git/hooks)
TEST_BUILD_TAGS ?= unit,integration
TEST_PARALLEL_PACKAGES ?= 1

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
	cd python && make pre-commit

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
# Target: swagger-docs
################################################################################
.PHONY: swagger-docs
swagger-docs:
	@echo "Building swagger docs..."
	swag fmt -g "pkg/publicapi/server.go" && \
	swag init --outputTypes "go,json" --parseDependency --parseInternal --parseDepth 1 --markdownFiles docs/swagger -g "pkg/publicapi/server.go"
	@echo "Swagger docs built."

################################################################################
# Target: clients
################################################################################
# Generate Bacalhau API clients but only if the swagger.json has actually been
# updated because the clients include random numbers and timestamps and hence
# will generate a lot of noisy diffs if regenerated all of the time
.PHONY: clients
clients:
	(test -n "$(shell git ls-files --modified docs/swagger.json)" && \
		cd clients && ${MAKE} -j all) || true

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

CMD_FILES := $(shell bash -c 'comm -23 <(git ls-files cmd) <(git ls-files cmd --deleted)')
PKG_FILES := $(shell bash -c 'comm -23 <(git ls-files pkg) <(git ls-files pkg --deleted)')

${BINARY_PATH}: ${CMD_FILES} ${PKG_FILES} main.go
	${GO} build -ldflags "${BUILD_FLAGS}" -trimpath -o ${BINARY_PATH} .

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
.PHONY: build-bacalhau-image
build-bacalhau-image:
	docker build --progress=plain \
		--tag ${BACALHAU_IMAGE}:latest \
		--file docker/bacalhau-image/Dockerfile \
		.

.PHONY: push-bacalhau-image
push-bacalhau-image:
	docker buildx build --push --progress=plain \
		--platform linux/amd64,linux/arm64 \
		--tag ${BACALHAU_IMAGE}:${BACALHAU_TAG} \
		--tag ${BACALHAU_IMAGE}:latest \
		--label org.opencontainers.artifact.created=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ") \
		--label org.opencontainers.image.version=${BACALHAU_TAG} \
		--cache-from=type=registry,ref=${BACALHAU_IMAGE}:latest \
		--file docker/bacalhau-image/Dockerfile \
		.

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
clean:
	${GO} clean
	${RM} -r bin/*
	${RM} dist/bacalhau_*
	${RM} docker/.images
	${RM} docker/.pulled


################################################################################
# Target: schema
################################################################################
SCHEMA_DIR ?= schema.bacalhau.org/jsonschema

.PHONY: schema
schema: ${SCHEMA_DIR}/$(shell git describe --tags --abbrev=0).json

${SCHEMA_DIR}/%.json:
	./scripts/build-schema-file.sh $$(basename -s .json $@) > $@

################################################################################
# Target: all_schemas
################################################################################
EARLIEST_TAG := v0.3.12
ALL_TAGS := $(shell git tag -l --contains $$(git rev-parse ${EARLIEST_TAG}) | grep -E 'v\d+\.\d+.\d+')
ALL_SCHEMAS := $(patsubst %,${SCHEMA_DIR}/%.json,${ALL_TAGS})

.PHONY: all_schemas
all_schemas: ${ALL_SCHEMAS}


################################################################################
# Target: test
################################################################################
.PHONY: test
test:
# unittests parallelize well (default go test behavior is to parallelize)
	go test ./... -v --tags=unit

.PHONY: test-python
test-python:
	cd python && make test

.PHONY: integration-test
integration-test:
# integration tests parallelize less well (hence -p 1)
	go test ./... -v --tags=integration -p 1

.PHONY: grc-test
grc-test:
	grc go test ./... -v
.PHONY: grc-test-short
grc-test-short:
	grc go test ./... -test.short -v

.PHONY: test-debug
test-debug:
	LOG_LEVEL=debug go test ./... -v

.PHONY: grc-test-debug
grc-test-debug:
	LOG_LEVEL=debug grc go test ./... -v

.PHONY: test-one
test-one:
	go test -v -count 1 -timeout 3000s -run ^$(TEST)$$ github.com/filecoin-project/bacalhau/cmd/bacalhau/

.PHONY: test-devstack
test-devstack:
	go test -v -count 1 -timeout 3000s -run '^Test\w+Suite$$' github.com/filecoin-project/bacalhau/pkg/test/devstack/

.PHONY: test-commands
test-commands:
	go test -v -count 1 -timeout 3000s -run '^Test\w+Suite$$' github.com/filecoin-project/bacalhau/cmd/bacalhau/

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
COMMA = ,
COVER_FILE := coverage/${PACKAGE}_$(subst ${COMMA},_,${TEST_BUILD_TAGS}).coverage

.PHONY: test-and-report
test-and-report: unittests.xml ${COVER_FILE}

${COVER_FILE} unittests.xml ${TEST_OUTPUT_FILE_PREFIX}_unit.json: ${BINARY_PATH} $(dir ${COVER_FILE})
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
generate:
	${GO} generate -gcflags '-N -l' -ldflags "-X main.VERSION=$(TAG)" ./...
	echo "[OK] Files added to pipeline template directory!"

.PHONY: security
security:
	gosec -exclude=G204,G304 -exclude-dir=test ./...
	echo "[OK] Go security check was completed!"

release: build-bacalhau
	cp bin/bacalhau .
