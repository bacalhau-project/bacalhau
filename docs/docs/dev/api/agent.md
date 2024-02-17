---
sidebar_label: Agent
---

# Agent API Documentation

The Bacalhau Agent APIs provide a convenient means to retrieve information about the Bacalhau node you are communicating with, whether it serves as the orchestrator or functions as a compute node. These APIs offer valuable insights into the node's health, capabilities, and deployed Bacalhau version.


## Is Alive

**Endpoint:** `GET /api/v1/agent/alive`

This API can be used to determine if the agent is operational and responding as expected.

**Response**:
```json
{
  "Status": "OK"
}
```


## Deployed Bacalhau Version

**Endpoint:** `GET /api/v1/agent/version`

This API provides details about the Bacalhau version, including major and minor version numbers, Git version, Git commit, build date, and platform information.

**Response**:
```json
{
  "Major": "1",
  "Minor": "1",
  "GitVersion": "v1.1.0",
  "GitCommit": "970e1a0f23c7eb739a097aa8212f7964434bcd97",
  "BuildDate": "2023-09-25T07:59:00Z",
  "GOOS": "linux",
  "GOARCH": "amd64"
}
```


## Node Info

**Endpoint:** `GET /api/v1/agent/node`

This API provides detailed information about the node, including its peer ID and network addresses, node type (e.g., Compute), labels, compute node capabilities, and the deployed Bacalhau version.

**Response**:
```json
{
  "PeerInfo": {
    "ID": "QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
    "Addrs": [
      "/ip4/35.245.245.245/tcp/1235"
    ]
  },
  "NodeType": "Compute",
  "Labels": {
    "Architecture": "amd64",
    "Operating-System": "linux",
    "git-lfs": "True",
    "owner": "bacalhau"
  },
  "ComputeNodeInfo": {
    "ExecutionEngines": [
      "docker",
      "wasm"
    ],
    "Publishers": [
      "noop",
      "ipfs",
      "s3"
    ],
    "StorageSources": [
      "repoclonelfs",
      "s3",
      "ipfs",
      "urldownload",
      "inline",
      "repoclone"
    ],
    "MaxCapacity": {
      "CPU": 12.8,
      "Memory": 53931121049,
      "Disk": 721417073459,
      "GPU": 0
    },
    "AvailableCapacity": {
      "CPU": 12.8,
      "Memory": 53931121049,
      "Disk": 721417073459,
      "GPU": 0
    },
    "MaxJobRequirements": {
      "CPU": 12.8,
      "Memory": 53931121049,
      "Disk": 721417073459,
      "GPU": 0
    },
    "RunningExecutions": 0,
    "EnqueuedExecutions": 0
  },
  "BacalhauVersion": {
    "Major": "1",
    "Minor": "1",
    "GitVersion": "v1.1.0",
    "GitCommit": "970e1a0f23c7eb739a097aa8212f7964434bcd97",
    "BuildDate": "2023-09-25T07:59:00Z",
    "GOOS": "linux",
    "GOARCH": "amd64"
  }
}
```
