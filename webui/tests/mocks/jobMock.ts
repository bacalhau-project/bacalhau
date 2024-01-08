import { v4 as uuidv4 } from "uuid";
import { randomBytes } from "crypto";
import { selectRandomLabels, selectRandomElements } from "./mockUtilities";

interface Task {
  Name: string;
  Engine: {
    Type: string;
    Params: {
      Entrypoint: string[];
      EnvironmentVariables: any;
      Image: string;
      Parameters: any;
      WorkingDirectory: string;
    };
  };
  Publisher: {
    Type: string;
  };
  Resources: object;
  Network: {
    Type: string;
  };
  Timeouts: {
    ExecutionTimeout: number;
  };
}

interface Job {
  ID: string;
  Name: string;
  Namespace: string;
  Type: string;
  Priority: number;
  Count: number;
  Constraints: { Key: string; Operator: string; Values: string[] }[];
  Meta: object;
  Labels: object;
  Tasks: Task[];
  State: {
    StateType: string;
  };
  Version: number;
  Revision: number;
  CreateTime: number;
  ModifyTime: number;
}

function createRandomConstraint(): {
  Key: string;
  Operator: string;
  Values: string[];
} {
  const keys = ["region", "owner", "instanceType"];
  const operators = ["=", "!=", ">", "<", "in", "not in"];

  return {
    Key: keys[Math.floor(Math.random() * keys.length)],
    Operator: operators[Math.floor(Math.random() * operators.length)],
    Values: [uuidv4()],
  };
}

// Select a random job type from the list of available types in constants.go
const jobTypes = ["batch", "service", "ops", "system"];
function selectRandomJobType() {
  return selectRandomElements(jobTypes);
}

// Select a random job state from the list of available states in constants.go
const jobStates = [
  "Pending",
  "Running",
  "Completed",
  "Failed",
  "Cancelled",
  "Rejected",
  "Completed",
  "Undefined",
  "Error",
];

// Create a set of random job labels as key value pairs
const jobLabels = {
  canary: ["true", "false"],
  region: ["us-east-1", "us-west-1", "us-west-2", "eu-west-1", "eu-central-1"],
  owner: ["bacalhau", "test", "dev", "prod"],
  instanceType: ["m4.large", "m4.xlarge", "m4.2xlarge", "m4.4xlarge"],
};

function generateSampleTask(): Task {
  // TODO: Implement actual random task
  // For now, just return a static task
  return {
    Name: "main",
    Engine: {
      Type: "docker",
      Params: {
        Entrypoint: ["echo", "hello λ!"],
        EnvironmentVariables: null,
        Image: "ubuntu",
        Parameters: null,
        WorkingDirectory: "",
      },
    },
    Publisher: {
      Type: "ipfs",
    },
    Resources: {},
    Network: {
      Type: "None",
    },
    Timeouts: {
      ExecutionTimeout: Math.floor(Math.random() * 3600), // 1 hour max
    },
  };
}

export function generateSampleJob(): Job {
  const namespace = randomBytes(64).toString("hex");
  return {
    ID: uuidv4(),
    Name: uuidv4(),
    Namespace: namespace,
    Type: selectRandomJobType(),
    Priority: Math.floor(Math.random() * 10),
    Count: Math.floor(Math.random() * 10),
    Constraints: [createRandomConstraint()],
    Meta: {
      "bacalhau.org/client.id": namespace,
      "bacalhau.org/requester.id": `Qm${randomBytes(44).toString("hex")}`,
      "bacalhau.org/requester.publicKey": `CAAS${randomBytes(128).toString(
        // eslint-disable-next-line prettier/prettier
        "base64"
      )}`,
    },
    Labels: { ...selectRandomLabels() },
    Tasks: [generateSampleTask()],
    State: {
      StateType: selectRandomElements(jobStates),
    },
    Version: Math.floor(Math.random() * 10),
    Revision: Math.floor(Math.random() * 10),
    CreateTime: Date.now(),
    ModifyTime: Date.now(),
  };
}
