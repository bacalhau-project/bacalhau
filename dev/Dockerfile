FROM python:3.11.8-slim

# Install Go
ARG GO_VERSION
ARG GO_ARCHIVE_NAME

RUN apt-get update
RUN apt-get install -y wget git gcc

RUN wget -P /tmp "https://go.dev/dl/${GO_ARCHIVE_NAME}"

RUN tar -C /usr/local -xzf "/tmp/${GO_ARCHIVE_NAME}"
RUN rm "/tmp/${GO_ARCHIVE_NAME}"

ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH
RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" && chmod -R 777 "$GOPATH"

WORKDIR $GOPATH

COPY ./docker-build.sh /
RUN chmod +x /docker-build.sh
RUN /docker-build.sh

COPY ./docker-entrypoint.sh /
RUN chmod +x /docker-entrypoint.sh
ENTRYPOINT [ "/docker-entrypoint.sh" ]
CMD [ "sleep", "infinity" ]