#!/usr/bin/env node
import * as cdk from 'aws-cdk-lib';
import {PipelineStack} from '../lib/pipeline-stack';
import {CanaryStack} from '../lib/canary-stack';
import {getCanaryConfig, getPipelineConfig} from "../lib/config";

const app = new cdk.App();
const config = getCanaryConfig(app);
const canaryStack = new CanaryStack(app, 'BacalhauCanary' + config.envTitle, {
        env: {
            account: config.account,
            region: config.region
        },
        description: 'Bacalhau Canary Stack for ' + config.envTitle + ' environment'
    },
    config);

// we only have a single pipeline for different environments. So we force reading prod configs.
const pipelineConfig = getPipelineConfig(app, 'prod');
new PipelineStack(app, 'BacalhauCanaryPipeline', {
        env: {
            account: pipelineConfig.account,
            region: pipelineConfig.region
        },
        lambdaCode: canaryStack.lambdaCode,
        description: 'Bacalhau Canary Pipeline Stack to deploy changes to all canary stacks'
    },
    pipelineConfig);

app.synth();