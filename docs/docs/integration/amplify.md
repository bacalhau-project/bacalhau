---
sidebar_label: 'Amplify'
sidebar_position: 1
description: Bacalhau Amplify is a tool for automatically explaining, enriching, and enhancing your data.
---

# Bacalhau Amplify

## Introduction

Bacalhau Amplify is a tool for automatically explaining, enriching and enhancing your data. This document explains how it works and how to use it.

### What is Bacalhau Amplify?

Amplify is both a service and a tool that leverages the power of Bacalhau to automatically run a wide range of data engineering tasks on your data.

It works by running a separate service, the Amplify daemon, that hosts a bundled UI and API, and all of the logic to manage jobs and communicate with executors.

It is designed to be easily extended and used in a variety of deployment scenarios.

### Who is Bacalhau Amplify For?

Amplify is for anyone who wants to automatically run data engineering tasks on their data. You can choose to use the hosted service or deploy it yourself.

## Getting Started

There are four ways you can leverage Amplify, depending on your needs.

### Beginners -- Use the UI and the Hosted Service

If you're just getting started with Bacalhau, or you don't want to manage your own infrastructure, you can use the hosted Amplify UI.

To get started, visit [amplify.bacalhau.org](http://amplify.bacalhau.org) and click on the `Submit a Job` button on the dashboard.

:::tip
This currently only accepts a CID as an input.
:::

### Developers -- As a Service -- Use the Amplify API

If you're a developer and you want to integrate Amplify into your own application, you can use the Amplify API.

You can [browse the OpenAPI specification on the Swagger Editor](https://editor.swagger.io/?url=https://raw.githubusercontent.com/bacalhau-project/amplify/main/api/openapi.yaml).

### On-Prem Developers -- Use the Amplify Container

If you're a developer and you want to run Amplify on your own infrastructure, you can use the Amplify container.

To start Amplify as a service, like the hosted version, run:

```bash
docker run -p 8080:8080 ghcr.io/bacalhau-project/amplify:latest serve
```

To run a single job without starting the service, run:

```bash
docker run ghcr.io/bacalhau-project/amplify:latest run QmS8ioZzB8foNEwFmJZTVJT1se5ycgRuc1Ey5fjHfZi5wb
```

:::tip
You can replace that CID with your own!
:::

### Advanced Users -- Use the Amplify Binary

If you're an advanced user and you want to bundle amplify, then you can use the Amplify binary, or indeed the raw [Go code](https://github.com/bacalhau-project/amplify/).

You can find the most recent binary builds on the [releases page](https://github.com/bacalhau-project/amplify/releases).

1. Download the latest version for your platform.
2. Untar the file.
3. Make the binary executable and place it in a location that is on the PATH.

Now you can run the binary:

```bash
amplify serve # for the service
# or
amplify run QmS8ioZzB8foNEwFmJZTVJT1se5ycgRuc1Ey5fjHfZi5wb # for a single job
```

## Configuration

Amplify can be configured using parameters or environment variables.

Get the most recent configuration options by passing `-h` to the subcommand of your choice:

```bash
docker run ghcr.io/bacalhau-project/amplify:latest serve -h
```

### Database

By default, Amplify runs with an in-memory database. But that implementation is very bare-bones and obviously, you will lose historical information when it restarts. We recommend running Amplify with a PostgreSQL database.

The instructions below describe how to start a PostgreSQL database using Docker. You can also use a managed database service like [Amazon RDS](https://aws.amazon.com/rds/postgresql/) or [Google Cloud SQL](https://cloud.google.com/sql/docs/postgres).

Start a PostgreSQL database and then point Amplify to it using the `AMPLIFY_DB_URI` environment variable:

```bash
docker network create anet
docker run -p 5432:5432 --network=anet --name amplify-postgres -e POSTGRES_DB=amplify -e POSTGRES_PASSWORD=mysecretpassword -d postgres
docker run -p 8080:8080 --network=anet --env AMPLIFY_DB_URI="postgres://postgres:mysecretpassword@amplify-postgres.anet/amplify?sslmode=disable" ghcr.io/bacalhau-project/amplify:latest serve
```

### Triggers

Amplify workflows are executed via a trigger. As of May 2023, Amplify supports the following triggers:

* API -- A trigger that accepts a POST request with a CID as the body.
* IPFS-Search.com -- A trigger that watches an IPFS-Search.com index for new IPFS CIDs. This must be enabled in the configuration.

### Creating New Workflows and Jobs

It's really easy to add new workflows and jobs to Amplify. You can see the [existing workflows and jobs in the `config.yaml` file in the repository](https://github.com/bacalhau-project/amplify/blob/main/config.yaml).

Jobs are simply Docker containers that are executed in Bacalhau. Workflows connect jobs into an execution graph. To find out more please read the [developer documentation](https://github.com/bacalhau-project/amplify/tree/main/docs).

## Developer Documentation

All of the documentation intended for a developer audience is located in the [developer documentation of the repository](https://github.com/bacalhau-project/amplify/tree/main/docs).
