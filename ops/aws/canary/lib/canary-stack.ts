import * as cdk from 'aws-cdk-lib';
import * as codedeploy from 'aws-cdk-lib/aws-codedeploy';
import * as cloudwatch from 'aws-cdk-lib/aws-cloudwatch';
import * as events from 'aws-cdk-lib/aws-events';
import * as targets from 'aws-cdk-lib/aws-events-targets';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import {Construct} from 'constructs';
import {IMetric} from "aws-cdk-lib/aws-cloudwatch/lib/metric-types";

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
        this.lambda({action: "submitAndDescribe", timeoutMinutes: 1, rateMinutes: 2, memorySize: 256});
        this.lambda({action: "submitWithConcurrency", timeoutMinutes: 1, rateMinutes: 2, memorySize: 256});
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
    lambda(props: LambdaProps) : lambda.Function {
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
        this.createAlarm(actionTitle, func)
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
                width: 8
            }),
            new cloudwatch.GraphWidget({
                title: "Duration",
                left: [func.metricDuration({label: "[avg: ${AVG}ms, max: ${MAX}ms] Duration"})],
                width: 8
            }),
            new cloudwatch.GraphWidget({
                title: "Error count and success rate (%)",
                left: [func.metricErrors()],
                right: [this.getAvailabilityMetric(func)],
                rightYAxis: {min: 0, max: 100},
                width: 8
            })
        )
    }

    private getAvailabilityMetric(func: lambda.Function) : cloudwatch.MathExpression {
        return new cloudwatch.MathExpression({
            expression: "100 - 100 * errors / MAX([errors, invocations])",
            label: "[avg: ${AVG}] Success rate",
            usingMetrics: {
                errors: func.metricErrors(),
                invocations: func.metricInvocations()
            }
        })
    }
    private createAlarm(actionTitle: string, func: lambda.Function) {
        const threshold = 95
        const availabilityMetric = this.getAvailabilityMetric(func)
        const alarm = availabilityMetric.createAlarm(this, actionTitle + "Alarm", {
            alarmDescription: actionTitle + ' availability is below ' + threshold + '%',
            threshold: threshold,
            comparisonOperator: cloudwatch.ComparisonOperator.LESS_THAN_THRESHOLD,
            evaluationPeriods: 3,
            datapointsToAlarm: 2,
            treatMissingData: cloudwatch.TreatMissingData.BREACHING,
        });
    }
}