POETRY := $(shell command -v poetry 2> /dev/null)

sources = bacalhau_sdk

.PHONY: test
test: format lint unittest

.PHONY: format
format:
	$(POETRY) run isort $(sources) tests
	$(POETRY) run black $(sources) tests

.PHONY: lint
lint:
	$(POETRY) run flake8 $(sources) tests
	$(POETRY) run mypy $(sources) tests

.PHONY: unittest
unittest:
	$(POETRY) run pytest --capture=no --log-cli-level=DEBUG

.PHONY: coverage
coverage:
	$(POETRY) run pytest --cov=$(sources) --cov-branch --cov-report=term-missing tests

.PHONY: pre-commit
pre-commit:
	$(POETRY) run pre-commit run --all-files

.PHONY: clean
clean:
	rm -rf .mypy_cache .pytest_cache
	rm -rf *.egg-info
	rm -rf .tox dist site
	rm -rf coverage.xml .coverage
