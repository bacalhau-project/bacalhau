GO:="go"
GOOS:=`go env GOOS`
GOARCH:=`go env GOARCH | perl -pe 's/^(armv[67])$$/arm/'`
GOARM:= `echo ${GOARCH} | perl -ne 'if (/^armv6$$/) { print "6" } elsif (/^armv7$$/) { print "7" }'`
UNAME_S := "$(uname -s)"

# Env Variables
export GO111MODULE := "on"
export CGO_ENABLED := "0"

BUILD_DIR:="bacalhau"

BINARY_NAME := `if [ $GOOS = "windows" ]; then echo "bacalhau.exe"; else echo "bacalhau"; fi`
CC := `if [ $GOOS = "windows" ]; then echo "gcc.exe"; else echo "gcc"; fi`

BINARY_PATH:="bin/${GOOS}/${GOARCH}${GOARM}/${BINARY_NAME}"

TAG:=`git describe --tags --always`
home_dir := env_var_or_default('COMMIT', `git rev-parse HEAD`)
REPO := `basename -s .git $(git config --get remote.origin.url)`
BRANCH := `git rev-parse --abbrev-ref HEAD`
BUILDDATE := env_var_or_default('BUILDDATE', `date -u +'%Y-%m-%dT%H:%M:%SZ'`)
PACKAGE := "bacalhau_${TAG}_${GOOS}_${GOARCH}${GOARM}"
TEST_BUILD_TAGS := "unit,integration"
TEST_PARALLEL_PACKAGES := "1"

PRIVATE_KEY_FILE := "/tmp/private.pem"
PUBLIC_KEY_FILE := "/tmp/public.pem"

export MAKE:=`command -v make 2> /dev/null`
export JUST:=`command -v just 2> /dev/null`

BUILD_FLAGS:="-X github.com/bacalhau-project/bacalhau/pkg/version.GITVERSION={{TAG}}"

# pypi version scheme (https://peps.python.org/pep-0440/) does not accept
# versions with dashes (e.g. 0.3.24-build-testing-01), so we replace them a valid suffix
export GIT_VERSION_COMMAND := "git describe --tags --dirty"
export PYPI_VERSION_SCRIPT := "python3 scripts/convert_git_version_to_pep440_compatible.py"

all: build

# Run init repo after cloning it
init:
	#!/usr/bin/env bash
	echo "Activating repo with flox..."
	flox activate -r "aronchick/bacalhau" 2>/dev/null
	if [ $? -eq 1 ]; then
		echo "Already active."
	fi

pre-commit:
	#!/usr/bin/env bash
	mkdir -p webui/build && touch webui/build/stub
	pre-commit run --all
	rm webui/build/stub
	cd python && flox activate -r "aronchick/python" -- just pre-commit

build:
	#!/usr/bin/env bash
	go build -ldflags ${BUILD_FLAGS} -o ${BINARY_PATH} cmd/bacalhau/main.go

build-python-apiclient:
	#!/usr/bin/env bash
	cd clients && flox activate -r "aronchick/clients" -- just clean all
	echo "Python API client built."

build-python-sdk:
	#!/usr/bin/env bash
	export PYPI_VERSION=$({{ PYPI_VERSION_SCRIPT }} $({{ GIT_VERSION_COMMAND }}))
	cd python
	flox activate -r "aronchick/python" -- just build
	echo "Python SDK built."

build-bacalhau-airflow:
	cd integration/airflow
	flox activate -r "aronchick/airflow" -- just clean all
	echo "Python bacalhau-airflow built."
