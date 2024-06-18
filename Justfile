export GO:="go"
export GOOS:=`go env GOOS`
export GOARCH:=`go env GOARCH | perl -pe 's/^(armv[67])$$/arm/'`
UNAME_S := `uname -s`

GOARM:= `echo $GOARCH | perl -ne 'if (/^armv6$$/) { print "6" } elsif (/^armv7$$/) { print "7" }'`

# Env Variables
export GO111MODULE := "on"
export CGO_ENABLED := "0"
export PRECOMMIT:="poetry run pre-commit"

BUILD_DIR:="bacalhau"

BINARY_NAME := `if [ $GOOS = "windows" ]; then echo "bacalhau.exe"; else echo "bacalhau"; fi`
CC := `if [ $GOOS = "windows" ]; then echo "gcc.exe"; else echo "gcc"; fi`

BINARY_PATH:="bin/${GOOS}/${GOARCH}${GOARM}/${BINARY_NAME}"

TAG:=`git describe --tags --always`
# COMMIT ?= $(eval COMMIT := $(shell git rev-parse HEAD))$(COMMIT)
# REPO ?= $(shell echo $$(cd ../${BUILD_DIR} && git config --get remote.origin.url) | $(SED) 's/git@\(.*\):\(.*\).git$$/https:\/\/\1\/\2/')
# BRANCH ?= $(shell cd ../${BUILD_DIR} && git branch | grep '^*' | awk '{print $$2}')
# BUILDDATE ?= $(eval BUILDDATE := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ'))$(BUILDDATE)
# PACKAGE := $(shell echo "bacalhau_$(TAG)_${GOOS}_$(GOARCH)${GOARM}")
# TEST_BUILD_TAGS ?= unit,integration
# TEST_PARALLEL_PACKAGES ?= 1

PRIVATE_KEY_FILE := "/tmp/private.pem"
PUBLIC_KEY_FILE := "/tmp/public.pem"

export MAKE:=`command -v make 2> /dev/null`
export JUST:=`command -v just 2> /dev/null`

BUILD_FLAGS:="-X github.com/bacalhau-project/bacalhau/pkg/version.GITVERSION={{TAG}}"

# pypi version scheme (https://peps.python.org/pep-0440/) does not accept
# versions with dashes (e.g. 0.3.24-build-testing-01), so we replace them a valid suffix
GIT_VERSION := `git describe --tags --dirty`
PYPI_VERSION := `python3 scripts/convert_git_version_to_pep440_compatible.py ` + GIT_VERSION

all: build

# Run init repo after cloning it
init:
	@ops/repo_init.sh 1>/dev/null
	@echo "Build environment initialized."

build:
