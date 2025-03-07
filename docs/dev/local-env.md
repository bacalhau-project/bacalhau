# Developer guide

This guide helps you get started developing the Bacalhau project.

## Dependencies

* [Git](https://git-scm.com/)
* [Python](https://www.python.org/) (see [pyproject.toml](../../pyproject.toml) for supported versions)
* [Go](https://go.dev/)
* [Earthly](https://github.com/earthly/earthly)
* [Docker](https://docs.docker.com/) (optional for building, required for running integration tests)

## Download Bacalhau

Clone the Bacalhau repository using your preferred tool. Refer to GitHub's [documentation](https://docs.github.com/en/github/creating-cloning-and-archiving-repositories/cloning-a-repository) for different options.

To clone Bacalhau repository using Git CLI:
```bash
git clone https://github.com/bacalhau-project/bacalhau.git
cd bacalhau
```

## Create and activate Python virtual environment (optional)

We recommend using Python [virtual environments](https://docs.python.org/3/library/venv.html) to avoid version conflicts.

Create a virtual environment in the `.venv` folder:
```bash
python -m venv .venv
```

And activate it:
```bash
source .venv/bin/activate
```

## Configure pre-commit hooks

Bacalhau uses pre-commit hooks for linting and formatting code. These checks will also be executed by Bacalhau's CI pipeline on all new pull requests. Check [.golangci.yml](../../.golangci.yml) for the linter rules.

To install the pre-commit hooks:

```bash
make install-pre-commit
```

To check if pre-commit passes:
```bash
make precommit
```

## Build Bacalhau

You can check individual build targets in the [Makefile](../../Makefile) or build all of them together.
Refer to [Key Concepts](https://docs.bacalhau.org/overview/architecture#core-components) to learn more about different Bacalhau components.

To build all Bacalhau components:

```bash
make build
```

## Run locally

You can spin up a local Bacalhau stack and interact with it. For a detailed guide, check Bacalhau [documentation](https://docs.bacalhau.org/getting-started/network-setup#option-3-devstack).

To run a local stack:
```bash
make devstack
```

You can run the local stack with a number of different configurations. For details, check the `devstack-*` targets in the [Makefile](../../Makefile).

## Run tests

Bacalhau tests can be generally divided into these categories:
* Unit tests
* Tests against a local stack
* Integration tests using Docker

### Unit tests

To run all unit tests:
```bash
make unit-test
```

### Tests against local stack

These tests will run a Bacalhau stack in local processes and execute tests against it. No Docker is required.

To run tests against a local stack:
```bash
make integration-test
```

### Integration tests

These tests mimic a real-life distributed installation of Bacalhau by running node processes in Docker containers (using [Testcontainers](https://docs.docker.com/testcontainers/)). Refer to the Integration Test [guide](../../test_integration/README.md) for detailed steps.
