set export
MAKE:=`command -v make 2> /dev/null`
JUST:=`command -v just 2> /dev/null`
GO:=`command -v go 2> /dev/null`
RM:=`command -v rm 2> /dev/null`

GOOS:=`go env GOOS`
GOARCH:=(```
python3 -c "
import os
goarch = os.popen('go env GOARCH').read().strip()
if goarch == 'armv6':
	print(f'{goarch}arm6')
elif goarch == 'armv7':
	print(f'{goarch}arm7')
else:
	print(goarch)"
```)
UNAME_S := `uname -s`

# Env Variables
GO111MODULE := "on"
CGO_ENABLED := "0"
BUILD_DIR:="bacalhau"
BINARY_NAME := (```
python3 -c "
import os
goos = os.popen('go env GOOS').read().strip()
if goos == 'windows':
	print('bacalhau.exe')
else:
	print('bacalhau')
"
```)
CC := (```
python3 -c "
import os
goos = os.popen('go env GOOS').read().strip()
if goos == 'windows':
	print('gcc.exe')
else:
	print('gcc')
"```)

BINARY_PATH:="bin/"+GOOS+"/"+GOARCH+"/"+BINARY_NAME

TAG:=`git describe --tags --always`
home_dir := env_var_or_default('COMMIT', `git rev-parse HEAD`)
REPO := `basename -s .git $(git config --get remote.origin.url)`
BRANCH := `git rev-parse --abbrev-ref HEAD`
BUILDDATE := env_var_or_default('BUILDDATE', `date -u +'%Y-%m-%dT%H:%M:%SZ'`)
PACKAGE := "bacalhau_" + TAG + "_" + GOOS + "_" + GOARCH
TEST_BUILD_TAGS := "unit,integration"
TEST_PARALLEL_PACKAGES := "4"

PRIVATE_KEY_FILE:=`echo ${PRIVATE_KEY_FILE:-${TEST_PRIVATE_KEY_FILE:-/tmp/private.pem}}`
PUBLIC_KEY_FILE:="${PUBLIC_KEY_FILE:-${TEST_PUBLIC_KEY_FILE:-/tmp/public.pem}}"
BUILD_FLAGS:="-X github.com/bacalhau-project/bacalhau/pkg/version.GITVERSION={{TAG}}"

# pypi version scheme (https://peps.python.org/pep-0440/) does not accept
# versions with dashes (e.g. 0.3.24-build-testing-01), so we replace them a valid suffix
GIT_VERSION_COMMAND := "git describe --tags --dirty"
PYPI_VERSION_SCRIPT := "python3 scripts/convert_git_version_to_pep440_compatible.py"

all: build

# Run init repo after cloning it
init:
	#!/usr/bin/env bash
	echo "Activating repo with flox..."
	flox activate -r "aronchick/bacalhau" -t 2>/dev/null
	if [ $? -eq 1 ]; then
		echo "Already active."
	fi

pre-commit:
	#!/usr/bin/env bash
	mkdir -p webui/build && touch webui/build/stub
	pre-commit run --all
	rm webui/build/stub
	cd python && flox activate -r "aronchick/python" -t -- just pre-commit

generate_public_key_pair:
	#!/usr/bin/env bash
	python scripts/generate_public_key_pair.py

build-python-apiclient:
	#!/usr/bin/env bash
	cd clients && flox activate -r "aronchick/clients" -t -- just clean all
	echo "Python API client built."

build-python-sdk:
	#!/usr/bin/env bash
	export PYPI_VERSION=$({{ PYPI_VERSION_SCRIPT }} $({{ GIT_VERSION_COMMAND }}))
	cd python
	flox activate -r "aronchick/python" -t -- just clean build
	echo "Python SDK built."

build-bacalhau-airflow:
	#!/usr/bin/env bash
	cd integration/airflow
	flox activate -r "aronchick/airflow" -t -- just clean all
	echo "Python bacalhau-airflow built."

build-python: build-python-apiclient build-python-sdk build-bacalhau-airflow

release-python-apiclient: build-python-apiclient
	#!/usr/bin/env bash
	cd clients
	flox activate -r "aronchick/clients" -t -- just publish
	echo "Python API client pushed to PyPi."

release-python-sdk: build-python-sdk
	#!/usr/bin/env bash
	cd python
	flox activate -r "aronchick/python" -t -- just publish
	echo "Python SDK pushed to PyPi."

