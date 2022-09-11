import * as cdk from 'aws-cdk-lib';
import * as codedeploy from 'aws-cdk-lib/aws-codedeploy';
import * as cloudwatch from 'aws-cdk-lib/aws-cloudwatch';
import * as events from 'aws-cdk-lib/aws-events';
import * as targets from 'aws-cdk-lib/aws-events-targets';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import {Construct} from 'constructs';

export interface LambdaProps {
    readonly action: string;
    readonly timeoutMinutes: number;
    readonly rateMinutes: number;
    readonly memorySize: number;
}

export class CanaryStack extends cdk.Stack {
    public readonly lambdaCode: lambda.CfnParametersCode;
    private dashboard: cloudwatch.Dashboard


    constructor(scope: Construct, id: string, props?: cdk.StackProps) {
        super(scope, id, props)
        this.lambdaCode = lambda.Code.fromCfnParameters();

        this.dashboard = this.createDashboard();

        this.lambda({action: "list", timeoutMinutes: 1, rateMinutes: 2, memorySize: 256});
        this.lambda({action: "submit", timeoutMinutes: 1, rateMinutes: 2, memorySize: 256});
        this.lambda({action: "submitAndGet", timeoutMinutes: 1, rateMinutes: 2, memorySize: 512});
    }

    createDashboard() {
        const dashboard = new cloudwatch.Dashboard(this, "Dashboard", {
            dashboardName: 'CanaryDashboard',
        })
        // Generate Outputs
        const cloudwatchDashboardURL = `https://${cdk.Aws.REGION}.console.aws.amazon.com/cloudwatch/home?region=${cdk.Aws.REGION}#dashboards:name=${dashboard.dashboardName}`
        new cdk.CfnOutput(this, 'DashboardOutput', {
            value: cloudwatchDashboardURL,
            description: 'URL of Sample CloudWatch Dashboard',
            exportName: 'SampleCloudWatchDashboardURL'
        });
        return dashboard
    }

    // Create a lambda function
    lambda(props: LambdaProps) {
        const actionTitle = props.action.charAt(0).toUpperCase() + props.action.slice(1)
        const func = new lambda.Function(this, actionTitle + 'Function', {
            code: this.lambdaCode,
            handler: 'main',
            runtime: lambda.Runtime.GO_1_X,
            timeout: cdk.Duration.minutes(props.timeoutMinutes),
            memorySize: props.memorySize,
            environment: {
                'BACALHAU_DIR': '/tmp', //bacalhau uses $HOME to store configs by default, which doesn't exist in lambda
                'LOG_LEVEL': 'DEBUG',
            }
        });

        // deployment
        const alias = new lambda.Alias(this, actionTitle + 'FunctionAlias', {
            aliasName: 'Prod',
            version: func.currentVersion,
        });

        new codedeploy.LambdaDeploymentGroup(this, actionTitle + 'FunctionAliasDeploymentGroup', {
            alias,
            deploymentConfig: codedeploy.LambdaDeploymentConfig.ALL_AT_ONCE,
        });

        // EventBridge rules
        const rule = new events.Rule(this, actionTitle + 'EventRule', {
            schedule: events.Schedule.rate(cdk.Duration.minutes(props.rateMinutes)),
        });

        rule.addTarget(new targets.LambdaFunction(func, {
            event: events.RuleTargetInput.fromObject({action: props.action}),
            retryAttempts: 0,
            maxEventAge: cdk.Duration.minutes(1),
        }));

        this.addDashboardWidgets(actionTitle, func);
        return func;
    }

    addDashboardWidgets(actionTitle: string, func: lambda.Function) {
        // Create Title for Dashboard
        this.dashboard.addWidgets(new cloudwatch.TextWidget({
            markdown: '## ' + actionTitle,
            height: 1,
            width: 24
        }))

        // Create CloudWatch Dashboard Widgets: Errors, Invocations, Duration, Throttles
        this.dashboard.addWidgets(
            new cloudwatch.GraphWidget({
                title: "Invocations",
                left: [func.metricInvocations()],
                right: [func.metricErrors()],
                width: 8
            }),
            new cloudwatch.GraphWidget({
                title: "Duration",
                left: [func.metricDuration()],
                width: 8
            }),
            new cloudwatch.GraphWidget({
                title: "Throttles",
                left: [func.metricThrottles()],
                width: 8
            })
        )
    }
}