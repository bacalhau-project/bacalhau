import * as cdk from 'aws-cdk-lib';
import * as codedeploy from 'aws-cdk-lib/aws-codedeploy';
import * as cloudwatch from 'aws-cdk-lib/aws-cloudwatch';
import * as cloudwatchActions from 'aws-cdk-lib/aws-cloudwatch-actions';
import * as events from 'aws-cdk-lib/aws-events';
import * as targets from 'aws-cdk-lib/aws-events-targets';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as lambdaSources from 'aws-cdk-lib/aws-lambda-event-sources';
import * as sns from 'aws-cdk-lib/aws-sns';
import {Construct} from 'constructs';

export interface LambdaProps {
    readonly action: string;
    readonly timeoutMinutes: number;
    readonly rateMinutes: number;
    readonly memorySize: number;
}

export class CanaryStack extends cdk.Stack {
    public readonly lambdaCode: lambda.CfnParametersCode;
    private readonly dashboard: cloudwatch.Dashboard
    private readonly dashboardUrl: string
    private readonly snsAlarmTopic: sns.ITopic

    constructor(scope: Construct, id: string, props?: cdk.StackProps) {
        super(scope, id, props)
        this.lambdaCode = lambda.Code.fromCfnParameters();

        this.dashboard = this.createDashboard();
        this.snsAlarmTopic = new sns.Topic(this, 'AlarmTopic');

        this.lambdaAlarmHandlerFunc()
        this.lambdaScenarioFunc({action: "list", timeoutMinutes: 1, rateMinutes: 2, memorySize: 256});
        this.lambdaScenarioFunc({action: "submit", timeoutMinutes: 1, rateMinutes: 2, memorySize: 256});
        this.lambdaScenarioFunc({action: "submitAndGet", timeoutMinutes: 1, rateMinutes: 2, memorySize: 512});
        this.lambdaScenarioFunc({action: "submitAndDescribe", timeoutMinutes: 1, rateMinutes: 2, memorySize: 256});
        this.lambdaScenarioFunc({action: "submitWithConcurrency", timeoutMinutes: 1, rateMinutes: 2, memorySize: 256});
    }

    createDashboard() {
        const dashboard = new cloudwatch.Dashboard(this, "Dashboard", {
            dashboardName: 'CanaryDashboard',
        })
        // Generate Outputs
        new cdk.CfnOutput(this, 'DashboardOutput', {
            value: this.getDashboardUrl(dashboard),
            description: 'URL of Sample CloudWatch Dashboard',
            exportName: 'DashboardURL'
        });
        return dashboard
    }

    getDashboardUrl(dashboard : cloudwatch.Dashboard) {
        return `https://${cdk.Aws.REGION}.console.aws.amazon.com/cloudwatch/home?region=${cdk.Aws.REGION}#dashboards:name=${dashboard.dashboardName}`
    }

    // Create a lambda function that triggers test scenarios
    lambdaAlarmHandlerFunc() : lambda.Function {
        const func = new lambda.Function(this,  'AlarmHandlerFunction', {
            code: this.lambdaCode,
            handler: 'alarm_handler',
            runtime: lambda.Runtime.GO_1_X,
            timeout: cdk.Duration.minutes(1),
            environment: {
                'DASHBOARD_URL': this.getDashboardUrl(this.dashboard),
            }
        });
        func.addEventSource(new lambdaSources.SnsEventSource(this.snsAlarmTopic));
        return func;
    }

    // Create a lambda function that triggers test scenarios
    lambdaScenarioFunc(props: LambdaProps) : lambda.Function {
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

        alarm.addAlarmAction(new cloudwatchActions.SnsAction(this.snsAlarmTopic));
    }
}