release-bacalhau-airflow: build-bacalhau-airflow
	#!/usr/bin/env bash
	cd integration/airflow
	flox activate -r "aronchick/airflow" -t -- just release
	echo "Python bacalhau-airflow pushed to PyPi."

################################################################################
# Target: build
################################################################################

build: build-bacalhau

build-ci: build-bacalhau

build-dev: build-ci
	#!/usr/bin/env bash
	sudo cp ${BINARY_PATH} /usr/local/bin

################################################################################
# Target: build-bacalhau
################################################################################

# This is a recipe that lists the files in the cmd directory excluding deleted files
cmd_files := `bash -c 'comm -23 <(git ls-files cmd | sort) <(git ls-files cmd --deleted | sort)'`

# This is a recipe that lists the files in the pkg directory excluding deleted files
pkg_files := `bash -c 'comm -23 <(git ls-files pkg | sort) <(git ls-files pkg --deleted | sort)'`

# This is a recipe that lists the source files in the webui directory excluding certain paths
web_src_files := `find webui -not -path 'webui/build/*' -not -path 'webui/build' -not -path 'webui/node_modules/*' -not -name '*.go'`

# This is a recipe that lists the Go files in the webui directory excluding certain paths
web_go_files := `find webui -not -path 'webui/build/*' -not -path 'webui/build' -not -path 'webui/node_modules/*' -name '*.go'`

# The binary recipe depends on cmd_files, pkg_files, and main.go
binary:
	@go build -ldflags "{{BUILD_FLAGS}}" -trimpath -o "{{BINARY_PATH}}" .

# Define the cmd_files dependency
cmd_files:
	@echo "{{cmd_files}}"

# Define the pkg_files dependency
pkg_files:
	@echo "{{pkg_files}}"

# Define the build-webui recipe
build-webui:
	@#!/usr/bin/env bash
	@cd webui && flox activate -r "aronchick/webui" -t -- just all

# Define the binary-web recipe
binary-web: build-webui
	@echo "Building binary-web with the following Go files:"
	@echo "{{web_go_files}}"

# This is the main target that depends on binary-web and binary
build-bacalhau: binary-web binary
	#!/usr/bin/env bash
	echo "Built bacalhau binary."
	echo "Executing ${BINARY_PATH} --version"
	${BINARY_PATH} --version


# Define the web_go_files dependency
web_go_files:
	@echo "{{web_go_files}}"

# ################################################################################
# # Target: build-docker-images
# ################################################################################
HTTP_GATEWAY_IMAGE := "ghcr.io/bacalhau-project/http-gateway"
HTTP_GATEWAY_TAG := TAG
build-http-gateway-image:
	#!/usr/bin/env bash
	echo -n "Building http-gateway image..."
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		-t ${HTTP_GATEWAY_IMAGE}:${HTTP_GATEWAY_TAG} \
		pkg/executor/docker/gateway
	echo "done."
# Define the main variables
BACALHAU_IMAGE := "docker.io/bacalhauproject/bacalhau"
BACALHAU_GCP_IMAGE := "gcr.io/bacalhau-project/bacalhau"
BACALHAU_TAG := TAG

IS_LATEST_TAG := if BACALHAU_TAG =~ '^v[0-9]+\.[0-9]+\.[0-9]+$' { "true" } else { "false" }

LATEST_TAG := if IS_LATEST_TAG == "true" { "--tag " + BACALHAU_IMAGE + ":latest"} else { "" }
LATEST_GCP_TAG := if IS_LATEST_TAG == "true" { "--tag " + BACALHAU_GCP_IMAGE + ":latest"} else { "" }

BACALHAU_BUILD_FLAGS := "--progress=plain \
		--platform linux/amd64,linux/arm64 \
		--tag " + BACALHAU_IMAGE +":" + BACALHAU_TAG +" \
		--tag " + BACALHAU_GCP_IMAGE +":" + BACALHAU_TAG +" \
		"+ LATEST_TAG +" \
		"+ LATEST_GCP_TAG +"  \
		--label org.opencontainers.artifact.created=" + `date -u +'%Y-%m-%dT%H:%M:%SZ'` + "\
		--label org.opencontainers.image.version="+BACALHAU_TAG +" \
		--cache-from=type=registry,ref=" + BACALHAU_IMAGE +":latest \
		--file docker/bacalhau-image/Dockerfile \
		."

