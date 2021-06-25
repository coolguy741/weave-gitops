# WEGO User Acceptance Tests

This suite contains the user acceptance tests for the Weave GitOps. To run these tests you can either use gingko runner or standard go test command .

By default test harness assumes that WEGO binary is available on `$PATH` but this can be overriden by exporting the following variable


```
export WEGO_BIN_PATH=<path/to/wego-binary>
```

Additional env vars used to run tests locally are:
```
export GITHUB_ORG=<github-org>
export GITHUB_TOKEN=<api-token>
export GITHUB_KEY=<ssh-key>
```
Please make sure that `GITHUB_TOKEN` has repo create and delete permissions on `GITHUB_ORG`

To use an existing cluster with active kubectl context, export the following variable before running the tests.

```
export CLUSTER_PROVIDER=kubectl
```
# Smoke Tests

To run the **smoke tests** from the suite, run the following the command from the repo root directory.

```
ginkgo --focus=SmokeTest --randomizeSuites  -v ./test/acceptance/test/...
```
# Acceptance Tests
To run the full **acceptance suite**, run the command


```
ginkgo --randomizeSuites -v ./test/acceptance/test/...
```

# How to add new test

Smoke test can be added to `smoke_tests.go` or create a new go file with smoke as build tag.

For non smoke tests, feel free to create appropriately named go file.

This suite follows the **BDD** gherkin style specs, when adding a new test, make every effort to adhere to `Given-When-Then` semantics. 
