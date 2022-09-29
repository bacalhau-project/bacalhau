#:::
#::: BUILD CONTAINER
#:::

# GO_VERSION is the golang version this image will be built against.
ARG GO_VERSION=1.16

# Dynamically select the golang version.
FROM golang:${GO_VERSION}-buster

# Download and get deps.
COPY /go.mod /go.mod
RUN cd / && go mod download

# Now copy the rest of the source and run the build.
COPY . /src
RUN cd /src/cmd && go build -o service

#:::
#::: RUNTIME CONTAINER
#:::

FROM golang:${GO_VERSION}-buster

RUN mkdir -p /usr/local/bin
COPY --from=0 /src/cmd/service /service
ENV PATH="/:/usr/local/bin:${PATH}"

EXPOSE 5050
WORKDIR "/"
ENTRYPOINT [ "/service" ]

