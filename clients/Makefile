PACKAGE_NAME := bacalhau-client
SWAGGER_JSON := ../docs/swagger.json
VERSION := $(shell git describe --tags --abbrev=0 | tr -d v)

#set cmd to be swagger-codegen if that doesnt exist make it the java command
#swagger-codegen is supported by brew install swagger-codegen@2
#the java command is supported by the docker image swaggerapi/swagger-codegen-cli
SWAGGER ?= swagger-codegen
ifeq ($(shell which ${SWAGGER}),)
SWAGGER := java -jar /opt/swagger-codegen-cli/swagger-codegen-cli.jar
ifeq ($(shell ls -1 ${SWAGGER}),)
$(info No swagger-codegen found?)
endif
endif

.PHONY: all
all: $(patsubst %,%/,$(shell cat supported_langs))

$(shell cat supported_langs | xargs mkdir -p)

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
	rm -rf $@ && ${SWAGGER} generate \
		-i ${SWAGGER_JSON} \
		-l $$(basename $@) \
		-o $@ \
		-c $$(basename $@)-config.json

clean: $(shell find . -type d -depth 1)
	$(RM) -r $^
