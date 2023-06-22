module github.com/bacalhau-project/bacalhau/apps/job-info-consumer/consumer

go 1.20

replace github.com/bacalhau-project/bacalhau => ../../..

require github.com/bacalhau-project/bacalhau v1.0.0

require (
	github.com/golang-migrate/migrate/v4 v4.16.2 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/lib/pq v1.10.9 // indirect
	go.opentelemetry.io/otel v1.14.0 // indirect
	go.uber.org/atomic v1.10.0 // indirect
)
