import * as cdk from "aws-cdk-lib";

// These values map to the keys in our cdk.json and make sure
// no values that aren't supported are passed in with the
// -c config=env flag
const supportedEnvironments = ["prod", "prodOwned", "staging"] as const;

type SupportedEnvironments = typeof supportedEnvironments[number];

// This maps to the values you specified in your cdk.json file
// if you add any values to your cdk.json file, also add them here!
export type CanaryConfig = {
    readonly env: SupportedEnvironments;
    readonly envTitle: string;
    readonly bacalhauEnvironment: string;
    readonly nodeSelectors: string;
    readonly account: string;
    readonly region: string;
    readonly createOperators: boolean;
    readonly dashboardPublicUrl: string;
};

export type PipelineConfig = {
    readonly env: SupportedEnvironments;
    readonly suffix: string;
    readonly account: string;
    readonly region: string;
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
export const getCanaryConfig = (app: cdk.App, forceEnv?: any): CanaryConfig => {
    const node = app.node.tryGetContext("canary");
    if (!node) {
        throw new Error(
            "`canary` is missing in cdk context"
        );
    }

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
    const unparsedEnv = node[env];
    const envTitle = env.charAt(0).toUpperCase() + env.slice(1);

    return {
        env: env,
        envTitle: envTitle,
        bacalhauEnvironment: ensureString(unparsedEnv, "bacalhauEnvironment"),
        nodeSelectors: unparsedEnv["nodeSelectors"],
        account: ensureString(unparsedEnv, "account"),
        region: ensureString(unparsedEnv, "region"),
        createOperators: ensureBool(unparsedEnv, "createOperators"),
        dashboardPublicUrl: ensureString(unparsedEnv, "dashboardPublicUrl"),
    };
};

// This function is used by your CDK app and pulls your config values
// from the context
export const getPipelineConfig = (app: cdk.App, forceEnv?: any): PipelineConfig => {
    const node = app.node.tryGetContext("pipeline");
    if (!node) {
        throw new Error(
            "`pipeline` is missing in cdk context"
        );
    }
    const env = forceEnv || app.node.tryGetContext("config") || "prod";

    if (!supportedEnvironments.includes(env)) {
        throw new Error(
            `${env} is not in supported environments: ${supportedEnvironments.join(
                ", "
            )}`
        );
    }
    // this contains the values in the context without being
    // validated
    const unparsedEnv = node[env];

    return {
        env: env,
        suffix: unparsedEnv["suffix"], // suffix can be blank
        account: ensureString(unparsedEnv, "account"),
        region: ensureString(unparsedEnv, "region"),
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
                      key: keyof CanaryConfig | keyof PipelineConfig | keyof SourceConnectionProps): string {
    if (!(key in object) ||
        typeof object[key] !== "string" ||
        object[key].trim().length === 0) {
        throw new Error(key + " does not exist in config: " + JSON.stringify(object));
    }
    return object[key];
}

function ensureBool(object: { [name: string]: any },
                      key: keyof CanaryConfig | keyof PipelineConfig | keyof SourceConnectionProps): boolean {
    if (!(key in object) || typeof object[key] !== "boolean") {
        throw new Error(key + " does not exist in config: " + JSON.stringify(object));
    }
    return object[key];
}