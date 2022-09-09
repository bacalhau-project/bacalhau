#!/usr/bin/env node
import * as cdk from 'aws-cdk-lib';
import { PipelineStack } from '../lib/pipeline-stack';
import { SecretsStack } from '../lib/secrets-stack';

const REPO_NAME = "https://github.com/filecoin-project/bacalhau"

const app = new cdk.App();

const secretsStack = new SecretsStack(app, 'BacalhauSecrets');

new PipelineStack(app, 'BacalhauCanary', {
    repositoryName: REPO_NAME,
});

app.synth();