build-bacalhau-image:
	@#!/usr/bin/env bash
	docker buildx build {{ BACALHAU_BUILD_FLAGS }}

push-bacalhau-image:
	@#!/usr/bin/env bash
	docker buildx build --push {{ BACALHAU_BUILD_FLAGS }}

build-docker-images: build-http-gateway-image

push-docker-images: build-http-gateway-image

# Release tarballs suitable for upload to GitHub release pages
################################################################################
# Target: build-bacalhau-tgz
################################################################################
build-bacalhau-tgz:
	#!/usr/bin/env bash
	# Make sure the dist directory exists
	PACKAGE_NAME={{PACKAGE}}.tar.gz
	SIGNATURE_FILE=${PACKAGE_NAME}.sha256
	mkdir -p dist
	tar cvzf "dist/${PACKAGE_NAME}" -C {{parent_directory(BINARY_PATH)}} {{BINARY_NAME}}
	echo "Private key file: {{PRIVATE_KEY_FILE}}"
	openssl dgst -sha256 -sign {{PRIVATE_KEY_FILE}} -passin pass:"${PRIVATE_KEY_PASSPHRASE}" dist/"${PACKAGE_NAME}" | openssl base64 -out dist/${SIGNATURE_FILE}


################################################################################
# Target: clean
################################################################################
clean:
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
test: unit-test bash-test

unit-test:
	go test ./... -v --tags=unit

test-python-sdk:
	cd python && ${JUST} test

integration-test:
	go test ./... -v --tags=integration -p 1

bash-test:
	cd test && bin/bashtub *.sh

################################################################################
# Target: test-debug
################################################################################
test-debug:
	LOG_LEVEL=debug go test ./... -v

test-one:
	go test -v -count 1 -timeout 3000s -run ^$(TEST)$$ github.com/bacalhau-project/bacalhau/cmd/bacalhau/

test-devstack:
	go test -v -count 1 -timeout 3000s -run '^Test\w+Suite$$' github.com/bacalhau-project/bacalhau/pkg/test/devstack/

test-commands:
	go test -v -count 1 -timeout 3000s -run '^Test\w+Suite$$' github.com/bacalhau-project/bacalhau/cmd/bacalhau/

test-all: test test-python-sdk
	cd webui && yarn run build && yarn run lint && yarn run test

################################################################################
# Target: devstack
################################################################################
devstack:
	go run . devstack

devstack-one:
	IGNORE_PID_AND_PORT_FILES=true PREDICTABLE_API_PORT=1 go run . devstack --requester-nodes 0 --compute-nodes 0 --hybrid-nodes 1

devstack-100:
	go run . devstack --compute-nodes 100

devstack-250:
	go run . devstack --compute-nodes 250

devstack-20:
	go run . devstack --compute-nodes 20

devstack-noop:
	go run . devstack --noop

devstack-noop-100:
	go run . devstack --noop --compute-nodes 100

devstack-race:
	go run -race . devstack

devstack-badactor:
	go run . devstack --bad-compute-actors 1

################################################################################
# Target: lint
################################################################################
lint:
	golangci-lint run --timeout 10m

lint-fix:
	golangci-lint run --timeout 10m

################################################################################
# Target: modtidy
################################################################################
modtidy:
	go mod tidy

################################################################################
# Target: check-diff
################################################################################
check-diff:
	git diff --exit-code ./go.mod # check no changes
	git diff --exit-code ./go.sum # check no changes

# Run the unittests and output results for recording
################################################################################
# Target: test-test-and-report
################################################################################
# Define variables
COVERAGE_DIR := "coverage"
TEST_OUTPUT_FILE_PREFIX := "test_output"

# Replace comma with underscore in build tags for coverage file name
COVER_FILE := replace("COVERAGE_DIR" + "/" + PACKAGE + "_" + TEST_BUILD_TAGS + ".coverage", ",", "_")

# Recipe to run tests and generate coverage and report files
# Target: test-and-report
test-and-report:
	#!/usr/bin/env bash
	mkdir -p ${COVERAGE_DIR}

	gotestsum \
		--jsonfile "${COVERAGE_DIR}/${TEST_OUTPUT_FILE_PREFIX}_unit.json" \
		--junitfile unittests.xml \
		--format testname \
		-- \
			-p ${TEST_PARALLEL_PACKAGES} \
			./pkg/... ./cmd/... \
			-coverpkg=./... -coverprofile="${COVERAGE_DIR}/${PACKAGE}_${TEST_BUILD_TAGS//,/_}.coverage" \
			--tags=${TEST_BUILD_TAGS}

