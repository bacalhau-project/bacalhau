#!/usr/bin/env node
import * as cdk from 'aws-cdk-lib';
import {PipelineStack} from '../lib/pipeline-stack';
import {CanaryStack} from '../lib/canary-stack';
import {getConfig} from "../lib/build-config";

const app = new cdk.App();
const config = getConfig(app);
const canaryStack = new CanaryStack(app, 'BacalhauCanary' + config.envTitle, {
        env: {
            account: config.account,
            region: config.region
        },
        description: 'Bacalhau Canary Stack for ' + config.envTitle + ' environment'
    },
    config);

// we only have a single pipeline for different environments. So we force reading prod configs.
const prodConfig = getConfig(app, 'prod');
new PipelineStack(app, 'BacalhauCanaryPipeline', {
        env: {
            account: prodConfig.account,
            region: prodConfig.region
        },
        lambdaCode: canaryStack.lambdaCode,
        description: 'Bacalhau Canary Pipeline Stack to deploy changes to all canary stacks'
    },
    prodConfig);

app.synth();