<p align="center">
  <a href="https://github.com/bacalhau-project/bacalhau">
    <img src="./docs/logo/Bacalhau-horizontal.svg" alt="Bacalhau" width="300"/>
  </a>
</p>

<h1 align="center">Globally Distributed Compute Orchestrator ⚡<br>Compute Over Data (CoD)</h1>
<br>

<p align="center">
    <a href="https://github.com/bacalhau-project/bacalhau/blob/main/LICENSE" alt="License">
        <img src="https://img.shields.io/badge/license-Apache-green" />
    </a>
    <a href="https://github.com/bacalhau-project/bacalhau/releases/" alt="Release">
        <img src="https://img.shields.io/github/v/release/bacalhau-project/bacalhau?display_name=tag" />
    </a>
    <a href="https://github.com/bacalhau-project/bacalhau/pulse" alt="Activity">
        <img src="https://img.shields.io/github/commit-activity/m/bacalhau-project/bacalhau" />
    </a>
    <a href="https://github.com/bacalhau-project/bacalhau/graphs/contributors">
        <img src="https://img.shields.io/github/contributors/bacalhau-project/bacalhau" alt="Bacalhau contributors" >
    </a>
    <a href="https://www.bacalhau.org/">
        <img alt="Bacalhau website" src="https://img.shields.io/badge/website-bacalhau.org-red">
    </a>
    <a href="https://bit.ly/bacalhau-project-slack" alt="Slack">
        <img src="https://img.shields.io/badge/slack-join_community-red.svg?color=0052FF&labelColor=090422&logo=slack" />
    </a>
    <a href="https://twitter.com/intent/follow?screen_name=BacalhauProject">
        <img src="https://img.shields.io/twitter/follow/BacalhauProject?style=social&logo=twitter" alt="follow on Twitter">
    </a>
</p>

## What is Bacalhau?

[Bacalhau](https://www.bacalhau.org/) is an open-source distributed compute orchestration framework designed to bring compute to the data. Instead of moving large datasets around networks, Bacalhau makes it easy to execute jobs close to the data's location, drastically reducing latency and resource overhead.

## Why Bacalhau?

- ⚡ **Fast job processing**: Jobs in Bacalhau are processed where the data was created and all jobs are parallel by default
- 💰 **Low cost**: Reduce (or eliminate) ingress/egress costs since jobs are processed closer to the source
- 🔒 **Secure**: Data scrubbing and security can happen before migration, with a granular, code-based permission model
- 🚛 **Large-scale data**: Process petabytes of data efficiently without massive data transfers
- 🏢 **Data sovereignty**: Process sensitive data within security boundaries without requiring it to leave your premises
- 🤝 **Cross-organizational computation**: Allow specific vetted computations on protected datasets without exposing raw data

## Key Features

1. **Single Binary Simplicity**: Bacalhau is a single self-contained binary that functions as a client, orchestrator, and compute node—making it incredibly easy to set up and scale
   
2. **Modular Architecture**: Support for multiple execution engines (Docker, WebAssembly) and storage providers through clean interfaces

3. **Orchestrator-Compute Model**: A dedicated orchestrator coordinates job scheduling, while compute nodes run tasks

4. **Flexible Storage Integrations**: Integrates with S3, HTTP/HTTPS, IPFS, and local storage systems

5. **Multiple Job Types**: Support for batch, ops, daemon, and service job types for different workflow requirements

6. **Declarative & Imperative Submissions**: Define jobs in YAML (declarative) or pass arguments via CLI (imperative)

7. **Publisher Support**: Output results to local volumes, S3, or other storage backends

## Getting Started

### Quick Installation

```bash 
# Install Bacalhau CLI (Linux/macOS)
curl -sL https://get.bacalhau.org/install.sh | bash

# Verify installation
bacalhau version
```

For the complete quick start guide, including running your first job, see our [Quick Start Documentation](https://docs.bacalhau.org/getting-started/quick-start).

## Use Cases

Bacalhau's distributed compute framework enables a wide range of applications:

- **Log Processing**: Process logs efficiently at scale by running distributed jobs directly at the source
- **Distributed Data Warehousing**: Query and analyze data across multiple regions without moving large datasets
- **Fleet Management**: Efficiently manage distributed nodes across multiple environments
- **Distributed Machine Learning**: Train and deploy ML models across a distributed compute fleet
- **Edge Computing**: Run compute tasks closer to the data source for applications requiring low latency

## Documentation

📚 [Read the Bacalhau docs guide here](https://docs.bacalhau.org/)! 📚

The Bacalhau documentation contains all the information you need to get started:

- [Installation Tutorial](https://docs.bacalhau.org/getting-started/installation)
- [Basic Usage](https://docs.bacalhau.org/getting-started/cli)
- [Common Workflows](https://docs.bacalhau.org/common-workflows)

## Community & Contributing

Bacalhau has a very friendly community, and we are always happy to help:

- [Join the Slack Community](https://join.slack.com/t/bacalhauproject/shared_invite/zt-1sihp4vxf-TjkbXz6JRQpg2AhetPzYYQ) and go to the `#general` channel - it is the easiest way to engage with other members in the community and get help

If you are interested in contributing to the Bacalhau project:

- Set up your [local environment](docs/dev/local-env.md)
- Check out our [Contributing Guide](https://docs.bacalhau.org/community/community/ways-to-contribute)
- For issues and feature requests, please [open a GitHub issue](https://github.com/bacalhau-project/bacalhau/issues)

We are excited to hear your feedback!

## Open Source

This repository contains the Bacalhau software, covered under the [Apache-2.0](./LICENSE) license, except where noted (any Bacalhau logos or trademarks are not covered under the Apache License, and should be explicitly noted by a LICENSE file.)

Bacalhau is a product produced from this open source software, exclusively by Expanso, Inc. It is distributed under our commercial terms.

Others are allowed to make their own distribution of the software, but they cannot use any of the Bacalhau trademarks, cloud services, etc.

We explicitly grant permission for you to make a build that includes our trademarks while developing Bacalhau software itself. You may not publish or share the build, and you may not use that build to run Bacalhau software for any other purpose.

We have borrowed the above Open Source clause from the excellent [System Initiative](https://github.com/systeminit/si/blob/main/CONTRIBUTING.md)
