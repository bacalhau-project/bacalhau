export EARTHLY := $(shell command -v earthly 2> /dev/null)

.PHONY: test-to-see-if-earthly-is-installed
test-to-see-if-earthly-is-installed:
ifndef EARTHLY
	$(error "earthly is not available please install it")
else
	@echo "earthly is installed"
endif

.PHONY: install-piptools
install-piptools: test-to-see-if-earthly-is-installed
	earthly --push +install-piptools

.PHONY: setup-development
setup-development: install-piptools ## Install requirements
	python3 -m pip install -r dev-requirements.in

.PHONY: test
test: ## Run tests
	cd plugins && $(MAKE) test

.PHONY: fmt
fmt:
	ruff format . --config ruff.toml

.PHONY: lint
lint: ## Run linters
	ruff . --config ruff.toml

.PHONY: spellcheck
spellcheck:  ## Runs a spellchecker over all code and documentation
	codespell -L "te,raison,fo" --skip="./docs/build,./.git" --ignore-words="./.cspell/custom-dictionary.txt"

.SILENT: release
release: ## upload a release
	twine upload --verbose dist/* -u __token__ -p ${PYPI_TOKEN}
