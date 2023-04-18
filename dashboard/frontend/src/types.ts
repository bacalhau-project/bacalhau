export interface ResourceUsageConfig {
  CPU?: string,
  Memory?: string,
  Disk?: string,
  GPU: string,
}

export interface ResourceUsageData {
  CPU?: number,
  Memory?: number,
  Disk?: number,
  GPU?: number,
}

export interface ResourceUsageProfile {
  Job?: ResourceUsageData,
  SystemUsing?: ResourceUsageData,
  SystemTotal?: ResourceUsageData,
}

export interface RunCommandResult {
  stdout: string,
  stdouttruncated: boolean,
  stderr: string,
  stderrtruncated: boolean,
  exitCode: number,
  runnerError: string,
}

export interface StorageSpec {
  StorageSource: string,
  Name?: string,
  CID?: string,
  URL?: string,
  path?: string,
  Metadata?: { [key: string]: string },
}

export interface PublishedResult {
  NodeID?: string,
  ShardIndex?: number,
  Data?: StorageSpec,
}

export interface JobMetadata {
  ID: string,
  CreatedAt: string,
  ClientID?: string,
}

export interface JobRequester {
  RequesterNodeID: string,
  RequesterPublicKey?: string,
}

export interface JobStatus {
  JobState: JobState,
  JobEvents: JobEvent[],
  LocalJobEvents: JobLocalEvent[],
  Requester: JobRequester,
}

export interface Job {
  APIVersion: string,
  Metadata: JobMetadata,
  Spec: Spec,
  Status: JobStatus,
}

export interface JobWithInfo {
  Job: Job,
  JobState: JobState,
  JobEvents: JobEvent[],
  JobLocalEvents: JobLocalEvent[],
}

export interface JobShard {
  Job?: Job,
  Index?: number,
}

export interface JobExecutionPlan {
  ShardsTotal?: number,
}

export interface JobShardingConfig {
  GlobPattern?: string,
  BatchSize?: number,
  GlobPatternBasePath?: string,
}

export interface JobState {
  Nodes: { [key: string]: JobNodeState },
}

export interface JobNodeState {
  Shards: { [key: number]: JobShardState },
}

export interface JobShardState {
  NodeId: string,
  ShardIndex: number,
  State: string,
  Status: string,
  VerificationProposal?: string,
  VerificationResult?: VerificationResult,
  PublishedResults: StorageSpec,
  RunOutput?: RunCommandResult,
}

export interface Deal {
  Concurrency?: number,
  Confidence?: number,
  MinBids?: number,
}

export interface Spec {
  Engine: string,
  Verifier: string,
  Publisher: string,
  Docker: JobSpecDocker,
  Language: JobSpecLanguage,
  Wasm: JobSpecWasm,
  Resources: ResourceUsageConfig,
  inputs: StorageSpec[],
  Contexts: StorageSpec[],
  outputs: StorageSpec[],
  Annotations: string[],
  Sharding: JobShardingConfig,
  DoNotTrack: boolean,
  Deal: Deal,
  ExecutionPlan: JobExecutionPlan,
}

export interface JobSpecDocker {
  Image?: string,
  Entrypoint?: string[],
  EnvironmentVariables?: string[],
  WorkingDirectory?: string,
}

export interface JobSpecLanguage {
  Language?: string,
  LanguageVersion?: string,
  DeterministicExecution?: boolean,
  JobContext?: StorageSpec,
  Command?: string,
  ProgramPath?: string,
  RequirementsPath?: string,
}

export interface JobSpecWasm {
  EntryPoint?: string,
  Parameters?: string[],
}

export interface JobLocalEvent {
  EventName?: string,
  JobID?: string,
  ShardIndex?: number,
  TargetNodeID?: string,
}

export interface JobEvent {
  APIVersion?: string,
  JobID: string,
  ShardIndex: number,
  ClientID: string,
  SourceNodeID: string,
  TargetNodeID: string,
  EventName: string,
  Spec?: Spec,
  JobExecutionPlan?: JobExecutionPlan,
  Deal?: Deal,
  Status?: string,
  VerificationProposal?: string,
  VerificationResult?: VerificationResult,
  PublishedResult?: StorageSpec,
  EventTime: string,
  SenderPublicKey?: string,
  RunOutput?: RunCommandResult,
}

export interface VerificationResult {
  Complete?: boolean,
  Result?: boolean,
}

export interface JobCreatePayload {
  ClientID?: string,
  Job?: Job,
  Context?: string,
}

export interface JobInfo {
  job: Job,
  events: JobEvent[],
  results: PublishedResult[],
  state: JobState,
  requests: JobModerationRequest[],
  moderations: JobModerationSummary[],
}

export interface JobRelation {
  JobID: string,
  CID: string,
}

export interface JobIO {
  JobID: string,
  InputOutput: string,
  IsInput: boolean,
}

export interface ResourceUsageData {
  CPU?: number,
  Memory?: number,
  Disk?: number,
  GPU?: number,
}

export interface PeerInfo {
  ID: string,
  Addrs: string[],
}

export interface ComputeNodeInfo {
  ExecutionEngines: string[],
  MaxCapacity: ResourceUsageData,
  AvailableCapacity: ResourceUsageData,
  MaxJobRequirements: ResourceUsageData,
  RunningExecutions: number,
  EnqueuedExecutions: number,
}

export interface NodeInfo {
  PeerInfo: PeerInfo,
  NodeType: string,
  ComputeNodeInfo: ComputeNodeInfo,
}

export interface Counter {
  count: number,
}

export interface AnnotationSummary {
  annotation: string,
  count: number,
}

export interface JobMonthSummary {
  month: string,
  count: number,
}

export interface JobExecutorSummary {
  executor: string,
  count: number,
}

export interface DashboardSummary {
  annotations: AnnotationSummary[],
  jobMonths: JobMonthSummary[],
  jobExecutors: JobExecutorSummary[],
  totalJobs: Counter,
  totalEvents: Counter,
  totalUsers: Counter,
  totalExecutors: Counter,
}

export interface TokenResponse {
  token: string,
}

export interface User {
  id: number,
  username: string,
}

export interface JobModeration {
  id: number,
  request_id: number,
  user_account_id: number,
  created: string,
  status: boolean,
  notes: string,
}

export enum ModerationType {
  Datacap = "datacap",
  Execution = "execution",
}

export interface JobModerationRequest {
  id: number,
  job_id: string,
  type: ModerationType,
  created: string,
  callback: string,
}

export interface ModerateRequest {
  approved: boolean,
  reason: string,
}

export interface JobModerationSummary {
  moderation: JobModeration,
  request: JobModerationRequest,
  user: User,
}
