.PHONY: install-piptools
install-piptools:
	# pip 22.1 broke pip-tools: https://github.com/jazzband/pip-tools/issues/1617
	python -m pip install -U pip-tools setuptools wheel "pip>=22.0.3,!=22.1"
	
.PHONY: setup
setup: install-piptools ## Install requirements
	pip install -r dev-requirements.in

.PHONY: test
test: ## Run tests
	cd plugins && $(MAKE) test

# .PHONY: fmt
# fmt: ## Format code with black and isort
# 	autoflake --remove-all-unused-imports --ignore-init-module-imports --ignore-pass-after-docstring --in-place -r plugins
# 	# pre-commit run black --all-files || true
# 	# pre-commit run isort --all-files || true

# .PHONY: lint
# lint: ## Run linters
# 	mypy plugins
# 	# allow-empty-bodies: Allow empty body in function.
# 	# disable-error-code="annotation-unchecked": Remove the warning "By default the bodies of untyped functions are not checked".
# 	# Mypy raises a warning because it cannot determine the type from the dataclass, despite we specified the type in the dataclass.
# 	mypy --allow-empty-bodies --disable-error-code="annotation-unchecked" tests/flytekit/unit/core
# 	# pre-commit run --all-files
