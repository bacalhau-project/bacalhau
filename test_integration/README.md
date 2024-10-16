# Integration Test Suite for Bacalhau

This test suite is designed to perform integration testing for Bacalhau using real docker containers, simulating a production-like environment.

## Key Features

- Uses TestContainers Go library for managing Docker environments
- Implements Testify Go testing suite for structured and extensible tests
- Compiles project binary and builds custom Docker images for testing
- Provides a base suite with common functionality, easily extendable for specific test cases
- Simulates real production usage by executing commands in a jumpbox container

## Architecture

1. **Base Suite**: Compiles the project binary and builds several Docker images:
    - Compute node
    - Requester node
    - Jumpbox node

2. **TestContainers**: Used to set up and manage a Docker Compose stack for each test suite.

3. **Test Structure**:
    - Each test suite inherits from the base suite
    - Suites have their own Docker Compose stack, shared by all tests within the suite
    - Tests within a suite run in series
    - Different suites can run in parallel if needed

4. **Test Execution**:
    - Tests use TestContainers' exec-in-container functionality
    - Commands are run in the jumpbox container, simulating real-world usage
    - No mocking is used, providing high-fidelity test results

## Run Test Suite:

To run test suite, you will need to have docker daemon on your local machine, as well as docker compose,

Then:

```shell
cd test_integration
go test -v -count=1 ./...
```
