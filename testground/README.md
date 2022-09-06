# Testground
***Note: Onboarding with testground is still work in progress***

These are test scenarios based on [testground](https://github.com/testground/testground) framework. It allows running test scenarios against a network of Bacalhau instances either locally or in a remote cluster (i.e. kubernetes). Other than testing the functionality, it is very useful to benchmark the performance of Bacalhau, and to test failure scenarios through network traffic shaping.

### Installation
You can install testground by following the instructions in the [testground repo](https://github.com/testground/testground#getting-started).

### Importing test plans
To import a test plan, you can use the `testground import` command. For example:
```shell
testground plan import --from /Users/walid/workspace/bacalhau/testground --name bacalhau
```
This will import the test plan in this repo and will give it the name `bacalhau`.

### Running tests locally
*Note: This is waiting for [this PR](https://github.com/testground/testground/pull/1443) to be merged to work.*

testground will by default run tests against a Bacalhau module that is merged into the master branch. If you want to run tests against a local version of Bacalhau, you can use the `--dep` flag to replace the module with a local path. For example:
```shell
testground run single --plan bacalhau \
  --testcase catFileToVolume \
  --builder exec:go \
  --runner local:exec  \
  --instances 3 \
  --wait \
  --dep github.com/filecoin-project/bacalhau=/Users/walid/workspace/bacalhau
```