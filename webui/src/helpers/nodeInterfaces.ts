// src/interfaces.ts

export interface NodesResponse {
  NextToken: string
  Nodes: Node[]
}

export interface Node {
  PeerInfo: PeerInfo
  NodeType: string
  Labels: Labels
  ComputeNodeInfo: ComputeNodeInfo
  BacalhauVersion: BacalhauVersion
}

export interface PeerInfo {
  ID: string
  Addrs: string[]
}

export interface Labels {
  Architecture: string
  "Operating-System": string
  "git-lfs": string
  name: string
  env: string
}

export interface ComputeNodeInfo {
  ExecutionEngines: string[]
  Publishers: string[]
  StorageSources: string[]
  MaxCapacity: ResourceCapacity
  AvailableCapacity: ResourceCapacity
  MaxJobRequirements: ResourceCapacity
  RunningExecutions: number
  EnqueuedExecutions: number
}

export interface ResourceCapacity {
  CPU: number
  Memory: number
  Disk: number
}

export interface BacalhauVersion {
  Major: string
  Minor: string
  GitVersion: string
  GitCommit: string
  BuildDate: string
  GOOS: string
  GOARCH: string
}

export interface ParsedNodeData {
  id: string
  type: string
  environment: string
  inputs: string[]
  outputs: string[]
  version: string
  // nodeHealth: string; // TODO: Add when available
  // healthCheck: string; // TODO: Add when available
  // action: string;
}

export interface NodeListRequest {
  labels: string | undefined
}
