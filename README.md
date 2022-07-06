[![CircleCI](https://dl.circleci.com/status-badge/img/null/filecoin-project/bacalhau/tree/main.svg?style=svg)](https://dl.circleci.com/status-badge/redirect/null/filecoin-project/bacalhau/tree/main)

<p align="center">
  <img src="docs/images/bacalhau-fish.jpg" alt="Bacalhau Logo" width="400" />
</p>
<p align=center>
  Compute Over Data == CoD
  <br>
  Bacalhau == "Salted CoD Fish" (Portuguese)
</p>
  
<br>

# The Filecoin Distributed Computation Framework  


## Project Background
 * Goals: Open, Collaborative Compute Ecosystem
 * [DESIGN.MD](DESIGN.md)
 * [Bacalhau Overview at PL Eng Res February 2022](https://youtu.be/wmu-lOhSSZo?t=3367)
 
## Latest Updates
  * Most recent [Bacalhau Project Report](https://github.com/filecoin-project/bacalhau/wiki)

## Running Bacalhau locally with devstack
The easiest way to spin up bacalhau and run a local demo is to use the devstack command. Please see [docs/running_locally.md](docs/running_locally.md) for instructions.


## Notes for Contributors
Bacalhau's CI pipeline performs a variety of linting and formatting checks on new pull requests. To have these checks run locally when you make a new commit, you can use the precommit hook in `./githooks`:

```bash
git config core.hooksPath ./githooks
```
