# Bacalhau Monitoring Canary
This is a canary that continuously call several Bacalhau APIs and alarm whenever the correctness or availability of those APIs fall below a threshold.

The canary is serverless using AWS Lambda. Infrastructure is defined using AWS CDK, and automatically deployed using AWS CodePipeline.

The monitoring dashboard is publicly accessible at [link](https://cloudwatch.amazonaws.com/dashboard.html?dashboard=BacalhauCanaryProd&context=eyJSIjoidXMtZWFzdC0xIiwiRCI6ImN3LWRiLTI4NDMwNTcxNzgzNSIsIlUiOiJ1cy1lYXN0LTFfUTlPMEVrM3llIiwiQyI6IjExc3NlYW1tZmVmaGdtYTFzMDk1c29jaDltIiwiSSI6InVzLWVhc3QtMTpmNGE5MGFiMi0yZWYwLTRlYTEtOWZkNS1jMmQ3MDkxYTA5OTQiLCJNIjoiUHVibGljIn0=).

## Infrastructure Stacks
There are two types of stacks in this project:
- Canary stack(s): one stack per environment (e.g. prod, dev), containing the Lambda function and the CloudWatch alarm.
- Pipeline stack: contains the CodePipeline and CodeBuild resources.

### Deploying Canary Stacks Changes
Changes to the canary stacks are automatically deployed as soon a new commit is pushed to the main branch. You *should not* deploy this stack manually.

**Note:** Currently only the prod stack is deployed.

### Deploying Pipeline Stack Changes
Changes to the pipeline such as adding a new stage or modifying the build scripts needs to be deployed manually. To do so, run the following command:
```bash
# Assuming you have the AWS CLI installed and configured with a profile named "bacalhau"
# Assuming you are in the ops/aws/canary directory
cdk --profile bacalhau deploy BacalhauCanaryPipeline -c config=prod
```
Note that we only have a single pipeline stack deployed using prod environment configuration, but it will deploy all canary stacks.

### Manual Resources
These are the resources that had to be created/updated manually outside of CDK:
1. GitHub Connection
2. CloudWatch public dashboard link
3. Update secret manager with Slack webhook URL