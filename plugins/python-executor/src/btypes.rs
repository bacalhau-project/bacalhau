#![allow(dead_code)]
use serde::{Deserialize, Serialize};

// temporary type definitions for interop with the compute node.
// The protobuf currently has each request/response object carry
// only []byte and leaves it up to us to encode/decode the JSON.

#[derive(Deserialize, Debug)]
pub struct OutputLimits {
    #[serde(alias = "MaxStdoutFileLength")]
    max_stdout_file_length: u64,

    #[serde(alias = "MaxStdoutReturnLength")]
    max_stdout_return_length: u64,

    #[serde(alias = "MaxStderrFileLength")]
    max_stderr_file_length: u64,

    #[serde(alias = "MaxStderrReturnLength")]
    max_stderr_return_length: u64,
}

#[derive(Deserialize, Debug)]
pub struct StorageVolume {
    #[serde(alias = "type")]
    r#type: i32,

    #[serde(alias = "ReadOnly")]
    read_only: bool,

    #[serde(alias = "Source")]
    source: String,

    #[serde(alias = "Target")]
    target: String,
}

#[derive(Deserialize, Debug)]
pub struct SpecConfig {
    #[serde(alias = "Type")]
    r#type: String,
    // params: map[string]interface{}
}

#[derive(Deserialize, Debug)]
pub struct InputSource {
    #[serde(alias = "Source")]
    source: SpecConfig,

    #[serde(alias = "Alias")]
    alias: String,

    #[serde(alias = "Target")]
    target: String,
}

#[derive(Deserialize, Debug)]
pub struct PreparedStorage {
    #[serde(alias = "InputSource")]
    input_source: InputSource,

    #[serde(alias = "Volume")]
    volume: StorageVolume,
}

#[derive(Deserialize, Debug)]
pub struct ResultPath {
    #[serde(alias = "Name")]
    name: String,

    #[serde(alias = "Path")]
    path: String,
}

#[derive(Deserialize, Debug)]
pub struct Resources {
    #[serde(alias = "CPU")]
    cpu: f64,

    #[serde(alias = "Memory")]
    memory: u64,

    #[serde(alias = "Disk")]
    disk: u64,

    #[serde(alias = "GPU")]
    gpu: u64,
}

#[derive(Deserialize, Debug)]
pub struct NetworkConfig {
    #[serde(alias = "Type")]
    r#type: i32,

    #[serde(alias = "Domains")]
    domains: Vec<String>,
}

#[derive(Deserialize, Debug)]
pub struct RunCommandRequest {
    #[serde(alias = "JobID")]
    pub job_id: Option<String>,

    #[serde(alias = "ExecutionID")]
    pub execution_id: String,

    #[serde(alias = "Resources")]
    pub resources: Option<Resources>,

    #[serde(alias = "Network")]
    pub network: Option<NetworkConfig>,

    #[serde(alias = "Outputs")]
    pub outputs: Option<Vec<ResultPath>>,

    #[serde(alias = "Inputs")]
    pub inputs: Option<Vec<PreparedStorage>>,

    #[serde(alias = "ResultsDir")]
    pub results_dir: Option<String>,

    #[serde(alias = "EngineParams")]
    pub engine_params: Option<SpecConfig>,

    #[serde(alias = "OutputLimits")]
    pub output_limits: Option<OutputLimits>,
}

#[derive(Serialize, Debug, Default)]
pub struct RunCommandResponse {
    #[serde(alias = "Stdout")]
    pub stdout: String,

    #[serde(alias = "StdoutTruncated")]
    pub stdout_truncated: bool,

    #[serde(alias = "Stderr")]
    pub stderr: String,

    #[serde(alias = "StderrTruncated")]
    pub stderr_truncated: bool,

    #[serde(alias = "ExitCode")]
    pub exit_code: i32,

    #[serde(alias = "ErrorMsg")]
    pub error_msg: Option<String>,
}

impl RunCommandResponse {
    pub fn new() -> Self {
        Self {
            ..Default::default()
        }
    }

    pub fn with_exit_code(mut self, code: i32) -> Self {
        self.exit_code = code;
        self
    }

    pub fn with_error(mut self, err: String) -> Self {
        self.error_msg = Some(err);
        self
    }
}
