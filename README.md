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
    <a href="https://github.com/badges/shields/graphs/contributors" alt="Contributors">
        <img src="https://img.shields.io/badge/license-Apache-green" /></a>  
    <a href="https://github.com/filecoin-project/bacalhau/releases/" alt="Release">
        <img src="https://img.shields.io/github/v/release/filecoin-project/bacalhau?display_name=tag" /></a>
    <a href="https://github.com/filecoin-project/bacalhau/pulse" alt="Activity">
        <img src="https://img.shields.io/github/commit-activity/m/filecoin-project/bacalhau" /></a>
    <a href="https://img.shields.io/github/downloads/filecoin-project/bacalhau/total">
        <img src="https://img.shields.io/github/downloads/filecoin-project/bacalhau/total" alt="total download"></a>
      <a href="https://filecoinproject.slack.com/" alt="Slack">
        <img src="https://img.shields.io/badge/slack-join_community-red.svg?color=0052FF&labelColor=090422&logo=slack" /></a>
    <a href="https://twitter.com/intent/follow?screen_name=BacalhauProject">
        <img src="https://img.shields.io/twitter/follow/BacalhauProject?style=social&logo=twitter" alt="follow on Twitter"></a>
</p>


[Bacalhau](https://www.bacalhau.org/) is a platform for public, transparent, and optionally verifiable distributed computation that helps you manage your parallel processing jobs. Bacalhau enables users to run arbitrary docker containers and wasm images as tasks against data stored in IPFS. This architecture is referred to as Compute Over Data (CoD). Bacalhau was coined from the Portuguese word for salted Cod fish.

Some benefits of using Bacalhau for your compute-over-data process include:

- 🛠 **Process jobs fast**: jobs are processed where the data was created (meaning no ingress/egress) and all jobs are parallel by default
- 💰 **Low cost:** it uses the compute that produced the data in the first place, reusing the existing hardware you already have. You also save on any ingress/egress fees you may have been charged.
- 🔐 **More secure**: data is not collected in a central location before processing, meaning all scrubbing and security can be applied at the point of collection.

## Learn how Bacalhau works

To learn more about how Bacalhau works, explore the following resources:
- [Bacalhau Docs: Architecture](https://docs.bacalhau.org/about-bacalhau/architecture)
- [Bacalhau Case Studies](https://www.bacalhau.org/casestudies/)

#### Demo and Overview of Project
- [Bacalhau State of the Union](https://www.youtube.com/watch?v=gAHaMsTknZM)
- [Revolutionizing the Big Data Age With Compute over Data](https://www.youtube.com/watch?v=RZopDyTJ1pk)
- [Bacalhau Demo July 1st](https://www.youtube.com/watch?v=4YHkmL4Ld74)

#### Build with Bacalhau
- [Build with Bacalhau: Stable diffusion on a GPU](https://www.youtube.com/watch?v=53uY48e1lis)
- [Build with Bacalhau: Analysing Ethereum Data with Bacalhau](https://www.youtube.com/watch?v=3b0kY13ugBo)
- [Build with Bacalhau: Bacalhau Examples](https://docs.bacalhau.org/examples/)


## Current state of Bacalhau 📈
Building never stops 🛠️.  **Bacalhau is a work in progress!**. In the meanwhile, we plan to deliver:

- A simplified job dashboard that lets you see all your jobs in flight
- A rich SDK for Python, Javascript, and Rust
- Job execution pipelines fully compatible with Airflow
- A job zoo that enables you to pick up existing pipelines from the community
- Automatic wrapping with metadata/lineage and transformation for known file types (columnar, video, audio, etc.)
- An on-premises deployment option for private and custom Hardware
- Internode networking for multi-tier applications
- A standard data store that automatically records data and lineage information of jobs

In the long term, our goal is to deliver a complete system that achieves the following:

- A fully distributed data processing system that can run on any device, anywhere
- A declarative pipeline that can both run the data processing and also record the lineage of the data
- Secure and verifiable results that can be used to confirm the integrity and reproducibility of the results forever

But you tell us👂! Join the [slack community](https://filecoin.io/slack/) `#bacalhau` channel. We'd love to hear about new directions we may need to include. 

Learn more about our future plans in this [roadmap document](https://www.starmaps.app/roadmap/github.com/filecoin-project/bacalhau/issues/1151).

## Try out Bacalhau - Get started

- [Getting Started](https://docs.bacalhau.org/getting-started/installation) ⚡
- [Documentation](https://docs.bacalhau.org/) :closed_book:
- [Code Examples Repository](https://github.com/bacalhau-project/examples)

## Get the latest Bacalhau updates 
- **Twitter** - Follow [Bacalhau](https://twitter.com/BacalhauProject).
- **Product report** - Get the latest [Bacalhau product report](https://bacalhau.substack.com/s/project-report).
- **Community Event and Highlights** - Get the lastest [Bacalhau community event and highlight](https://bacalhau.substack.com/s/cod-summit-highlight).
- **Email** - Subscribe to our [newsletter](https://bacalhau.substack.com/).
- **Slack Community** - For usage questions or issues reach out the Bacalhau team either in the  [slack community](https://filecoin.io/slack/). Join the `#bacalhau` channel :raising_hand: or open a new issue here on github.
- **Bacalhau Videos & Media** - Watch tutorials, community highlights and lots more on our [Youtube channel](https://www.youtube.com/@bacalhauproject).

## Ways to Contribute ❤️
Please see the [contributing guide](https://docs.bacalhau.org/community/ways-to-contribute). Join the [slack community](https://filecoin.io/slack/) `#bacalhau` channel to be part of community discussions about contributions, development, and more.

## Developer Guide 🧭

### Running Bacalhau locally

Developers can spin up bacalhau and run a local demo using the `devstack` command. 
Please see [docs/running_locally.md](docs/running_locally.md) for instructions.
Also, see [docs/debugging_locally.md](docs/debugging_locally.md) for some useful tricks for debugging.

### Release a new version

To ship a new version of the CLI & Bacalhau network please follow the instuctions at [docs/create_release.md](docs/create_release.md).

### Notes for Contributors

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

## Licence

[Apache-2.0](./LICENSE)
