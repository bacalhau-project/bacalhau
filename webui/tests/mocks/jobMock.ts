import { randomBytes, randomUUID } from "crypto"
import { Job, Task } from "../../src/helpers/jobInterfaces"
import { selectRandomElements, selectRandomKeyAndValue } from "./mockUtilities"

function createRandomConstraint(): {
  Key: string
  Operator: string
  Values: string[]
} {
  const keys = ["region", "owner", "instanceType"]
  const operators = ["=", "!=", ">", "<", "in", "not in"]

  return {
    Key: keys[Math.floor(Math.random() * keys.length)],
    Operator: operators[Math.floor(Math.random() * operators.length)],
    Values: [randomUUID()],
  }
}

// Select a random job type from the list of available types in constants.go
const jobTypes = ["batch", "service", "ops", "system"]
function selectRandomJobType() {
  return selectRandomElements(jobTypes)
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
]

// Create a set of random job labels as key value pairs
const jobLabels: { [key: string]: string[] } = {
  canary: ["true", "false"],
  region: ["us-east-1", "us-west-1", "us-west-2", "eu-west-1", "eu-central-1"],
  owner: ["bacalhau", "test", "dev", "prod"],
  instanceType: ["m4.large", "m4.xlarge", "m4.2xlarge", "m4.4xlarge"],
}

function generateSampleTask(): Task {
  // TODO: Implement actual random task
  // For now, just return a static task
  return {
    Name: "main",
    Engine: {
      Type: "docker",
      Params: {
        Entrypoint: "echo",
        EnvironmentVariables: [],
        Image: "ubuntu",
        Parameters: [],
        WorkingDirectory: "",
      },
    },
    Publisher: {
      Type: "ipfs",
    },
    Resources: {
      CPU: "1",
      Disk: "2",
      Memory: "3",
      GPU: "0",
    },
    Network: {
      Type: "None",
    },
    Timeouts: {
      ExecutionTimeout: Math.floor(Math.random() * 3600), // 1 hour max
    },
    ResultPaths: [{ Name: "tmp", Path: "/tmp" }],
  }
}

export function generateSampleJob(): Job {
  const namespace = randomBytes(64).toString("hex")
  return {
    ID: randomUUID(),
    Name: randomUUID(),
    Namespace: namespace,
    Type: selectRandomJobType() as string,
    Priority: Math.floor(Math.random() * 10),
    Count: Math.floor(Math.random() * 10),
    Constraints: [createRandomConstraint()],
    Meta: {
      "bacalhau.org/client.id": namespace,
      "bacalhau.org/requester.id": `Qm${randomBytes(44).toString("hex")}`,
      // cspell: disable-next-line
      "bacalhau.org/requester.publicKey": `CAAS${randomBytes(128).toString(
        // eslint-disable-next-line prettier/prettier
        "base64"
      )}`,
    },
    Labels: { ...selectRandomKeyAndValue(jobLabels) },
    Tasks: [generateSampleTask()],
    State: {
      StateType: selectRandomElements(jobStates) as string,
      Message: "State Message",
    },
    Version: Math.floor(Math.random() * 10),
    Revision: Math.floor(Math.random() * 10),
    CreateTime: Date.now(),
    ModifyTime: Date.now(),
  }
}
