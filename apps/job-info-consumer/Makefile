.PHONY: all
all: update build

TARGET_COMMIT ?= $(shell git describe --tags --abbrev=0)

# still using dashboards data dir as we are using the same vm and existing database.
DATA_DIR ?= /data/dashboard

# docker
GOOS ?= linux
GOARCH ?= amd64
CI_COMMIT_SHA ?= $(shell git rev-parse HEAD)
DOCKER_REGISTRY ?= gcr.io
GCP_PROJECT_ID ?= bacalhau-production
IMAGE_TAG ?= "latest"
IMAGE ?= ${DOCKER_REGISTRY}/${GCP_PROJECT_ID}/bacalhau-consumer:${IMAGE_TAG}

.PHONY: update
update:
	cd consumer && go get github.com/bacalhau-project/bacalhau@${TARGET_COMMIT} && go mod tidy

.PHONY: build
build:
	CGO_ENABLED=0 go build -o bin/consumer ./consumer

.PHONY: clean
clean:
	$(RM) -r bin

.PHONY: docker-build
docker-build: build
	docker build --platform "${GOOS}/${GOARCH}" -t ${IMAGE} .
	@echo ${IMAGE}

.PHONY: docker-push
docker-push: docker-build
	docker save ${IMAGE} | bzip2 | gcloud compute ssh dashboard-vm-default-0 -- sudo docker load

.PHONY: docker-compose-push
docker-compose-push:
	# take backup of docker-compose.yml
	gcloud compute ssh dashboard-vm-default-0 -- "sudo cp ${DATA_DIR}/docker-compose.yml ${DATA_DIR}/bk-docker-compose.yml"

	# copy docker-compose.yml to vm
	gcloud compute scp docker-compose.yml dashboard-vm-default-0:/tmp/bacalhau-consumer.yml
	gcloud compute ssh dashboard-vm-default-0 -- "sudo mv /tmp/bacalhau-consumer.yml ${DATA_DIR}/docker-compose2.yml"


.PHONY: restart
restart:
	gcloud compute ssh dashboard-vm-default-0 -- "cd ${DATA_DIR} && sudo docker-compose down && sudo docker-compose up -d"