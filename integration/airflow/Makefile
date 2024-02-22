.PHONY: clean clean-build clean-pyc clean-test coverage dist docs help install lint lint/flake8 lint/black setup
.DEFAULT_GOAL := help
PYPI_VERSION ?= $(or ${PYPI_VERSION}, $(shell ../../../scripts/get_current_version_for_builds.py --pep440))
ifeq ($(PYPI_VERSION),)
	PYPI_VERSION := 0.0.0+develop
endif

EARTHLY ?= $(shell command -v earthly 2> /dev/null)

define BROWSER_PYSCRIPT
import os, webbrowser, sys

from urllib.request import pathname2url

webbrowser.open("file://" + pathname2url(os.path.abspath(sys.argv[1])))
endef
export BROWSER_PYSCRIPT

define PRINT_HELP_PYSCRIPT
import re, sys

for line in sys.stdin:
	match = re.match(r'^([a-zA-Z_-]+):.*?## (.*)$$', line)
	if match:
		target, help = match.groups()
		print("%-20s %s" % (target, help))
endef
export PRINT_HELP_PYSCRIPT

BROWSER := python -c "$$BROWSER_PYSCRIPT"

all:
	$(MAKE) clean && $(EARTHLY) --push +dist
	@echo "Airflow plugin built."

help:
	@python3 -c "$$PRINT_HELP_PYSCRIPT" < $(MAKEFILE_LIST)

setup:
	pip3 install -r dev-requirements.txt

clean: clean-build clean-pyc clean-test ## remove all build, test, coverage and Python artifacts

clean-build: ## remove build artifacts
	rm -fr build/
	rm -fr dist/
	rm -fr .eggs/
	find . -name '*.egg-info' -exec rm -fr {} +
	find . -name '*.egg' -exec rm -f {} +

clean-pyc: ## remove Python file artifacts
	find . -name '*.pyc' -exec rm -f {} +
	find . -name '*.pyo' -exec rm -f {} +
	find . -name '*~' -exec rm -f {} +
	find . -name '__pycache__' -exec rm -fr {} +

clean-test: ## remove test and coverage artifacts
	rm -fr .tox/
	rm -f .coverage
	rm -fr htmlcov/
	rm -fr .pytest_cache

lint:
	$(EARTHLY) --push +run-lint

lint-local: ruff --fix ## check style locally (outside of Earthly)

test:
	$(EARTHLY) --push +run-tests

test-local: ## run tests quickly with the default Python
	python3 setup.py test

test-all: ## run tests on every Python version with tox
	$(EARTHLY) --push +run-tests

test-all-local: ## run tests on every Python version with tox
	tox

coverage: ## check code coverage quickly with the default Python
	coverage run --source bacalhau_airflow setup.py test
	coverage report -m
	coverage html
	$(BROWSER) htmlcov/index.html

docs: ## generate Sphinx HTML documentation, including API docs
	rm -f docs/bacalhau_airflow.rst
	rm -f docs/modules.rst
	$(MAKE) -C docs clean # clean the docs
	$(EARTHLY) --push +make-docs

servedocs: docs ## compile the docs watching for changes
	watchmedo shell-command -p '*.rst' -c '$(MAKE) -C docs html' -R -D .

.SILENT: release
release: ## upload a release
	$(EARTHLY) --push +release --PYPI_VERSION=${PYPI_VERSION}

dist: clean ## builds source and wheel package
	$(EARTHLY) --push +dist --PYPI_VERSION=${PYPI_VERSION}

install: clean ## install the package to the active Python's site-packages
	pip3 install .
