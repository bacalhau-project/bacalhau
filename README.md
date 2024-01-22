<p align="center">
  <a href="https://github.com/bacalhau-project/bacalhau">
    <img src="./docs/static/img/logo/Bacalhau-horizontal.svg" alt="Bacalhau" width="300"/>
  </a>
</p>

<h1 align="center">The Distributed Computation Framework ⚡<br>Compute Over Data (CoD)</h1>
<br>

<p align="center">
    <a href="https://github.com/bacalhau-project/bacalhau/blob/dev/LICENSE" alt="Contributors">
        <img src="https://img.shields.io/badge/license-Apache-green" />
        </a>
    <a href="https://github.com/bacalhau-project/bacalhau/releases/" alt="Release">
        <img src="https://img.shields.io/github/v/release/bacalhau-project/bacalhau?display_name=tag" />
        </a>
    <a href="https://github.com/bacalhau-project/bacalhau/pulse" alt="Activity">
        <img src="https://img.shields.io/github/commit-activity/m/bacalhau-project/bacalhau" />
        </a>
    <a href="https://img.shields.io/github/downloads/bacalhau-project/bacalhau/total">
        <img src="https://img.shields.io/github/downloads/bacalhau-project/bacalhau/total" alt="total download">
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

[Bacalhau](https://www.bacalhau.org/) is a platform for fast, cost efficient, and secure computation by running jobs where the data is generated and stored. With Bacalhau you can streamline your existing workflows without the need of extensive rewriting by running  arbitrary Docker containers and WebAssembly (wasm) images as tasks.

## Table of Contents
- [Why Bacalhau](#why-bacalhau)
- [Getting started](#getting-started---bacalhau-in-1-minute)
  - [Learn more](#learn-more)
- [Documentation](#documentation)
- [Developers guide](#developers-guide)
  - [Running Bacalhau locally](#running-bacalhau-locally)
  - [Notes for Dev contributors](#notes-for-dev-contributors)
- [Ways to contribute ](#ways-to-contribute)
- [Current state of Bacalhau](#current-state-of-bacalhau-)
- [License](#license)

## Why Bacalhau?
- :zap: **Fast job processing**: Jobs in Bacalhau are processed where the data was created and all jobs are parallel by default.
- :moneybag: **Low cost**: Reduce (or eliminate) ingress/egress costs since jobs are processed closer to the source. Take advantage of as well idle computation capabilities at the edge.
- :lock: **Secure**: Data scrubbing and security can before migration to reduce the chance of leaking private information, and with a far more granular, code-based permission model.
- 🚛 **Large-scale data**: Bacalhau operates on a network of open compute resources made available to serve any data processing workload. With Bacalhau, you can batch process petabytes (quadrillion bytes) of data.

## Getting started - Bacalhau in 1 minute

Go to the folder directory that you want to store your job results

Install the bacalhau client

```bash
curl -sL https://get.bacalhau.org/install.sh | bash
```

Submit a "Hello World" job

```bash
bacalhau docker run ubuntu echo Hello World
```

Download your result

```bash
bacalhau get 63d08ff0..... # make sure to use the right job id from the docker run command
```

![](docs/static/img/terminal.gif)

For a more detailed tutorial, check out our [Getting Started tutorial](https://docs.bacalhau.org/getting-started/installation).

### Learn more
- Understand [Bacalhau Concepts](https://youtu.be/WnTlwXHhbcI)
- Get an overview of the [different usecases](https://www.youtube.com/watch?v=gAHaMsTknZM) that you can use with Bacalhau.
- To see Bacalhau in action, check out the [Bacalhau Examples](https://docs.bacalhau.org/examples/)
- You can check out this featured example video tutorial [Text to image- Stable Diffusion GPU](https://www.youtube.com/playlist?list=PL_1oLZF_wrbTIZdRWqFbtOeI78SdDdsEz). You can watch more tutorials [here](https://www.youtube.com/playlist?list=PL_1oLZF_wrbTIZdRWqFbtOeI78SdDdsEz)

## Documentation
📚 [Read the Bacalhau docs guide here](https://docs.bacalhau.org/)! 📚

The Bacalhau docs is the best starting point as it contains all the information to ensure that everyone who uses Bacalhau is doing so efficiently.

## Developers guide

### Running Bacalhau locally

Developers can spin up bacalhau and run a local demo using the `devstack` command.

Please see [docs/running_locally.md](docs/running_locally.md) for instructions. Also, see [docs/debugging_locally.md](docs/debugging_locally.md) for some useful tricks for debugging.

### Notes for Dev contributors

Bacalhau's CI pipeline performs a variety of linting and formatting checks on new pull requests.
To have these checks run locally when you make a new commit, you can use the precommit hook in `./githooks`:

```bash
make install-pre-commit

# check if pre-commit works
make precommit
```
If you want to run the linter manually:

```bash
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sudo sh -s -- -b /usr/local/go/bin
golangci-lint --version
make lint
```
The config lives in `.golangci.yml`

### OpenAPI

OpenAPI v2 annotations sit by the endpoints in `pkg/publicapi`; these are built using [swag](https://github.com/swaggo/swag), a Go converter for Swagger documentation.
Find more details about the Swag annotations [in their docs](https://github.com/swaggo/swag#declarative-comments-format).
The swagger specification is built automatically by the CI pipeline (see the `build_swagger` workflow) but you can trigger a local build with `make swagger-docs`.

The build parses the OpenAPI annotations as well as the markdown files in `docs/swagger/` (containing  long-form descriptions of the API endpoints), and generates the following swagger specification files:

* `docs/docs.go`
* `docs/swagger.json`
* `docs/swagger.yaml`

Note that the CI pipeline will open and automatically merge a pull request titled `[circleci] Build swagger reference - this is an automatic commit ...` containing *only* the updated spec files.

### Python Libraries

We ship two Python Bacalhau libraries:

* `bacalhau-apiclient` wraps *only* the API endpoint calls and request/response models. It's autogenerated from the OpenAPI specification (see paragraph above). Read more about it in its [readme](./clients/README.md).
* `bacalhau-sdk` is a **high-level** Bacalhau SDK that ships all the client-side logic (e.g. signing requests) needed to query the endpoints. Its [examples folder](./python/examples) contains code snippets to create, list and inspect jobs. Under the hood, it uses the `bacalhau-apiclient` to call the API. Please use this library in your projects. Read more about it in its [readme](./python/README.md).

## Issues, feature requests, and questions

We are excited to hear your feedback!
* For issues and feature requests, please [open a GitHub issue](https://github.com/bacalhau-project/bacalhau/issues).
* For questions, give feedback or answer questions that will help other user product please use [GitHub Discussions](https://github.com/bacalhau-project/bacalhau/discussions).
* To engage with other members in the community, join us in our [slack community](https://join.slack.com/t/bacalhauproject/shared_invite/zt-1sihp4vxf-TjkbXz6JRQpg2AhetPzYYQ) `#bacalhau` channel :raising_hand:

## Ways to contribute
**All manner of contributions are more than welcome!**

We have highlighted the different ways you can contribute in our [contributing guide](https://docs.bacalhau.org/community/ways-to-contribute). You can be part of community discussions, development, and more.

## Open Source
This repository contains the Bacalhau software, covered under the [Apache-2.0](./LICENSE), except where noted (any Bacalhau logos or trademarks are not covered under the Apache License, and should be explicitly noted by a LICENSE file.)

Bacalhau is a product produced from this open source software, exclusively by Expanso, Inc. It is distributed under our commercial terms.

Others are allowed to make their own distribution of the software, but they cannot use any of the Bacalhau trademarks, cloud services, etc.

We explicitly grant permission for you to make a build that includes our trademarks while developing Bacalhau software itself. You may not publish or share the build, and you may not use that build to run Bacalhau software for any other purpose.

We have borrowed the above Open Source clause from the excellent [System Initiative](https://github.com/systeminit/si/blob/main/CONTRIBUTING.md)
