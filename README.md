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

* [Website](https://www.bacalhau.org/)
* [Documentation](https://docs.bacalhau.org/)
* [Slack Channel (#bacalhau)](https://filecoinproject.slack.com/archives/C02RLM3JHUY)
* [Twitter](https://twitter.com/BacalhauProject)
 
## Latest Updates
* Most recent [Bacalhau Project Reports](https://github.com/filecoin-project/bacalhau/wiki)
* [Bacalhau Overview at DeSci Berlin June 2022](https://www.youtube.com/watch?v=HA8ijt4dzAY)


## Getting Started
Please see the instructions here to get started running a hello example and then onboarding your workload to Bacalhau: [Getting Started with the Bacalhau Public Network](https://docs.bacalhau.org/getting-started/installation)


## Developers: Running Bacalhau locally
Developers can spin up bacalhau and run a local demo using the `devstack` command. 
Please see [docs/running_locally.md](docs/running_locally.md) for instructions.


## Notes for Contributors
Bacalhau's CI pipeline performs a variety of linting and formatting checks on new pull requests. To have these checks run locally when you make a new commit, you can use the precommit hook in `./githooks`:

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
