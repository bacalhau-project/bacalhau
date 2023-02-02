PACKAGE_NAME := bacalhau-apiclient
SWAGGER_JSON := ../docs/swagger.json
VERSION := $(shell git describe --tags --abbrev=0 | tr -d v).post4

#set cmd to be swagger-codegen if that doesnt exist make it the java command
#swagger-codegen is supported by brew install swagger-codegen@2
#the java command is supported by the docker image swaggerapi/swagger-codegen-cli
SWAGGER ?= swagger-codegen
ifeq ($(shell which ${SWAGGER}),)
SWAGGER := $(shell echo "${HOME}/bin/swagger-codegen-cli.jar")
ifeq ($(shell ls -1 ${SWAGGER}),)
$(info No swagger-codegen found?)
endif
endif

.PHONY: all
all: $(patsubst %,%/,$(shell cat supported_langs)) pypi-build pypi-upload
	

%-config.json: config.json.template
	cat $< | \
		sed -e "s/VERSION/${VERSION}/g" | \
		sed -e "s/PACKAGE-NAME/${PACKAGE_NAME}/g" > $@

UNDERSCORE_PACKAGE_NAME=$(subst -,_,${PACKAGE_NAME})

.INTERMEDIATE: python-config.json
python-config.json: config.json.template
	cat $< | \
		sed -e "s/VERSION/${VERSION}/g" | \
		sed -e "s/PACKAGE-NAME/${UNDERSCORE_PACKAGE_NAME}/g"> $@

python/: python-config.json ${SWAGGER_JSON}
	# swagger codegen names the generated methods after the related endpoint @ID annotation (e.g. pkg/requester/publicapi/submit)
	# this results into a names that are not quite user-friendly (e.g. pkgrequesterpublicapisubmit)
	# futhermore, codegen does not allow overriding the prefix name for python
	# thus, we patch the swagger spec file to remove the prefix above
	# TODO: remove the line below when https://github.com/swagger-api/swagger-codegen/issues/11993 is addressed
	cat ${SWAGGER_JSON} | sed -e 's/model.//g;s/publicapi.//g;s/pkg\/requester//g;s/types.//g' | tee ./swagger-edited-tmp.json >> /dev/null
	
	jq '.info += {"version":"${VERSION}"}' ./swagger-edited-tmp.json > ./swagger-edited.json

	# generate python client
	rm -rf python/ && ${SWAGGER} generate \
		-i ./swagger-edited.json \
		-l python \
		-o python/ \
		-c python-config.json \
		--remove-operation-id-prefix \
		-Dapis -DapiTests=false -DapiDocs=true \
		-Dmodels -DmodelTests=false -DmodelDocs=true \
		-DsupportingFiles=true

	# patch package version
	cat pyproject.toml.template | sed -e "s/VERSION/${VERSION}/g" > python/pyproject.toml
	# patch README
	cat python/README.md | sed -e "s/Python 2.7 and 3.4+/Python 3.6+/g" | tee python/README.md

	# clean up 
	rm ./swagger-edited*.json || true
	rm python/git_push.sh || true
	rm python/.travis.yml || true
	rm pyproject.toml || true
	cp ../LICENSE python/LICENSE

pypi-build: python/
	cd python && python3 -m pip install --upgrade build && python3 -m build

pypi-upload: pypi-build
	cd python && \
	python3 -m pip install --upgrade twine && \
	python3 -m twine upload \
		--non-interactive \
		--disable-progress-bar \
		--skip-existing \
		-u __token__ -p ${PYPI_TOKEN} \
		dist/*

.PHONY: clean
clean: $(shell find . -type d -depth 1 -not -name bacalhau_sdk)
	$(RM) -r $^
	$(RM) *-config.json
	$(RM) ./swagger-edited.json