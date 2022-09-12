import * as cdk from 'aws-cdk-lib';
import * as codebuild from 'aws-cdk-lib/aws-codebuild';
import * as codepipeline from 'aws-cdk-lib/aws-codepipeline';
import * as codepipeline_actions from 'aws-cdk-lib/aws-codepipeline-actions';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import {Construct} from 'constructs';

export interface PipelineStackProps extends cdk.StackProps {
    readonly lambdaCode: lambda.CfnParametersCode;
    readonly repositoryName: string;
}


export class PipelineStack extends cdk.Stack {
    constructor(scope: Construct, id: string, props: PipelineStackProps) {
        super(scope, id, props)

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
                            'npm run cdk synth -- -o dist',
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
                            owner: "wdbaruni",
                            repo: "bacalhau",
                            branch: "canary",
                            connectionArn: "arn:aws:codestar-connections:eu-west-1:284305717835:connection/6a4a94b6-0388-4b0b-acf7-d8feefedd5b6",
                        })
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
                    stageName: 'Deploy',
                    actions: [
                        new codepipeline_actions.CloudFormationCreateUpdateStackAction({
                            actionName: 'Canary_CFN_Deploy',
                            templatePath: cdkBuildOutput.atPath('BacalhauCanary.template.json'),
                            stackName: 'BacalhauCanary',
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