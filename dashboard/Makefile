.PHONY: all
all: update

TARGET_COMMIT ?= $(shell git describe --tags --abbrev=0)

.PHONY: update
update:
	go get github.com/filecoin-project/bacalhau@${TARGET_COMMIT} && go mod tidy
