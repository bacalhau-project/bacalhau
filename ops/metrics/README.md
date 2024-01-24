# Usage
**Start containers:**
```shell
docker-compose up
```
**Export collection endpoint for bacalhau**
```shell
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
 ```
**Start Bacalhau**
```shell
bacalhau serve --node-type=compute,requester
```
**Open Browser**
- Grafana: http://localhost:3000
  - Username: `admin`
  - Password: `admin`
- Jaeger: http://localhost:16686

**Clean up**
- Remove volumes associated with containers to reset state.