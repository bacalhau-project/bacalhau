<!-- commenting out until we can fix the image logo [![CircleCI](https://dl.circleci.com/status-badge/img/null/filecoin-project/bacalhau/tree/main.svg?style=svg)](https://dl.circleci.com/status-badge/redirect/null/filecoin-project/bacalhau/tree/main)
-->

<!-- commenting out until we can fix the image logo [![CircleCI](https://dl.circleci.com/status-badge/img/null/filecoin-project/bacalhau/tree/main.svg?style=svg)](https://dl.circleci.com/status-badge/redirect/null/filecoin-project/bacalhau/tree/main)
-->
<p align="center">
  <a href="https://github.com/filecoin-project/bacalhau">
    <img src="https://github.com/filecoin-project/bacalhau/blob/dev/docs/images/bacalhau-fish.jpg" alt="Bacalhau logo" width="300"/>

  </a>
  </p>

<h1 align="center">The Filecoin Distributed Computation Framework⚡️ <br>Compute Over Data(CoD)</h1>
<br>

<p align="center">
    <a href="https://github.com/filecoin-project/bacalhau/blob/dev/LICENSE" alt="Contributors">
        <img src="https://img.shields.io/badge/license-Apache-green" />
        </a>  
    <a href="https://github.com/filecoin-project/bacalhau/releases/" alt="Release">
        <img src="https://img.shields.io/github/v/release/filecoin-project/bacalhau?display_name=tag" />
        </a>
    <a href="https://github.com/filecoin-project/bacalhau/pulse" alt="Activity">
        <img src="https://img.shields.io/github/commit-activity/m/filecoin-project/bacalhau" />
        </a>
    <a href="https://img.shields.io/github/downloads/filecoin-project/bacalhau/total">
        <img src="https://img.shields.io/github/downloads/filecoin-project/bacalhau/total" alt="total download">
        </a>
     <a href="https://github.com/filecoin-project/bacalhau/graphs/contributors">
    <img src="https://img.shields.io/github/contributors/filecoin-project/bacalhau" alt="Bacalhau contributors" >
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

[Bacalhau](https://www.bacalhau.org/) is a platform for public, transparent, and optionally verifiable distributed computation that helps you manage your parallel processing jobs. Bacalhau enables users to run arbitrary docker containers and wasm images as tasks against data stored in IPFS. This architecture is referred to as Compute Over Data (CoD). Bacalhau was coined from the Portuguese word for salted Cod fish. 

## Table of Contents
- [Features](#features)
- [Getting started](#getting-started)
  - [Learn more](#learn-more)
- [Documentation](#documentation)
- [Developers guide](#developers-guide)
  - [Running Bacalhau locally](#running-bacalhau-locally)
  - [Releasing a new version](#releasing-a-new-version)
  - [Notes for Dev contributors](#notes-for-dev-contributors)
- [Ways to contribute ](#ways-to-contribute)
- [Current state of Bacalhau](current-state-of-bacalhau)
- [License](#license)

## Features
- **Process jobs fast**: Jobs in Bacalhau are processed where the data was created (meaning no ingress/egress) and all jobs are parallel by default.
- **Low cost:** Bacalhau uses the compute that produced the data in the first place and reuses the existing hardware you already have. You also save on any ingress/egress fees you may have been charged. The cost of computing is much cheaper than managed services.
- **Secure**: Data is not collected in a central location before processing, meaning all scrubbing and security can be applied at the point of collection.
- **Large-scale data**: With Bacalhau, you can manage your own containers that can scale to batch process petabytes (quadrillion bytes) files.

## Getting started
To hit the ground running, check out our [getting started tutorial](https://docs.bacalhau.org/getting-started/installation) on how to install and run a job with the Bacalhau client.

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

### Releasing a new version

To ship a new version of the CLI & Bacalhau network please follow the instuctions at [docs/create_release.md](docs/create_release.md).

### Notes for Dev contributors

Bacalhau's CI pipeline performs a variety of linting and formatting checks on new pull requests. 
To have these checks run locally when you make a new commit, you can use the precommit hook in `./githooks`:

```bash
git config core.hooksPath ./githooks
```
If you want to run the linter manually:

```bash
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sudo sh -s -- -b /usr/local/go/bin
golangci-lint --version
make lint
```
The config lives in `.golangci.yml`


## Ways to contribute 
**All manner of contributions are more than welcome!**

We have highlighted the different ways you can contribute in our [contributing guide](https://docs.bacalhau.org/community/ways-to-contribute). You can be part of community discussions, development, and more. 

If you have any questions (specific or general), do not hesitate to reach out the Bacalhau team either in the  [slack community](https://filecoin.io/slack/). Join the `#bacalhau` channel :raising_hand: or open a new [issue](https://github.com/filecoin-project/bacalhau/issues) here on github.

## Current state of Bacalhau 📈
Building never stops 🛠️.  **Bacalhau is a work in progress!**. Learn more about our future plans in this [roadmap document](https://www.starmaps.app/roadmap/github.com/filecoin-project/bacalhau/issues/1151)

## License

[Apache-2.0](./LICENSE)
