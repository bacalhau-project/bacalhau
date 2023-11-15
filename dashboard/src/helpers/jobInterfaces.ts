// src/interfaces.ts

export interface JobsResponse {
  NextToken: string;
  Jobs: Job[];
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
  };
  Version: number;
  Revision: number;
  CreateTime: number;
  ModifyTime: number;
}

export interface Tasks {
  Name: string;
  Engine: {
    Type: string;
    Params: {
      Entrypoint: null | string;
      EnvironmentVariables: string[];
      Image: string;
      Parameters: string[];
      WorkingDirectory: string;
    };
  };
  Publisher: {
    Type: string;
  };
  ResultPaths: {
    Name: string;
    Path: string;
  }[];
  Resources: object;
  Network: {
    Type: string;
  };
  Timeouts: {
    ExecutionTimeout: number;
  };
}

export interface Engine {
  Type: string;
  Params: EngineParams;
}

export interface EngineParams {
  Entrypoint: null | string;
  EnvironmentVariables: string[];
  Image: string;
  Parameters: string[];
  WorkingDirectory: string;
}

export interface ParsedJobData {
  id: string;
  name: string;
  createdAt: string;
  tasks: Tasks;
  jobType: string;
  label: string;
  status: string;
  action: string;
}

export interface JobListRequest {
  labels: undefined | string;
  next_token: undefined | string;
}
