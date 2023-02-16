import * as cdk from 'aws-cdk-lib';
import * as codebuild from 'aws-cdk-lib/aws-codebuild';
import * as codepipeline from 'aws-cdk-lib/aws-codepipeline';
import * as codepipeline_actions from 'aws-cdk-lib/aws-codepipeline-actions';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import {CanaryConfig, getCanaryConfig, PipelineConfig} from "./config";

export interface PipelineStackProps extends cdk.StackProps {
    readonly lambdaCode: lambda.CfnParametersCode;
}


export class PipelineStack extends cdk.Stack {
    constructor(app: cdk.App, id: string, props: PipelineStackProps, config: PipelineConfig) {
        super(app, id, props)

        // Source artifacts
        const sourceOutput = new codepipeline.Artifact("SourceOutput")

        // Configs
        const prodConfig = getCanaryConfig(app, 'prod');
        const stagingConfig = getCanaryConfig(app, 'staging');
        const allConfigs = [prodConfig, stagingConfig]

        // Build artifacts
        const cloudformationBuild = this.getCloudformationBuild(allConfigs);
        const canaryBuild = this.getCanaryBuild();

        const cdkBuildOutput = new codepipeline.Artifact('CFBuildOutput');
        const canaryBuildOutput = new codepipeline.Artifact('CanaryBuildOutput');

        new codepipeline.Pipeline(this, 'Pipeline' + config.suffix, {
            stages: [
                {
                    stageName: 'Source',
                    actions: [
                        new codepipeline_actions.CodeStarConnectionsSourceAction({
                            actionName: "BacalhauCommit",
                            output: sourceOutput,
                            owner: config.bacalhauSourceConnection.owner,
                            repo: config.bacalhauSourceConnection.repo,
                            branch: config.bacalhauSourceConnection.branch,
                            connectionArn: config.bacalhauSourceConnection.connectionArn,
                        })
                    ],
                },
                {
                    stageName: 'Build',
                    actions: [
                        new codepipeline_actions.CodeBuildAction({
                            actionName: 'BuildCF',
                            project: cloudformationBuild,
                            input: sourceOutput,
                            outputs: [cdkBuildOutput],
                        }),
                        new codepipeline_actions.CodeBuildAction({
                            actionName: 'BuildCanary',
                            project: canaryBuild,
                            input: sourceOutput,
                            outputs: [canaryBuildOutput],
                        })
                    ],
                },
                {
                    stageName: 'PreStagingVerification',
                    actions: [
                        new codepipeline_actions.CodeBuildAction({
                            actionName: 'IntegrationTest',
                            project: this.getIntegrationTest(stagingConfig),
                            input: sourceOutput,
                        }),
                    ],
                },

                {
                    stageName: 'DeployStaging',
                    actions: [
                        new codepipeline_actions.CloudFormationCreateUpdateStackAction({
                            actionName: 'DeployCanary',
                            templatePath: cdkBuildOutput.atPath('BacalhauCanaryStaging.template.json'),
                            stackName: 'BacalhauCanaryStaging',
                            adminPermissions: true,
                            parameterOverrides: {
                                ...props.lambdaCode.assign(canaryBuildOutput.s3Location),
                            },
                            extraInputs: [canaryBuildOutput],
                        }),
                    ],
                },
                {
                    stageName: 'PreProdVerification',
                    actions: [
                        new codepipeline_actions.CodeBuildAction({
                            actionName: 'IntegrationTest',
                            project: this.getIntegrationTest(prodConfig),
                            input: sourceOutput,
                        }),
                        new codepipeline_actions.ManualApprovalAction({
                            actionName: 'ManualApproval',
                            additionalInformation: 'Approve the Canary deployment to production',
                        }),
                    ],
                },
                {
                    stageName: 'DeployProd',
                    actions: [
                        new codepipeline_actions.CloudFormationCreateUpdateStackAction({
                            actionName: 'DeployCanary',
                            templatePath: cdkBuildOutput.atPath('BacalhauCanaryProd.template.json'),
                            stackName: 'BacalhauCanaryProd',
                            adminPermissions: true,
                            parameterOverrides: {
                                ...props.lambdaCode.assign(canaryBuildOutput.s3Location),
                            },
                            extraInputs: [canaryBuildOutput],
                        }),
                    ],
                }
            ],
        });
    }


    private getCanaryBuild() {
        return new codebuild.PipelineProject(this, 'CanaryBuild', {
            buildSpec: codebuild.BuildSpec.fromObject({
                version: '0.2',
                phases: {
                    install: {
                        commands: [
                            'rm -rf `goenv root`',
                            'curl --silent --show-error --location --fail https://go.dev/dl/go1.19.6.linux-amd64.tar.gz | tar --extract --gzip --file=- --directory=/usr/local',
                            'ln -s /usr/local/go/bin/go /usr/local/bin/go',
                        ],
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
    }

    private getCloudformationBuild(configs: CanaryConfig[]) {
        const synthCommands: string[] = new Array(configs.length)
        for (const config of configs) {
            synthCommands.push(`npm run cdk synth -- -c config=${config.env} -o dist`);
        }

        return new codebuild.PipelineProject(this, 'CFBuild', {
            buildSpec: codebuild.BuildSpec.fromObject({
                version: '0.2',
                phases: {
                    install: {
                        commands: [
                            'cd ops/aws/canary',
                            'npm install',
                        ]
                    },
                    pre_build: {
                        commands: [
                            'npm run build',
                        ],
                    },
                    build: {
                        commands: synthCommands,
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
    }

    private getIntegrationTest(config: CanaryConfig) {
        return new codebuild.PipelineProject(this, 'IntegrationTest' + config.envTitle, {
            buildSpec: codebuild.BuildSpec.fromObject({
                version: '0.2',
                env: {
                    variables: {
                        'BACALHAU_ENVIRONMENT': config.bacalhauEnvironment,
                    },
                },
                phases: {
                    install: {
                        commands: [
                            'rm -rf `goenv root`',
                            'curl --silent --show-error --location --fail https://go.dev/dl/go1.19.6.linux-amd64.tar.gz | tar --extract --gzip --file=- --directory=/usr/local',
                            'ln -s /usr/local/go/bin/go /usr/local/bin/go',
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
    }
}