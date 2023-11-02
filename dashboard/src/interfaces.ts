// src/interfaces.ts

export interface Job {
  Job: {
    Metadata: JobMetadata;
    Spec: JobSpec;
  };
  State: JobState;
}

export interface JobMetadata {
  ID: string;
  CreatedAt: string;
}

export interface JobSpec {
  EngineSpec: EngineSpec;
}

export interface EngineSpec {
  Type: string;
  Params: Params;
}

export interface JobState {
  State: string;
}

export interface Params {
  Image: string;
  Parameters: string[];
}
