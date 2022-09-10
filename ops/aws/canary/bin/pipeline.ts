#!/usr/bin/env node
import * as cdk from 'aws-cdk-lib';
import { PipelineStack } from '../lib/pipeline-stack';
import { CanaryStack } from '../lib/canary-stack';

const REPO_NAME = "https://github.com/filecoin-project/bacalhau"

const app = new cdk.App();

const canaryStack = new CanaryStack(app, 'BacalhauCanary');

new PipelineStack(app, 'BacalhauCanaryPipeline', {
    lambdaCode: canaryStack.lambdaCode,
    repositoryName: REPO_NAME,
});

app.synth();