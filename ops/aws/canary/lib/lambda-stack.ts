import * as cdk from 'aws-cdk-lib';
import * as codedeploy from 'aws-cdk-lib/aws-codedeploy';
import * as sqs from 'aws-cdk-lib/aws-sqs';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import {Construct} from 'constructs';


export class LambdaStack extends cdk.Stack {
    public readonly lambdaCode: lambda.CfnParametersCode;

    constructor(scope: Construct, id: string, props?: cdk.StackProps) {
        super(scope, id, props)
        this.lambdaCode = lambda.Code.fromCfnParameters();

        const dlq = new sqs.Queue(this, "Dlq", {
            retentionPeriod: cdk.Duration.days(7),
            visibilityTimeout: cdk.Duration.minutes(5)
        });

        const func = new lambda.Function(this, 'Function', {
            code: this.lambdaCode,
            handler: 'main',
            runtime: lambda.Runtime.GO_1_X,
            deadLetterQueue: dlq,
            timeout: cdk.Duration.minutes(1)
        });

        // deployment
        const alias = new lambda.Alias(this, 'Alias', {
            aliasName: 'Prod',
            version: func.currentVersion,
        });

        new codedeploy.LambdaDeploymentGroup(this, 'AliasDeploymentGroup', {
            alias,
            deploymentConfig: codedeploy.LambdaDeploymentConfig.ALL_AT_ONCE,
        });
    }
}