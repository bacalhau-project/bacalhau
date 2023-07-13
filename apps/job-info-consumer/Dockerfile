FROM alpine:latest
WORKDIR /app
COPY ./bin/consumer /app/consumer
RUN chmod a+x /app/consumer
ENTRYPOINT [ "/app/consumer" ]
CMD [ "start" ]
