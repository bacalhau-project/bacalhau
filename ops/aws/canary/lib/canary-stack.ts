import * as cdk from 'aws-cdk-lib';
import * as cloudwatch from 'aws-cdk-lib/aws-cloudwatch';
import * as cloudwatchActions from 'aws-cdk-lib/aws-cloudwatch-actions';
import * as events from 'aws-cdk-lib/aws-events';
import * as targets from 'aws-cdk-lib/aws-events-targets';
import * as iam from 'aws-cdk-lib/aws-iam';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as lambdaSources from 'aws-cdk-lib/aws-lambda-event-sources';
import * as sns from 'aws-cdk-lib/aws-sns';
import * as secretsmanager from 'aws-cdk-lib/aws-secretsmanager';
import {CanaryConfig} from "./config";
import {Size} from "aws-cdk-lib";

export interface ScenarioProps {
    readonly action: string;
    readonly timeoutMinutes: number;
    readonly rateMinutes: number;
    readonly memorySize: number;
    readonly storageSize: number;
    readonly evaluationPeriods: number;
    readonly datapointsToAlarm: number;
    readonly availabilityThreshold: number;
    readonly doAlarm: boolean;
    readonly logLevel: string
}

const DEFAULT_SCENARIO_PROPS: ScenarioProps = {
    action: '_',
    timeoutMinutes: 1,
    rateMinutes: 2,
    memorySize: 256,
    storageSize: 512,
    evaluationPeriods: 5,
    datapointsToAlarm: 3,
    availabilityThreshold: 95,
    doAlarm: true,
    logLevel: 'DEBUG',
}

export class CanaryStack extends cdk.Stack {
    public readonly lambdaCode: lambda.CfnParametersCode;
    private readonly config: CanaryConfig;
    private readonly dashboard: cloudwatch.Dashboard
    private readonly snsAlarmTopic: sns.ITopic

    constructor(app: cdk.App, id: string, props: cdk.StackProps, config: CanaryConfig) {
        super(app, id, props)

        this.config = config;
        this.lambdaCode = lambda.Code.fromCfnParameters();
        this.dashboard = new cloudwatch.Dashboard(this, "Dashboard", {
            dashboardName: "BacalhauCanary" + this.config.envTitle
        });
        this.snsAlarmTopic = new sns.Topic(this, 'AlarmTopic');

        this.createLambdaAlarmSlackHandlerFunc()
        this.createLambdaScenarioFunc({ ...DEFAULT_SCENARIO_PROPS, ...{action: "list"}});
        this.createLambdaScenarioFunc({ ...DEFAULT_SCENARIO_PROPS, ...{action: "submit"}});
        this.createLambdaScenarioFunc({ ...DEFAULT_SCENARIO_PROPS, ...{action: "submitAndGet", memorySize: 1024}});
        this.createLambdaScenarioFunc({ ...DEFAULT_SCENARIO_PROPS, ...{action: "submitAndDescribe"}});
        this.createLambdaScenarioFunc({ ...DEFAULT_SCENARIO_PROPS, ...{action: "submitWithConcurrency"}});
        this.createLambdaScenarioFunc({ ...DEFAULT_SCENARIO_PROPS, ...{action: "submitWithConcurrencyOwnedNodes"}});
        this.createLambdaScenarioFunc({ ...DEFAULT_SCENARIO_PROPS, ...{
                action: "submitDockerIPFSJobAndGet", timeoutMinutes: 5, rateMinutes: 5, memorySize: 5120, storageSize: 5012,
                datapointsToAlarm: 4, evaluationPeriods: 6, doAlarm: false}});

        if (config.createOperators) {
            this.createOperatorGroup()
        }
    }

    // Create a lambda function that handles alarms and sends a slack notification
    private createLambdaAlarmSlackHandlerFunc() : lambda.Function {
        const slackSecretes = new secretsmanager.Secret(this, 'SlackWebhooksSecret', {
            description: 'Slack webhook URLs',
            secretObjectValue: {
                webhookUrl: cdk.SecretValue.unsafePlainText('https://...'),
            },
        });

        const func = new lambda.Function(this,  'AlarmHandlerFunction', {
            code: this.lambdaCode,
            handler: 'alarm_handler',
            runtime: lambda.Runtime.GO_1_X,
            timeout: cdk.Duration.minutes(1),
            environment: {
                'DASHBOARD_URL': this.config.dashboardPublicUrl,
                'SLACK_SECRET_NAME': slackSecretes.secretName,
            }
        });
        func.addEventSource(new lambdaSources.SnsEventSource(this.snsAlarmTopic));
        slackSecretes.grantRead(func);
        return func;
    }

