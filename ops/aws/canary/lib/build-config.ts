import * as cdk from "aws-cdk-lib";

// These values map to the keys in our cdk.json and make sure
// no values that aren't supported are passed in with the
// -c config=env flag
const supportedEnvironments = ["prod"] as const;

type SupportedEnvironments = typeof supportedEnvironments[number];

// This maps to the values you specified in your cdk.json file
// if you add any values to your cdk.json file, also add them here!
export type BuildConfig = {
    readonly env: SupportedEnvironments;
    readonly envTitle: string;
    readonly bacalhauEnvironment: string;
    readonly account: string;
    readonly region: string;
    readonly dashboardPublicUrl: string;
    readonly bacalhauSourceConnection: SourceConnectionProps
};

export type SourceConnectionProps = {
    readonly owner: string;
    readonly repo: string;
    readonly branch: string;
    readonly connectionArn: string;
}

// This function is used by your CDK app and pulls your config values
// from the context
export const getConfig = (app: cdk.App, forceEnv?: any): BuildConfig => {
    const env = forceEnv || app.node.tryGetContext("config");

    if (!env) {
        throw new Error(
            "Context variable defining the environment must be passed to cdk: `cdk -c config=XXX`"
        );
    }
    if (!supportedEnvironments.includes(env)) {
        throw new Error(
            `${env} is not in supported environments: ${supportedEnvironments.join(
                ", "
            )}`
        );
    }
    // this contains the values in the context without being
    // validated
    const unparsedEnv = app.node.tryGetContext(env);
    const envTitle = env.charAt(0).toUpperCase() + env.slice(1);

    return {
        env: env,
        envTitle: envTitle,
        bacalhauEnvironment: ensureString(unparsedEnv, "bacalhauEnvironment"),
        account: ensureString(unparsedEnv, "account"),
        region: ensureString(unparsedEnv, "region"),
        dashboardPublicUrl: ensureString(unparsedEnv, "dashboardPublicUrl"),
        bacalhauSourceConnection: {
            owner: ensureString(unparsedEnv['bacalhauSourceConnection'], "owner"),
            repo: ensureString(unparsedEnv['bacalhauSourceConnection'], "repo"),
            branch: ensureString(unparsedEnv['bacalhauSourceConnection'], "branch"),
            connectionArn: ensureString(unparsedEnv['bacalhauSourceConnection'], "connectionArn"),
        }
    };
};

// this function ensures that the value from the config is
// the correct type. If you have any types other than
// strings be sure to create a new validation function
function ensureString(object: { [name: string]: any },
                      key: keyof BuildConfig | keyof SourceConnectionProps): string {
    if (!object[key] ||
        typeof object[key] !== "string" ||
        object[key].trim().length === 0) {
        throw new Error(key + " does not exist in cdk config");
    }
    return object[key];
}