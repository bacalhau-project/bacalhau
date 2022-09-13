# Bacalhau Monitoring Canary
This is a canary that continuously call several Bacalhau APIs and alarm whenever the correctness or availability of those APIs fall below a threshold.

The canary is fully serverless using AWS Lambda. Infrastructure is defined using AWS CDK, and automatically deployed using AWS CodePipeline.



# Welcome to your CDK TypeScript project

This is a blank project for CDK development with TypeScript.

The `cdk.json` file tells the CDK Toolkit how to execute your app.

## Useful commands

* `npm run build`   compile typescript to js
* `npm run watch`   watch for changes and compile
* `npm run test`    perform the jest unit tests
* `cdk deploy`      deploy this stack to your default AWS account/region
* `cdk diff`        compare deployed stack with current state
* `cdk synth`       emits the synthesized CloudFormation template
