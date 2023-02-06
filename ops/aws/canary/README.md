# Bacalhau Monitoring Canary
This is a canary that continuously call several Bacalhau APIs and alarm whenever the correctness or availability of those APIs fall below a threshold.

The canary is serverless using AWS Lambda. Infrastructure is defined using AWS CDK, and automatically deployed using AWS CodePipeline.

## Quick LInks
- [Public Dashboard](https://cloudwatch.amazonaws.com/dashboard.html?dashboard=BacalhauCanaryProd&context=eyJSIjoidXMtZWFzdC0xIiwiRCI6ImN3LWRiLTI4NDMwNTcxNzgzNSIsIlUiOiJ1cy1lYXN0LTFfUTlPMEVrM3llIiwiQyI6IjExc3NlYW1tZmVmaGdtYTFzMDk1c29jaDltIiwiSSI6InVzLWVhc3QtMTpmNGE5MGFiMi0yZWYwLTRlYTEtOWZkNS1jMmQ3MDkxYTA5OTQiLCJNIjoiUHVibGljIn0=)
- [AWS Account Sign-in](https://284305717835.signin.aws.amazon.com/console/?region=eu-west-1)
- [Canary Prod Logs](https://eu-west-1.console.aws.amazon.com/cloudwatch/home?region=eu-west-1#logsV2:log-groups)
- [Canary Lambda Functions](https://eu-west-1.console.aws.amazon.com/lambda/home?region=eu-west-1#/functions?fo=and&o0=%3A&v0=BacalhauCanary)
- [Deployment Pipeline](https://console.aws.amazon.com/codesuite/codepipeline/pipelines/BacalhauCanaryPipeline-PipelineC660917D-I0DZJY6IFHTO/view?region=eu-west-1)

## Canary Scenarios
The canary is composed of several scenarios, each is executed periodically on its own lambda function. The scenarios are defined in the `lambda/pkg/scenarios` directory, and include:
- `list`: Call Bacalhau's list API and verify the response.
- `submit`: Submits a job to Bacalhau and verify it was successfully completed
- `submitAndDescribe`: Submits a job to Bacalhau, waits for it to complete, and then calls the describe related APIs.
- `submitAndGet`: Submits a job to Bacalhau, waits for it to complete, and then download the output and verify its correctness.
- `submitDockerIPFSJobAndGet`: Submits a job to Bacalhau with an IPFS input, waits for it to complete, and then download the output and verify its correctness.
- `submitWithConcurrency`: Submits a job to Bacalhau with a concurrency of 3, and waits for it to complete.
- `submitWithConcurrencyOwnedNodes`: Submits a job to Bacalhau owned nodes with a concurrency of 3, and waits for it to complete.

### Local Testing
You can run the scenarios locally before deploying to lambda by using the following command:
```bash
# Assuming you are in the ops/aws/canary directory
go run ./lambda/cmd/scenario_local_runner --action list # or any other scenario

# If you get a `no packages loaded from` error just cd into the /ops/aws/canary/lambda/cmd/scenario_local_runner directory
go run . --action list
```

## Releasing a New Version
Follow these steps when a new version of Bacalhau is released and deployed to prod so that the canary client is also updated to a compatible version and deployed:
1. Update the `go.mod` in the [ops/aws/canary/lambda directory](ops/aws/canary/lambda/go.mod) to point to the new version of Bacalhau.
2. Run `go mod tidy` to update the `go.sum` file by running `(cd ops/aws/canary/lambda && go mod tidy)`
3. Update any breaking changes in Bacalhau client API.
4. Verify the canary is compiling locally by running `(cd ops/aws/canary/lambda &&  go build -o /dev/null ./cmd/scenario_lambda_runner)`
5. Push the changes to main, and the canary pipeline will automatically deploy the new version.

This is a [sample commit](https://github.com/filecoin-project/bacalhau/commit/958630dbe4ad9ba35b0715be2f82c66c60797ba4) updating the canary to Bacalhau v0.2.6

## Infrastructure Stacks
There are two types of stacks in this project:
- Canary stack(s): one stack per environment (e.g. prod, dev), containing the Lambda function and the CloudWatch alarm.
- Pipeline stack: contains the CodePipeline and CodeBuild resources.

### Deploying Canary Stacks Changes
Changes to the canary stacks are automatically deployed as soon a new commit is pushed to the main branch. You *should not* deploy this stack manually.

**Note:** Currently only the prod stack is deployed.

### Deploying Pipeline Stack Changes
Changes to the pipeline such as adding a new stage or modifying the build scripts needs to be deployed manually. To do so, run the following command:
```bash
# Assuming you have the AWS CLI installed and configured with a profile named "bacalhau"
# Assuming you are in the ops/aws/canary directory
cdk --profile bacalhau deploy BacalhauCanaryPipeline -c config=prod
```
Note that we only have a single pipeline stack deployed using prod environment configuration, but it will deploy all canary stacks.

### Manual Resources
These are the resources that had to be created/updated manually outside of CDK:
1. GitHub Connection
2. CloudWatch public dashboard link
3. Update secret manager with Slack webhook URL


## Useful CDK commands
Keep in mind that you might need to pass your AWS profile and the stack name in some of these commands:
* `npm run build`   compile typescript to js
* `npm run postinstall` deletes cdk golang templates that can result in breaking go commands due to invalid file naming pattern
* `npm run watch`   watch for changes and compile
* `npm run test`    perform the jest unit tests
* `cdk deploy`      deploy this stack to your default AWS account/region
* `cdk diff`        compare deployed stack with current state
* `cdk synth`       emits the synthesized CloudFormation template
