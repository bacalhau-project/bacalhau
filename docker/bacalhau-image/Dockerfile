# syntax=docker/dockerfile:1.4
# Pinned to 1.20
FROM cgr.dev/chainguard/go@sha256:8ed3fdc8f6375a3fd84b4b8b696a2366c3a639931aab492d6f92ca917e726ad6 as build

# Release tag. Used to build the binary and tag the version.
ARG TAG

WORKDIR /work

# only copy what we need to build the binary instead of
# the whole repo and then filter using .dockerignore
COPY pkg ./pkg
COPY cmd ./cmd
COPY testdata ./testdata
COPY docs ./docs
COPY go.mod .
COPY main.go .
COPY Makefile .
RUN make modtidy
RUN make build-bacalhau
RUN find ./bin -name 'bacalhau' -exec mv -t ./bin {} +

FROM cgr.dev/chainguard/static:latest
COPY --from=build /work/bin/bacalhau /usr/local/bin/bacalhau
ENV PATH="/usr/local/bin"
ENTRYPOINT ["bacalhau"]
LABEL org.opencontainers.image.source https://github.com/bacalhau-project/bacalhau
LABEL org.opencontainers.image.title "Bacalhau"
LABEL org.opencontainers.image.description "The Bacalhau network provices decentralised compute for compute over data. See https://bacalhau.org for more info."
LABEL org.opencontainers.image.licenses Apache-2.0
LABEL org.opencontainers.image.url https://bacalhau.org
