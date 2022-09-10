#!/usr/bin/env node
import * as cdk from 'aws-cdk-lib';
import { PipelineStack } from '../lib/pipeline-stack';
import { LambdaStack } from '../lib/lambda-stack';

const REPO_NAME = "https://github.com/filecoin-project/bacalhau"

const app = new cdk.App();

const lambdaStack = new LambdaStack(app, 'BacalhauCanaryLambda');

new PipelineStack(app, 'BacalhauCanaryPipeline', {
    lambdaCode: lambdaStack.lambdaCode,
    repositoryName: REPO_NAME,
});

app.synth();