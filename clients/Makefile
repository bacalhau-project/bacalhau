PACKAGE_NAME := bacalhau-apiclient
SWAGGER_JSON := ../docs/swagger.json
PYPI_VERSION ?= $(or ${PYPI_VERSION}, $(shell ../../../scripts/get_current_version_for_builds.py --pep440))
ifeq ($(PYPI_VERSION),)
	PYPI_VERSION := 0.0.0+develop
endif

EARTHLY := $(shell command -v earthly --push 2> /dev/null)
MAKE := $(shell command -v make 2> /dev/null)
CP := $(shell command -v cp 2> /dev/null)
RM := $(shell command -v rm 2> /dev/null)

.PHONY: all
all:
	$(CP) ${SWAGGER_JSON} ./swagger.json
	${MAKE} clean && ${EARTHLY} --push +all --PACKAGE_NAME=${PACKAGE_NAME} --SWAGGER_JSON=${SWAGGER_JSON} --VERSION=${PYPI_VERSION}
	@echo "Python API client built."
	$(RM) ./swagger.json

.PHONY: clean
clean:
	$(RM) -rf ./python
	$(RM) -f python-config.json
	$(RM) -f ./swagger-edited*.json
	mkdir -p python
