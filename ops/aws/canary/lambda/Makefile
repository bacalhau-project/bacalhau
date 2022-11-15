.PHONY: all
all: update build

TARGET_COMMIT ?= $(shell git describe --tags --abbrev=0)

.PHONY: update
update:
	go get github.com/filecoin-project/bacalhau@${TARGET_COMMIT}


CMD_TARGETS := $(wildcard ./cmd/*)
BIN_TARGETS := $(patsubst ./cmd/%,./bin/%,${CMD_TARGETS})

.PHONY: build
build: ${BIN_TARGETS}

./bin/%: ./cmd/% ./bin go.mod go.sum
	go build -o $(dir $@) ./$<

./bin:
	mkdir $@

.PHONY: clean
clean:
	$(RM) -r ./bin
	git checkout HEAD go.mod go.sum
