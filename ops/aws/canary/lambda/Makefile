.PHONY: all
all: update build

TARGET_COMMIT ?= $(shell git describe --tags --abbrev=0)
TEST_BUILD_TAGS ?= unit,integration

.PHONY: update
update:
	go get github.com/bacalhau-project/bacalhau@${TARGET_COMMIT} && go mod tidy


CMD_TARGETS := $(wildcard ./cmd/*)
BIN_TARGETS := $(patsubst ./cmd/%,./bin/%,${CMD_TARGETS})

.PHONY: build
build: ${BIN_TARGETS}

./bin/%: ./cmd/% ./bin go.mod go.sum
	go build -o $(dir $@) ./$<

./bin:
	mkdir $@


.PHONY: test
test: TEST_BUILD_TAGS=unit
test: tests.xml

.PHONY: integration-test
integration-test: TEST_BUILD_TAGS=integration
integration-test: tests.xml

tests.xml: $(shell find ./pkg ./cmd -type f)
	gotestsum --junitfile $@ -- -timeout 300s --tags=$(TEST_BUILD_TAGS) ./...

.PHONY: clean
clean:
	$(RM) -r ./bin
	git checkout HEAD go.mod go.sum
