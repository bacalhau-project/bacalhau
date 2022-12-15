<!-- commenting out until we can fix the image logo [![CircleCI](https://dl.circleci.com/status-badge/img/null/filecoin-project/bacalhau/tree/main.svg?style=svg)](https://dl.circleci.com/status-badge/redirect/null/filecoin-project/bacalhau/tree/main)
-->
# The Filecoin Distributed Computation Framework  
<p align="center">
  <img src="docs/images/bacalhau-fish.jpg" alt="Bacalhau Logo" width="400" />
</p>
<p align=center>
  Compute Over Data == CoD
  <br>
  Bacalhau == "Salted CoD Fish" (Portuguese)
</p>
  
<br>

The purpose of Bacalhau is to provide a platform for public, transparent, and optionally verifiable computation. Bacalhau enables users to run arbitrary docker containers and wasm images as tasks against data stored in IPFS. This architecture is also referred to as Compute Over Data (or CoD). The Portuguese word for salted Cod fish is "Bacalhau" which is the origin of the project's name.

Initially, the Bacalhau project will focus on serving data processing and analytics use cases. Over time Bacalhau will expand to other compute workloads, learn more about it future plans in the [roadmap document](ROADMAP.md).

* [Getting Started](https://docs.bacalhau.org/getting-started/installation) ⚡
* [Documentation](https://docs.bacalhau.org/) :closed_book:
* [Slack Community](https://filecoin.io/slack) is open to anyone! Join the `#bacalhau` channel :raising_hand:
* [Code Examples Repository](https://github.com/bacalhau-project/examples) :mag:

Watch a 90 seconds demo of Bacalhau in action:

<p align=center>
  <a href="https://www.youtube.com/watch?v=4YHkmL4Ld74" target="_blank">
    <img src="https://github.com/filecoin-project/bacalhau/raw/a49f4e9c89acce2890aa444fdbb5aa47674ede68/docs/images/thumb-bacalhau-demo-1st-july.jpg" alt="Watch the video" width="580" border="10" />
  </a>
</p>


Learn more about the project from our [Website](https://www.bacalhau.org/), [Twitter](https://twitter.com/BacalhauProject) & [YouTube Channel](https://www.youtube.com/channel/UC45IQagLzNR3wdNCUn4vi0A).

## Latest Updates

* [Weekly Bacalhau Project Reports](https://github.com/filecoin-project/bacalhau/wiki)
* [Bacalhau Overview at DeSci Berlin June 2022](https://www.youtube.com/watch?v=HA8ijt4dzAY)


## Getting Started

Please see the instructions here to get started running a hello example: [Getting Started with Bacalhau](https://docs.bacalhau.org/getting-started/installation).
For a more data intensive demo, check out the [Image Processing tutorial](https://docs.bacalhau.org/examples/data-engineering/image-processing/).

## Getting Help

For usage questions or issues reach out the Bacalhau team either in the [Slack channel](https://filecoinproject.slack.com/archives/C02RLM3JHUY) or open a new issue here on github.

## Developer Guide

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

## Licence

[Apache-2.0](./LICENSE)
