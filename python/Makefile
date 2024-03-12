export EARTHLY ?= $(shell command -v earthly 2> /dev/null)
export POETRY ?= $(shell command -v poetry 2> /dev/null)

# Should come from global make file
VERSION ?= $(or ${PYPI_VERSION}, $(shell ../scripts/get_current_version_for_builds.py --pep440))
ifeq ($(VERSION),)
	VERSION := 0.0.0+develop
endif

sources = bacalhau_sdk

.PHONY: all
all: publish

.PHONY: setup
setup:
	$(EARTHLY) --push +install-deps
	$(EARTHLY) --push +install-precommit
	$(EARTHLY) --push +check-python-version-is-same-as-in-pyproject
	$(EARTHLY) --push +copy-code

.PHONY: test
test: unittest coverage

.PHONY: ruff
ruff:
	$(EARTHLY) --push +run-linter

.PHONY: unittest
unittest:
	$(EARTHLY) --push +run-unittest

.PHONY: coverage
coverage:
	$(EARTHLY) --push +run-coverage

.PHONY: install-pre-commit
install-pre-commit:
	$(POETRY) run pre-commit install
	@echo "Pre-commit installed."

.PHONY: pre-commit
pre-commit: install-pre-commit
	$(POETRY) run pre-commit run --files ./**/*

.PHONY: clean
clean:
	rm -rf .mypy_cache .pytest_cache
	rm -rf *.egg-info
	rm -rf .tox dist site
	rm -rf coverage.xml .coverage

.PHONY: build
build:
	# Echo PYPI_VERSION to .arg file
	echo "PYPI_VERSION=$(VERSION)" > .arg

	# Get PYPI_TOKEN from environment or .env file and push to .secret/ file
	python get_pypi_token.py

	$(EARTHLY) --push +build

.SILENT: publish
.PHONY: publish
publish:
	# Echo PYPI_VERSION to .arg file
	echo "PYPI_VERSION=$(VERSION)" > .arg

	# Get PYPI_TOKEN from environment or .env file and push to .secret/ file
	python ../scripts/get_pypi_token.py

	$(EARTHLY) --push +publish --PYPI_VERSION=$(VERSION)