    // Create a lambda function that triggers test scenarios
    private createLambdaScenarioFunc(props: ScenarioProps) : lambda.Function {
        const actionTitle = props.action.charAt(0).toUpperCase() + props.action.slice(1)
        const func = new lambda.Function(this, actionTitle + 'Function', {
            code: this.lambdaCode,
            handler: 'scenario_handler',
            runtime: lambda.Runtime.GO_1_X,
            timeout: cdk.Duration.minutes(props.timeoutMinutes),
            memorySize: props.memorySize,
            ephemeralStorageSize: Size.mebibytes(props.storageSize),
            retryAttempts: 0,
            environment: {
                'BACALHAU_DIR': '/tmp', //bacalhau uses $HOME to store configs by default, which doesn't exist in lambda
                'LOG_LEVEL': props.logLevel,
                'BACALHAU_ENVIRONMENT': this.config.bacalhauEnvironment,
            }
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
        this.createAlarm(props, func)
        return func;
    }

    private addDashboardWidgets(actionTitle: string, func: lambda.Function) {
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

    private createAlarm(props: ScenarioProps, func: lambda.Function) {
        const actionTitle = props.action.charAt(0).toUpperCase() + props.action.slice(1)
        const availabilityMetric = this.getAvailabilityMetric(func)
        const alarm = availabilityMetric.createAlarm(this, actionTitle + "Alarm", {
            alarmDescription: actionTitle + ' ' + this.config.envTitle + ' Availability',
            threshold: props.availabilityThreshold,
            comparisonOperator: cloudwatch.ComparisonOperator.LESS_THAN_THRESHOLD,
            evaluationPeriods: props.evaluationPeriods,
            datapointsToAlarm: props.datapointsToAlarm,
            treatMissingData: cloudwatch.TreatMissingData.BREACHING,
        });

        if (props.doAlarm) {
            alarm.addAlarmAction(new cloudwatchActions.SnsAction(this.snsAlarmTopic));
            alarm.addOkAction(new cloudwatchActions.SnsAction(this.snsAlarmTopic));
        }
    }

    private createOperatorGroup() {
        const group = new iam.Group(this, 'OperatorGroup', {
            groupName: 'BacalhauCanaryOperators-' + this.config.envTitle
        })

        // add managed policies
        group.addManagedPolicy(iam.ManagedPolicy.fromAwsManagedPolicyName('CloudWatchReadOnlyAccess'))
        group.addManagedPolicy(iam.ManagedPolicy.fromAwsManagedPolicyName('AWSCloudFormationReadOnlyAccess'))
        group.addManagedPolicy(iam.ManagedPolicy.fromAwsManagedPolicyName('AWSLambda_ReadOnlyAccess'))
        group.addManagedPolicy(iam.ManagedPolicy.fromAwsManagedPolicyName('AmazonEventBridgeReadOnlyAccess'))
        group.addManagedPolicy(iam.ManagedPolicy.fromAwsManagedPolicyName('AmazonEventBridgeSchemasReadOnlyAccess'))
        group.addManagedPolicy(iam.ManagedPolicy.fromAwsManagedPolicyName('AWSCodePipeline_ReadOnlyAccess'))
        group.addManagedPolicy(iam.ManagedPolicy.fromAwsManagedPolicyName('AWSCodeBuildReadOnlyAccess'))
        group.addManagedPolicy(iam.ManagedPolicy.fromAwsManagedPolicyName('AWSCodeDeployReadOnlyAccess'))
        group.addManagedPolicy(iam.ManagedPolicy.fromAwsManagedPolicyName('AWSCodeCommitReadOnly'))
        group.addManagedPolicy(iam.ManagedPolicy.fromAwsManagedPolicyName('IAMUserChangePassword'))

        // Create users and add them to the group
        const users = [
            'kai.davenport',
            'luke.marsden',
            'enrico.rotundo',
        ]

        const initialPassword = new secretsmanager.Secret(this, 'CanaryOperatorsInitialPassword', {
            description: 'Canary Operators Initial Password',
        });

        users.forEach(username => {
            new iam.User(this, 'OperatorUser' + username, {
                userName: username,
                password: initialPassword.secretValue,
                passwordResetRequired: true,
                groups: [group]
            })
        })
    }
}