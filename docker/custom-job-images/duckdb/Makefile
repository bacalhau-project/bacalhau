MACHINE = $(shell uname -m)
USERNAME ?= bacalhauproject
VERSION ?= 0.2

ifeq ($(MACHINE),x86_64)
    MACHINE := amd64
endif

local:
	@echo - Building local duckdb $(VERSION)
	docker buildx build \
		--platform linux/$(MACHINE) \
		--label org.opencontainers.artifact.created=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ") \
		-t $(USERNAME)/exec-duckdb:$(VERSION) --load .

build:
	@echo - Building duckdb $(VERSION)
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		--label org.opencontainers.artifact.created=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ") \
		-t $(USERNAME)/exec-duckdb:$(VERSION) --push .

.PHONY: build local