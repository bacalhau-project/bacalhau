<!-- commenting out until we can fix the image logo [![CircleCI](https://dl.circleci.com/status-badge/img/null/bacalhau-project/bacalhau/tree/main.svg?style=svg)](https://dl.circleci.com/status-badge/redirect/null/bacalhau-project/bacalhau/tree/main)
-->

<!-- commenting out until we can fix the image logo [![CircleCI](https://dl.circleci.com/status-badge/img/null/bacalhau-project/bacalhau/tree/main.svg?style=svg)](https://dl.circleci.com/status-badge/redirect/null/bacalhau-project/bacalhau/tree/main)
-->

<!-- commenting out until we can fix the image logo [![CircleCI](https://dl.circleci.com/status-badge/img/null/bacalhau-project/bacalhau/tree/main.svg?style=svg)](https://dl.circleci.com/status-badge/redirect/null/bacalhau-project/bacalhau/tree/main)
-->
<p align="center">
  <a href="https://github.com/bacalhau-project/bacalhau">
    <img src="https://github.com/bacalhau-project/bacalhau/blob/645acb1734a7bdfc1a7dab5461083b077d34e667/docs/images/logo.png" alt="Bacalhau logo" width="300"/>

  </a>
  </p>

<h1 align="center">The Distributed Computation Framework⚡️ <br>Compute Over Data(CoD)</h1>
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
      <a href="https://filecoinproject.slack.com/" alt="Slack">
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
- [Current state of Bacalhau](current-state-of-bacalhau)
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

![](docs/images/terminal.gif)

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

## Issues, feature requests, and questions

We are excited to hear your feedback!
* For issues and feature requests, please [open a GitHub issue](https://github.com/bacalhau-project/bacalhau/issues).
* For questions, give feedback or answer questions that will help other user product please use [GitHub Discussions](https://github.com/bacalhau-project/bacalhau/discussions).
* To engage with other members in the community, join us in our [slack community](https://filecoin.io/slack/) `#bacalhau` channel :raising_hand:

## Ways to contribute
**All manner of contributions are more than welcome!**

We have highlighted the different ways you can contribute in our [contributing guide](https://docs.bacalhau.org/community/ways-to-contribute). You can be part of community discussions, development, and more.

## Current state of Bacalhau 📈
Building never stops 🛠️.  **Bacalhau is a work in progress!**. Learn more about our future plans in this [roadmap document](https://www.starmaps.app/roadmap/github.com/bacalhau-project/bacalhau/issues/1151)

## License

[Apache-2.0](./LICENSE)
