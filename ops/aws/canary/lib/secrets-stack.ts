import * as secretsmanager from 'aws-cdk-lib/aws-secretsmanager';
import {Stack, App, StackProps, Duration} from "aws-cdk-lib"


export class SecretsStack extends Stack {
    public readonly githubToken: secretsmanager.Secret;

    constructor(app: App, id: string, props?: StackProps) {
        super(app, id, props);
        this.githubToken = new secretsmanager.Secret(this, 'SecretGithubToken');
    }

}