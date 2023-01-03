PACKAGE_NAME := bacalhau-apiclient
SWAGGER_JSON := ../docs/swagger.json
VERSION := $(shell git describe --tags --abbrev=0 | tr -d v)

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
all: $(patsubst %,%/,$(shell cat supported_langs))
	
# $(shell cat supported_langs | xargs mkdir -p)

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

%/: %-config.json ${SWAGGER_JSON}
	# TODO - remove this when https://github.com/swagger-api/swagger-codegen/issues/11993 is addressed
	cat ${SWAGGER_JSON} | sed -e 's/model.//g;s/publicapi.//g;s/types.//g' | tee ./swagger-edited.json >> /dev/null &&
	rm -rf $@ && ${SWAGGER} generate \
		-i ./swagger-edited.json \
		-l $$(basename $@) \
		-o $@ \
		-c $$(basename $@)-config.json \
		--remove-operation-id-prefix=true \
		-Dapis -DapiTests=false -DapiDocs=true \
		-Dmodels -DmodelTests=false -DmodelDocs=true \
		-DsupportingFiles=true
	rm ./swagger-edited.json &&
	cp pyproject.toml $@/pyproject.toml &&
	cp ../LICENSE $@/LICENSE

pypi: python/
	cd python && python3 -m pip install --upgrade build && python3 -m build

pypi-upload: pypi
	cd python && python3 -m pip install --upgrade twine && python3 -m twine upload dist/*

.PHONY: clean
clean: $(shell find . -type d -depth 1 -not -name bacalhau_sdk)
	$(RM) -r $^ &&
	$(RM) *-config.json &&
	$(RM) ./swagger-edited.json