# Bacalhau API Documentation

Welcome to the official API documentation for Bacalhau. This guide provides a detailed insight into Bacalhau's RESTful HTTP APIs and demonstrates how to make the most out of them.

## Overview

Bacalhau operates on an "API-first" approach, providing an interface for users to interact with the system programmatically.

- **Endpoint Prefix**: All APIs are versioned and prefixed with `/api/v1`.
- **Default Port**: By default, Bacalhau listens on port `1234`.
- **API Nodes**:
  - **Orchestrator**: Handles user requests, schedules, and monitors jobs. Majority of Bacalhau's APIs are dedicated to Orchestrator interactions. These are accessible at `/api/v1/orchestrator`.
  - **Compute Nodes**: Acts as worker nodes and executes the jobs. Both Orchestrator and Compute nodes expose some common APIs under `/api/v1/agent` for querying agent info and health status.

## Features

### Label Filtering

Bacalhau supports label filtering on certain endpoints, such as `/api/v1/orchestrator/jobs` and `/api/v1/orchestrator/nodes`. This mechanism works similarly to constraints, letting you narrow down your search based on certain criteria.

**Example**:
```bash
curl --get "0.0.0.0:1234/api/v1/orchestrator/jobs" --data-urlencode 'labels=env in (prod,dev)'
```


### Pagination

To handle large datasets, Bacalhau supports pagination. Users can define the `limit` in their request and then utilize the `next_token` from the response to fetch subsequent data chunks.

### Ordering

To sort the results of list-based queries, use the `order_by` parameter. By default, the list will be sorted in ascending order. If you want to reverse it, use the `reverse` parameter. Note that the fields available for sorting might vary depending on the specific API endpoint.


### Pretty JSON Output

By default, Bacalhau's APIs provide a minimized JSON response. If you want to view the output in a more readable format, append `pretty` to the query string.

### HTTP Methods

Being RESTful in nature, Bacalhau's API endpoints rely on standard HTTP methods to perform various actions:

- **GET**: Fetch data.
- **PUT**: Update or create data.
- **DELETE**: Remove data.

The behavior of an API depends on its HTTP method. For example, `/api/v1/orchestrator/jobs`:

- **GET**: Lists all jobs.
- **PUT**: Submits a new job.
- **DELETE**: Stops a job.

### HTTP Response Codes

Understanding HTTP response codes is crucial. A `2xx` series indicates a successful operation, `4xx` indicates client-side errors, and `5xx` points to server-side issues. Always refer to the message accompanying the code for more information.
