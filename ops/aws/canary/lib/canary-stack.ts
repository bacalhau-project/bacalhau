import * as cdk from 'aws-cdk-lib';
import * as codedeploy from 'aws-cdk-lib/aws-codedeploy';
import * as events from 'aws-cdk-lib/aws-events';
import * as targets from 'aws-cdk-lib/aws-events-targets';
import * as sqs from 'aws-cdk-lib/aws-sqs';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import {Construct} from 'constructs';


export class CanaryStack extends cdk.Stack {
    public readonly lambdaCode: lambda.CfnParametersCode;

    constructor(scope: Construct, id: string, props?: cdk.StackProps) {
        super(scope, id, props)
        this.lambdaCode = lambda.Code.fromCfnParameters();

        this.lambda()
    }

    // Create a lambda function
    lambda() {
        const dlq = new sqs.Queue(this, "LambdaDlq", {
            retentionPeriod: cdk.Duration.days(7),
            visibilityTimeout: cdk.Duration.minutes(5)
        });

        const func = new lambda.Function(this, 'Lambda', {
            code: this.lambdaCode,
            handler: 'handler',
            runtime: lambda.Runtime.GO_1_X,
            deadLetterQueue: dlq,
            timeout: cdk.Duration.minutes(5),
            environment: {
                'BACALHAU_DIR': '/tmp' // bacalhau uses $HOME to store configs by default, which doesn't exist in lambda
            }
        });

        // deployment
        const alias = new lambda.Alias(this, 'LambdaAlias', {
            aliasName: 'Prod',
            version: func.currentVersion,
        });

        new codedeploy.LambdaDeploymentGroup(this, 'LambdaAliasDeploymentGroup', {
            alias,
            deploymentConfig: codedeploy.LambdaDeploymentConfig.ALL_AT_ONCE,
        });

        const rule = new events.Rule(this, 'LambdaRule', {
            ruleName: 'MyRule',
            schedule: events.Schedule.rate(cdk.Duration.minutes(1)),
            targets: [new targets.LambdaFunction(func)]
        });

        rule.addTarget(new targets.LambdaFunction(func, {
            retryAttempts: 0,
            maxEventAge: cdk.Duration.minutes(1)
        }));

        return func;
    }


}