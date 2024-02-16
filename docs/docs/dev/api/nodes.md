---
sidebar_label: Nodes
---

# Nodes API Documentation

Nodes API provides a way to query information about the nodes in the cluster.

## Describe Node

**Endpoint:** `GET /api/v1/orchestrator/nodes/:nodeID`

Retrieve information about a specific node.

**Parameters**:
  - `:nodeID`: Identifier of the node to describe. (e.g. `QmUDAXvv31WPZ8U9CzuRTMn9iFGiopGE7rHiah1X8a6PkT`)

**Response**:
- **Node**: Detailed information about the requested node.

**Example**:
```bash
curl 127.0.0.1:1234/api/v1/orchestrator/nodes/QmUDAXvv31WPZ8U9CzuRTMn9iFGiopGE7rHiah1X8a6PkT
{
  "Node": {
    "PeerInfo": {
      "ID": "QmUDAXvv31WPZ8U9CzuRTMn9iFGiopGE7rHiah1X8a6PkT",
      "Addrs": [
        "/ip4/34.34.247.247/tcp/1235"
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
        "s3",
        "noop",
        "ipfs"
      ],
      "StorageSources": [
        "urldownload",
        "inline",
        "repoclone",
        "repoclonelfs",
        "s3",
        "ipfs"
      ],
      "MaxCapacity": {
        "CPU": 3.2,
        "Memory": 12561049190,
        "Disk": 582010404864,
        "GPU": 1
      },
      "AvailableCapacity": {
        "CPU": 3.2,
        "Memory": 12561049190,
        "Disk": 582010404864,
        "GPU": 1
      },
      "MaxJobRequirements": {
        "CPU": 3.2,
        "Memory": 12561049190,
        "Disk": 582010404864,
        "GPU": 1
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
}
```

## List Nodes

**Endpoint:** `GET /api/v1/orchestrator/nodes`

Retrieve a list of nodes.

**Parameters**:
  - `labels`: Use label-based criteria to filter nodes. See [Label Filtering](../api) for usage details.
  - `limit`: Set the maximum number of jobs to return. Default is set to 10.
  - `next_token`: Utilize this parameter for pagination continuation.
  - `order_by`: Determine the ordering of jobs. Choose between `id`, `type`, `available_cpu`, `available_memory`, `available_disk` or `available_gpu`. (default is `id`).
  - `reverse`: Opt to reverse the default order of displayed jobs.

**Response**:
- **Nodes**: List of matching nodes.
- **NextToken** `(string)`: Pagination token.

**Example**:

Find two linux nodes with most available Memory
```bash
curl --get  "127.0.0.1:1234/api/v1/orchestrator/nodes?limit=2&order_by=available_memory" --data-urlencode 'labels=Operating-System=linux'
{
  "NextToken": "",
  "Nodes": [
    {
      "PeerInfo": {
        "ID": "QmcC3xifiiCuGGQ9rpvefUoary9tY65x2HaNxSdeMTvM9U",
        "Addrs": [
          "/ip4/212.248.248.248/tcp/1235"
        ]
      },
      "NodeType": "Compute",
      "Labels": {
        "Architecture": "amd64",
        "Operating-System": "linux",
        "env": "prod",
        "git-lfs": "False",
        "name": "saturnia_len20"
      },
      "ComputeNodeInfo": {
        "ExecutionEngines": [
          "wasm",
          "docker"
        ],
        "Publishers": [
          "noop",
          "ipfs"
        ],
        "StorageSources": [
          "urldownload",
          "inline",
          "ipfs"
        ],
        "MaxCapacity": {
          "CPU": 102,
          "Memory": 858993459200,
          "Disk": 562967789568,
          "GPU": 2
        },
        "AvailableCapacity": {
          "CPU": 102,
          "Memory": 858993459200,
          "Disk": 562967789568,
          "GPU": 2
        },
        "MaxJobRequirements": {
          "CPU": 96,
          "Memory": 858993459200,
          "Disk": 562967789568,
          "GPU": 2
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
    },
    {
      "PeerInfo": {
        "ID": "QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF",
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
          "s3",
          "ipfs",
          "urldownload",
          "inline",
          "repoclone",
          "repoclonelfs"
        ],
        "MaxCapacity": {
          "CPU": 12.8,
          "Memory": 53931124326,
          "Disk": 718749414195,
          "GPU": 0
        },
        "AvailableCapacity": {
          "CPU": 12.8,
          "Memory": 53931124326,
          "Disk": 718749414195,
          "GPU": 0
        },
        "MaxJobRequirements": {
          "CPU": 12.8,
          "Memory": 53931124326,
          "Disk": 718749414195,
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
  ]
}
```
