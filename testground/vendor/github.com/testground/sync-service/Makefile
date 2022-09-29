.PHONY: install goinstall lint build docker test

install: goinstall docker

goinstall:
	go install .

lint:
	GOGC=75 golangci-lint run --concurrency 32 --deadline 4m ./...

build:
	cd cmd && go build -o ../sync-service

docker:
	docker build -t iptestground/sync-service:latest -f Dockerfile .

test:
	go test -v ./...