################################################################################
# Target: coverage-report
################################################################################
coverage-report:
	#!/usr/bin/env bash
	COVERAGE_OUT="${COVERAGE_DIR}/coverage.out"
	COVERAGE_HTML="${COVERAGE_DIR}/coverage.html"

	# Target: coverage-report
	echo "Building coverage report..."

	# Create coverage directory if it doesn't exist
	mkdir -p "${COVERAGE_DIR}"

	# Generate coverage.out using gocovmerge
	gocovmerge $(find coverage -name "*.coverage") > "${COVERAGE_OUT}"

	# Generate coverage.html using go tool cover
	go tool cover -html="${COVERAGE_OUT}" -o "${COVERAGE_HTML}"

	echo "Coverage report generated: ${COVERAGE_HTML}"

# Target: generate
generate: generate-tools
	${GO} generate -gcflags '-N -l' -ldflags "-X main.VERSION=${TAG}" ./...
	@echo "[OK] Files added to pipeline template directory!"

# Target: generate-tools
generate-tools:
	@which mockgen > /dev/null || (echo "Installing 'mockgen'"; ${GO} install go.uber.org/mock/mockgen@latest)
	@which stringer > /dev/null || (echo "Installing 'stringer'"; ${GO} install golang.org/x/tools/cmd/stringer)

# Target: security
security:
	gosec -exclude=G204,G304 -exclude-dir=test ./...
	echo "[OK] Go security check was completed!"

# Target: release
release: build-bacalhau
	cp bin/bacalhau .

# ifeq ($(OS),Windows_NT)
#     detected_OS := Windows
# else
#     detected_OS := $(shell sh -c 'uname 2>/dev/null || echo Unknown')
# endif

# # TODO make the plugin path configurable instead of using the bacalhau config path.
# BACALHAU_CONFIG_PATH := $(shell echo $$BACALHAU_PATH)
# INSTALL_PLUGINS_DEST := $(if $(BACALHAU_CONFIG_PATH),$(BACALHAU_CONFIG_PATH)plugins/,~/.bacalhau/plugins/)

# EXECUTOR_PLUGINS := $(wildcard ./pkg/executor/plugins/executors/*/.)

# # TODO fix install on windows
# ifeq ($(detected_OS),Windows)
#     build-plugins clean-plugins install-plugins:
# 	@echo "Skipping executor plugins on Windows"
# else
#     build-plugins: plugins-build
#     clean-plugins: plugins-clean
#     install-plugins: plugins-install

#     .PHONY: plugins-build $(EXECUTOR_PLUGINS)

#     plugins-build: $(EXECUTOR_PLUGINS)
# 	@echo "Building executor plugins..."
# 	@$(foreach plugin,$(EXECUTOR_PLUGINS),$(MAKE) --no-print-directory -C $(plugin) &&) true

#     .PHONY: plugins-clean $(addsuffix .clean,$(EXECUTOR_PLUGINS))

#     plugins-clean: $(addsuffix .clean,$(EXECUTOR_PLUGINS))
# 	@echo "Cleaning executor plugins..."
# 	@$(foreach plugin,$(addsuffix .clean,$(EXECUTOR_PLUGINS)),$(MAKE) --no-print-directory -C $(basename $(plugin)) clean &&) true

#     .PHONY: plugins-install $(addsuffix .install,$(EXECUTOR_PLUGINS))

#     plugins-install: plugins-build $(addsuffix .install,$(EXECUTOR_PLUGINS))
# 	@echo "Installing executor plugins..."
# 	@$(foreach plugin,$(addsuffix .install,$(EXECUTOR_PLUGINS)),mkdir -p $(INSTALL_PLUGINS_DEST) && cp $(basename $(plugin))/bin/* $(INSTALL_PLUGINS_DEST) &&) true
# endif

spellcheck-code:  ## Runs a spellchecker over all code - MVP just does one file
	cspell -c .cspell-code.json lint ./pkg/authn/**

spellcheck-docs:  ## Runs a spellchecker over all documentation - MVP just does one directory
	cspell -c .cspell-docs.json lint ./docs/docs/dev/**
