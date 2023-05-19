FROM alpine:latest
WORKDIR /app
COPY dashboard-api /app/dashboard-api
RUN chmod a+x /app/dashboard-api
ENTRYPOINT [ "/app/dashboard-api" ]
CMD [ "serve" ]
