import * as cdk from 'aws-cdk-lib';
import * as codebuild from 'aws-cdk-lib/aws-codebuild';
import * as codepipeline from 'aws-cdk-lib/aws-codepipeline';
import * as codepipeline_actions from 'aws-cdk-lib/aws-codepipeline-actions';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import {Construct} from 'constructs';
import {BuildConfig, getConfig} from "./build-config";

export interface PipelineStackProps extends cdk.StackProps {
    readonly lambdaCode: lambda.CfnParametersCode;
}


export class PipelineStack extends cdk.Stack {
    constructor(app: cdk.App, id: string, props: PipelineStackProps, config: BuildConfig) {
        super(app, id, props)

        // Source artifacts
        const sourceOutput = new codepipeline.Artifact("SourceOutput")

        // Build artifacts
        const cdkBuild = new codebuild.PipelineProject(this, 'CdkBuild', {
            buildSpec: codebuild.BuildSpec.fromObject({
                version: '0.2',
                phases: {
                    install: {
                        commands: [
                            'cd ops/aws/canary',
                            'npm install',
                        ]
                    },
                    build: {
                        commands: [
                            'npm run build',
                            'npm run cdk synth -- -c config=prod -o dist',
                        ],
                    },
                },
                artifacts: {
                    'base-directory': 'ops/aws/canary/dist',
                    files: [
                        '**/*'
                    ],
                },
            }),
            environment: {
                buildImage: codebuild.LinuxBuildImage.STANDARD_6_0,
            },
        });

        const lambdaBuild = new codebuild.PipelineProject(this, 'LambdaBuild', {
            buildSpec: codebuild.BuildSpec.fromObject({
                version: '0.2',
                phases: {
                    install: {
                        'runtime-versions': {
                            'golang': 1.18
                        },
                    },
                    build: {
                        commands: [
                            'cd ops/aws/canary/lambda',
                            'go build -ldflags="-s -w" -o scenario_handler ./cmd/scenario_lambda_runner/',
                            'go build -ldflags="-s -w" -o alarm_handler ./cmd/alarm_slack_handler/',
                        ],
                    },
                },
                artifacts: {
                    'base-directory': 'ops/aws/canary/lambda',
                    files: [
                        'scenario_handler',
                        'alarm_handler',
                    ],
                },
            }),
            environment: {
                buildImage: codebuild.LinuxBuildImage.AMAZON_LINUX_2_4,
            },
        });

        // Test artifacts
        const canaryIntegrationTest = new codebuild.PipelineProject(this, 'IntegrationTest', {
            buildSpec: codebuild.BuildSpec.fromObject({
                version: '0.2',
                phases: {
                    install: {
                        commands: [
                            'export GOBIN=${HOME}/bin',
                            'export PATH=$GOBIN:$PATH',
                            'go install gotest.tools/gotestsum@v1.8.2',
                        ],
                    },
                    build: {
                        commands: [
                            'cd ops/aws/canary/lambda',
                            'make integration-test',
                        ],
                    },
                },
                reports: {
                    IntegrationTest: {
                        files: [
                            'ops/aws/canary/lambda/tests.xml',
                        ],
                        'discard-paths': 'yes',
                    }
                },
            }),
            environment: {
                buildImage: codebuild.LinuxBuildImage.AMAZON_LINUX_2_4,
            },
        });

        const cdkBuildOutput = new codepipeline.Artifact('CdkBuildOutput');
        const lambdaBuildOutput = new codepipeline.Artifact('LambdaBuildOutput');

        new codepipeline.Pipeline(this, 'Pipeline', {
            stages: [
                {
                    stageName: 'Source',
                    actions: [
                        new codepipeline_actions.CodeStarConnectionsSourceAction({
                            actionName: "Bacalhau_Commit",
                            output: sourceOutput,
                            owner: config.bacalhauSourceConnection.owner,
                            repo: config.bacalhauSourceConnection.repo,
                            branch: config.bacalhauSourceConnection.branch,
                            connectionArn: config.bacalhauSourceConnection.connectionArn,
                        })
                    ],
                },
                {
                    stageName: 'Test',
                    actions: [
                        new codepipeline_actions.CodeBuildAction({
                            actionName: 'Integration_Test',
                            project: canaryIntegrationTest,
                            input: sourceOutput,
                        }),
                    ],
                },
                {
                    stageName: 'Build',
                    actions: [
                        new codepipeline_actions.CodeBuildAction({
                            actionName: 'CDK_Build',
                            project: cdkBuild,
                            input: sourceOutput,
                            outputs: [cdkBuildOutput],
                        }),
                        new codepipeline_actions.CodeBuildAction({
                            actionName: 'Lambda_Code_Build',
                            project: lambdaBuild,
                            input: sourceOutput,
                            outputs: [lambdaBuildOutput],
                        })
                    ],
                },
                {
                    stageName: 'DeployProd',
                    actions: [
                        new codepipeline_actions.CloudFormationCreateUpdateStackAction({
                            actionName: 'Canary_Deploy_Prod',
                            templatePath: cdkBuildOutput.atPath('BacalhauCanaryProd.template.json'),
                            stackName: 'BacalhauCanaryProd',
                            adminPermissions: true,
                            parameterOverrides: {
                                ...props.lambdaCode.assign(lambdaBuildOutput.s3Location),
                            },
                            extraInputs: [lambdaBuildOutput],
                        }),
                    ],
                }
            ],
        });
    }
}