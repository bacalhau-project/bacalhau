// interfaces.ts

export interface JobsResponse {
  NextToken: string;
  Jobs: Job[];
}

export interface JobResponse {
  Job: Job;
}

export interface JobExecutionsResponse {
  NextToken: string;
  Executions: Execution[];
}

export interface Job {
  ID: string;
  Name: string;
  Namespace: string;
  Type: string;
  Priority: number;
  Count: number;
  Constraints: object[];
  Meta: {
    [key: string]: string; // Assuming all values in Meta are strings
  };
  Labels: { [key: string]: string }; // Assuming all values in Labels are strings
  Tasks: Tasks[];
  State: {
    StateType: string;
    Message: string;
  };
  Version: number;
  Revision: number;
  CreateTime: number;
  ModifyTime: number;
}

export interface Tasks {
  Name: string;
  Engine: Engine;
  Publisher: {
    Type: string;
  };
  ResultPaths: {
    Name: string;
    Path: string;
  }[];
  Resources: Resources;
  Network: {
    Type: string;
  };
  Timeouts: {
    ExecutionTimeout: number;
  };
}

export interface Engine {
  Type: string;
  Params: Partial<EngineParams>;
}

export interface EngineParams {
  Entrypoint: string;
  EnvironmentVariables: string[];
  Image: string;
  Parameters: string[];
  WorkingDirectory: string;
}

export interface Resources {
  CPU: string;
  Memory: string;
  Disk: string;
  GPU: string;
}

export interface ParsedJobData {
  id: string;
  longId: string;
  name: string;
  createdAt: Date;
  tasks: Tasks;
  jobType: string;
  label: string;
  status: string;
  action: string;
}

export interface JobListRequest {
  order_by: string | undefined;
  reverse: boolean | undefined;
  limit: number | undefined;
  labels: undefined | string;
  next_token: undefined | string;
}

export interface Execution {
  ID: string;
  Namespace: string;
  EvalID: string;
  Name: string;
  NodeID: string;
  JobID: string;
  AllocatedResources: AllocatedResources;
  DesiredState: StateInfo;
  ComputeState: StateInfo;
  PublishedResult: PublishedResult;
  RunOutput: RunOutput;
  PreviousExecution: string;
  NextExecution: string;
  FollowupEvalID: string;
  Revision: number;
  CreateTime: number;
  ModifyTime: number;
}

interface AllocatedResources {
  Tasks: Record<string, unknown>; // Assuming Tasks is an object with dynamic keys
}

interface StateInfo {
  StateType: number;
  Message: string;
}

interface PublishedResult {
  Type: string;
  Params: null | Record<string, unknown>; // Assuming Params can be an object with dynamic keys
}

interface RunOutput {
  Stdout: string;
  StdoutTruncated: boolean;
  stderr: string;
  StderrTruncated: boolean;
  exitCode: number;
  runnerError: string;
}