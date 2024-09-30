PACKAGE_NAME := bacalhau-apiclient
SWAGGER_JSON := ../docs/swagger.json
VERSION ?= $(or ${PYPI_VERSION}, $(shell ../scripts/get_current_version_for_builds.py --pep440))
ifeq ($(VERSION),)
	VERSION := 0.0.0+develop
endif

EARTHLY := $(shell command -v earthly --push 2> /dev/null)
MAKE := $(shell command -v make 2> /dev/null)
CP := $(shell command -v cp 2> /dev/null)
RM := $(shell command -v rm 2> /dev/null)

.PHONY: release
release:
	# Echo PYPI_VERSION to .arg file
	echo "PYPI_VERSION=$(VERSION)" > .arg

	# Get PYPI_TOKEN from environment or .env file and push to .secret/ file
	python3 ../scripts/get_pypi_token.py

	$(CP) ${SWAGGER_JSON} ./swagger.json
	${MAKE} clean && ${EARTHLY} +pypi-upload --PACKAGE_NAME=${PACKAGE_NAME} --SWAGGER_JSON=${SWAGGER_JSON} --VERSION=${VERSION} --RELEASE_PYTHON_PACKAGES=1
	@echo "Python API client released."

.PHONY: all
all:
	$(CP) ${SWAGGER_JSON} ./swagger.json
	${MAKE} clean && ${EARTHLY} +all --PACKAGE_NAME=${PACKAGE_NAME} --SWAGGER_JSON=${SWAGGER_JSON} --VERSION=${VERSION}
	@echo "Python API client built."
	$(RM) ./swagger.json

.PHONY: clean
clean:
	$(RM) -rf ./python
	$(RM) -f python-config.json
	$(RM) -f ./swagger-edited*.json
	mkdir -p python
