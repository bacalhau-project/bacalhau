MACHINE = $(shell uname -m)
USERNAME ?= bacalhauproject
VERSION ?= 0.5

ifeq ($(MACHINE),x86_64)
    MACHINE := amd64
endif

local:
	@echo - Building local python $(VERSION) - $(MACHINE)
	docker buildx build \
		--platform linux/$(MACHINE) \
		-t $(USERNAME)/exec-python-3.11:$(VERSION) \
		--label org.opencontainers.artifact.created=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ") \
		--load .

build:
	@echo - Building python $(VERSION)
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		-t $(USERNAME)/exec-python-3.11:$(VERSION) \
		--label org.opencontainers.artifact.created=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ") \
		--push .


.PHONY: build local