import * as cdk from 'aws-cdk-lib';
import * as codedeploy from 'aws-cdk-lib/aws-codedeploy';
import * as cloudwatch from 'aws-cdk-lib/aws-cloudwatch';
import * as events from 'aws-cdk-lib/aws-events';
import * as targets from 'aws-cdk-lib/aws-events-targets';
import * as sqs from 'aws-cdk-lib/aws-sqs';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import {Construct} from 'constructs';


export class CanaryStack extends cdk.Stack {
    public readonly lambdaCode: lambda.CfnParametersCode;
    private dashboard: cloudwatch.Dashboard


    constructor(scope: Construct, id: string, props?: cdk.StackProps) {
        super(scope, id, props)
        this.lambdaCode = lambda.Code.fromCfnParameters();

        this.dashboard = this.createDashboard();

        this.lambda("list", cdk.Duration.minutes(1));
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
    lambda(action: string, rate: cdk.Duration) {
        const actionTitle = action.charAt(0).toUpperCase() + action.slice(1)
        const func = new lambda.Function(this, actionTitle + 'Function', {
            code: this.lambdaCode,
            handler: 'main',
            runtime: lambda.Runtime.GO_1_X,
            timeout: cdk.Duration.minutes(5),
            environment: {
                'BACALHAU_DIR': '/tmp' // bacalhau uses $HOME to store configs by default, which doesn't exist in lambda
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

        // eventbridge rules
        const rule = new events.Rule(this, actionTitle + 'EventRule', {
            schedule: events.Schedule.rate(rate),
            targets: [new targets.LambdaFunction(func)]
        });

        rule.addTarget(new targets.LambdaFunction(func, {
            event: events.RuleTargetInput.fromObject({action: action}),
            retryAttempts: 0,
            maxEventAge: cdk.Duration.minutes(1)
        }));

        return func;
    }

    addDashboardWidgets(func: lambda.Function) {
        // Create Title for Dashboard
        this.dashboard.addWidgets(new cloudwatch.TextWidget({
            markdown: `# Dashboard: ${func.functionName}`,
            height: 1,
            width: 24
        }))

        // Create CloudWatch Dashboard Widgets: Errors, Invocations, Duration, Throttles
        this.dashboard.addWidgets(new cloudwatch.GraphWidget({
            title: "Invocations",
            left: [func.metricInvocations()],
            width: 24
        }))

        this.dashboard.addWidgets(new cloudwatch.GraphWidget({
            title: "Errors",
            left: [func.metricErrors()],
            width: 24
        }))

        this.dashboard.addWidgets(new cloudwatch.GraphWidget({
            title: "Duration",
            left: [func.metricDuration()],
            width: 24
        }))

        this.dashboard.addWidgets(new cloudwatch.GraphWidget({
            title: "Throttles",
            left: [func.metricThrottles()],
            width: 24
        }))

        // Create Widget to show last 20 Log Entries
        this.dashboard.addWidgets(new cloudwatch.LogQueryWidget({
            logGroupNames: [func.logGroup.logGroupName],
            queryLines:[
                "fields @timestamp, @message",
                "sort @timestamp desc",
                "limit 20"],
            width: 24,
        }))
    }
}