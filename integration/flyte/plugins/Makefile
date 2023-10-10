VERSION ?= $${PYPI_VERSION}

UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
export SED = gsed
endif
ifeq ($(UNAME_S),Linux)
export SED = sed
endif

# make sure tests don't point to production api
# https://github.com/bacalhau-project/bacalhau/issues/2710
export BACALHAU_API_HOST=myhost
export BACALHAU_API_PORT=99999

.PHONY: test
test:
	pip install -e flytekit-bacalhau
	find . -maxdepth 1 -type d | grep 'flytekit-' | xargs -L1 python3 -m pytest --log-cli-level=DEBUG --capture=no

.PHONY: build_all_plugins
build_all_plugins:
	./run_all_plugins.sh python3 setup.py sdist bdist_wheel

.PHONY: publish_all_plugins
publish_all_plugins:
	twine upload --verbose */dist/* -u __token__ -p ${PYPI_TOKEN}

.PHONY: all_requirements
all_requirements:
	./run_all_plugins.sh pip-compile requirements.in --upgrade --verbose

PLACEHOLDER := "__version__\ =\ \"0.0.0+develop\""
VERSION_FILE := "setup.py"


.PHONY: update_all_versions
update_all_versions:
	# ensure the placeholder is there. If grep doesn't find the placeholder
	# it exits with exit code 1 and github actions aborts the build.
	./run_all_plugins.sh grep "$(PLACEHOLDER)" "$(VERSION_FILE)"
	./run_all_plugins.sh $(SED) -i "s/$(PLACEHOLDER)/__version__ = \"${VERSION}\"/g" $(VERSION_FILE)


#### Bacalhau additions ####

.PHONY: docker-build
docker-build:
	docker build -t winderresearch/flytekit-bacalhau:${VERSION} -f Dockerfile .

.PHONY: docker-run
docker-run:
	docker run -it --rm -p 8000:8000 winderresearch/flytekit-bacalhau:${VERSION}

.PHONY: docker-push
docker-push:
	docker push winderresearch/flytekit-bacalhau:${VERSION}