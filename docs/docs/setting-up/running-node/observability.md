---
sidebar_label: 'Observability'
sidebar_position: 200
---
# Observability

Bacalhau supports the three main 'pillars' of observability - logging, metrics, and tracing.  
Bacalhau uses the [OpenTelemetry Go SDK](https://github.com/open-telemetry/opentelemetry-go) for **metrics** and **tracing**, which can be configured using the [standard environment variables](https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/protocol/exporter.md). Exporting metrics and traces can be as simple as setting the `OTEL_EXPORTER_OTLP_PROTOCOL` and `OTEL_EXPORTER_OTLP_ENDPOINT` environment variables. 

:::info
Here are [all the environment variables](https://github.com/bacalhau-project/bacalhau/blob/main/pkg/telemetry/constants.go) users may set to configure the exportation of telemetry data from bacalhau 
:::

Custom code is used for **logging**, as the [OpenTelemetry Go SDK currently doesn't support logging](https://github.com/open-telemetry/opentelemetry-go#project-status).

## Logging
Logging in Bacalhau outputs in human-friendly format to `stderr` at `info` level by default, but this can be changed by two environment variables:
1. `LOG_LEVEL` - Can be one of **`info`**, **`trace`**, **`debug`**, **`error`**, **`warn`** or **`fatal`** (default `info`) to output more or fewer logging messages as required  

2. `LOG_TYPE` - Can be one of the following values:  
**`default`** - output logs to stderr in a human-friendly format  
**`json`** - log messages outputted to stdout in JSON format  
**`combined`** - log JSON formatted messages to stdout and human-friendly format to stderr  
**`event`** - will not print any of the usual log messages  
**`station`** - will print messages with just the log level and message, and no timestamp or other punctuation. Additionally, it will also cause the API endpoint to be printed on boot

Log statements should include the relevant trace, span and job ID so it can be tracked back to the work being performed.

## Metrics
Bacalhau produces a number of different metrics. You can find all the files in bacalhau containing metrics via the following command:

```bash
%%bash
find . -type f -name "metrics.go"
```
Here is a list of metrics produced by Bacalhau and their respective files:  

`./pkg/compute/metrics.go` : metrics regarding the compute node  
`./pkg/storage/tracing/metrics.go` : metrics regarding the storage systems  
`./pkg/requester/metrics.go` : metrics regarding the requester node  
`./pkg/orchestrator/metrics.go` : metrics regarding the orchestrator  
`./pkg/executor/docker/metrics.go` : metrics regarding docker executions  
`./pkg/executor/wasm/metrics.go` : metrics regarding WASM executions  
`./pkg/publisher/tracing/metrics.go` : metrics regarding the publisher systems  


## Tracing
Traces are generated for all significant processes during job execution. You can find relevant traces covering working on a job by searching for the `jobid` attribute.  

The list of possible spans is extensive, and it's not limited to traces emitted directly by Bacalhau. Libraries used by Bacalhau also emit traces. If you want a comprehensive list of spans emitted by Bacalhau, run:

```bash
%%bash
grep -E 'NewSpan|NewRootSpan' -r --include='*.go' .
```

## Viewing
The metrics and traces can easily be forwarded to a variety of different services as we use OpenTelemetry, such as Honeycomb or Datadog.

To view the data locally you can use the following guide that describes the process of setting up a comprehensive observability stack for Bacalhau, leveraging popular tools such as Prometheus, Grafana, OpenTelemetry Collector, and Jaeger.

### Step 1: Prepare Configuration Files

`docker-compose.yaml`: Defines the services and their configurations, ensuring compatibility with the required dependencies

```yaml title="docker-compose.yaml"
version: '3.5'

services:
  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./prometheus/:/etc/prometheus/
      - prometheus-storage:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/usr/share/prometheus/console_libraries'
      - '--web.console.templates=/usr/share/prometheus/consoles'
    ports:
      - 9090:9090
    restart: always

  grafana:
    image: grafana/grafana
    depends_on:
      - prometheus
    volumes:
      - ./grafana/provisioning/datasources:/etc/grafana/provisioning/datasources  # Datasource provisioning
      - ./grafana/provisioning/dashboards:/etc/grafana/provisioning/dashboards  # Dashboard provisioning

    ports:
      - 3000:3000
    restart: always

  opentelemetry-collector:
    image: otel/opentelemetry-collector:latest
    command: [ "--config=/etc/otel-collector-config.yaml" ] # Command to use the custom config
    volumes:
      - ./otel-collector-config.yaml:/etc/otel-collector-config.yaml
    ports:
      - 127.0.0.1:4318:4318 # HTTP
      - 55681:55681 # OpenTelemetry protocol
    depends_on:
      - prometheus

  jaeger:
    container_name: jaeger
    image: jaegertracing/all-in-one:latest
    ports:
      - "6831:6831/udp"
      - "5778:5778"
      - "4316:4316"
      - "16686:16686"
      - "14268:14268"

volumes:
  prometheus-storage: {}
```

`otel-collector-config.yaml`: Configures the OpenTelemetry Collector, a crucial component for collecting and exporting telemetry data

```yaml title="otel-collector-config.yaml"
# receive telemetry data from bacalhau otel sdk.
receivers:
  otlp:
    protocols:
      http:
        endpoint: "0.0.0.0:4318"

# batch process data and label it with 'otel' as the service colector
processors:
  batch:
  memory_limiter:
    check_interval: 5s
    limit_mib: 4000
    spike_limit_mib: 500
  resource:
    attributes:
      - key: service.collector
        value: otel
        action: insert
  attributes/metrics:
    actions:
      - pattern: net\.sock.+
        action: delete


exporters:
  # metrics are exported to prometheus
  prometheus:
    endpoint: "0.0.0.0:9095"
    namespace: "bacalhau"
  # uncomment for debugging, will print all metrics to stdout
  #logging:
    #loglevel: debug
  # traces go to jaeger instance
  otlp/jaeger:
    endpoint: "jaeger:4317"
    tls:
      insecure: true
      insecure_skip_verify: true

service:
  pipelines:
    metrics:
      receivers: [otlp]
      processors: [memory_limiter, resource, attributes/metrics, batch]
      exporters: [prometheus]
      #exporters: [prometheus, logging]
    traces:
      receivers: [otlp]
      processors: [memory_limiter, resource, attributes/metrics, batch]
      exporters: [otlp/jaeger]
      #exporters: [logging, otlp/jaeger]
```
[`datasources.yml`](https://github.com/bacalhau-project/bacalhau/blob/main/ops/metrics/grafana/provisioning/datasources/datasources.yml) and [`dashboard.json`](https://github.com/bacalhau-project/bacalhau/blob/main/ops/metrics/grafana/provisioning/dashboards/dashboard.json): Configure data sources and Grafana dashboards, respectively, for a seamless visualization experience.

[`dashboards.yml`](https://github.com/bacalhau-project/bacalhau/blob/main/ops/metrics/grafana/provisioning/dashboards/dashboards.yml): Specifies the path to the folder containing Grafana dashboard files.

[`prometheus.yml`](https://github.com/bacalhau-project/bacalhau/blob/main/ops/metrics/prometheus/prometheus.yml): Configures Prometheus to scrape metrics from the OpenTelemetry Collector.


### Step 2: Launching the stack and configuring Bacalhau

Execute the following commands in the terminal:

```shell
# Start containers:
docker-compose up

# Export collection endpoint for Bacalhau
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318

# Start Bacalhau
bacalhau serve --node-type=compute,requester
```

### Step 3: Access Interfaces

Open a browser and go to the following links:

**Grafana**: http://localhost:3000 (Use `admin` as both the Username and Password)  

**Jaeger**: http://localhost:16686

### Step 4: Cleanup After Use

To remove volumes associated with containers to reset state, run:

```bash
%%bash
docker-compose down -v
```

### Step 5: Saving Changes to a Grafana Dashboard

To save changes made to Grafana dashboards, follow these steps:

1. Export dashboard data from Grafana as json  
2. Save it to file `./grafana/provisioning/dashboards/dashboard.json`


