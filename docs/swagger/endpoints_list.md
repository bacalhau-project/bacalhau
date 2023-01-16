Returns the first (sorted) #`max_jobs` jobs that belong to the `client_id` passed in the body payload (by default).
If `return_all` is set to true, it returns all jobs on the Bacalhau network.

If `id` is set, it returns only the job with that ID.

Example response:
```json
{
  "jobs": [
    {
      "APIVersion": "V1beta1",
      "ID": "9304c616-291f-41ad-b862-54e133c0149e",
      "RequesterNodeID": "QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF",
      "RequesterPublicKey": "...",
      "ClientID": "ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51",
      "Spec": {
        "Engine": "Docker",
        "Verifier": "Noop",
        "Publisher": "Estuary",
        "Docker": {
          "Image": "ubuntu",
          "Entrypoint": [
            "date"
          ]
        },
        "Language": {
          "JobContext": {}
        },
        "Wasm": {},
        "Resources": {
          "gpu": ""
        },
        "Timeout": 1800,
        "outputs": [
          {
            "StorageSource": "IPFS",
            "Name": "outputs",
            "path": "/outputs"
          }
        ],
        "Sharding": {
          "BatchSize": 1,
          "GlobPatternBasePath": "/inputs"
        }
      },
      "Deal": {
        "Concurrency": 1
      },
      "ExecutionPlan": {
        "ShardsTotal": 1
      },
      "CreatedAt": "2022-11-17T13:32:55.33837275Z",
      "JobState": {
        "Nodes": {
          "QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86": {
            "Shards": {
              "0": {
                "NodeId": "QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86",
                "State": "Cancelled",
                "VerificationResult": {},
                "PublishedResults": {}
              }
            }
          },
          "QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3": {
            "Shards": {
              "0": {
                "NodeId": "QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3",
                "State": "Cancelled",
                "VerificationResult": {},
                "PublishedResults": {}
              }
            }
          },
          "QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL": {
            "Shards": {
              "0": {
                "NodeId": "QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
                "State": "Completed",
                "Status": "Got results proposal of length: 0",
                "VerificationResult": {
                  "Complete": true,
                  "Result": true
                },
                "PublishedResults": {
                  "StorageSource": "IPFS",
                  "Name": "job-9304c616-291f-41ad-b862-54e133c0149e-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
                  "cid": "QmTVmC7JBD2ES2qGPqBNVWnX1KeEPNrPGb7rJ8cpFgtefe"
                },
                "RunOutput": {
                  "stdout": "Thu Nov 17 13:32:55 UTC 2022\n",
                  "stdouttruncated": false,
                  "stderr": "",
                  "stderrtruncated": false,
                  "exitCode": 0,
                  "runnerError": ""
                }
              }
            }
          }
        }
      }
    },
    {
      "APIVersion": "V1beta1",
      "ID": "92d5d4ee-3765-4f78-8353-623f5f26df08",
      "RequesterNodeID": "QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF",
      "RequesterPublicKey": "...",
      "ClientID": "ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51",
      "Spec": {
        "Engine": "Docker",
        "Verifier": "Noop",
        "Publisher": "Estuary",
        "Docker": {
          "Image": "ubuntu",
          "Entrypoint": [
            "sleep",
            "4"
          ]
        },
        "Language": {
          "JobContext": {}
        },
        "Wasm": {},
        "Resources": {
          "gpu": ""
        },
        "Timeout": 1800,
        "outputs": [
          {
            "StorageSource": "IPFS",
            "Name": "outputs",
            "path": "/outputs"
          }
        ],
        "Sharding": {
          "BatchSize": 1,
          "GlobPatternBasePath": "/inputs"
        }
      },
      "Deal": {
        "Concurrency": 1
      },
      "ExecutionPlan": {
        "ShardsTotal": 1
      },
      "CreatedAt": "2022-11-17T13:29:01.871140291Z",
      "JobState": {
        "Nodes": {
          "QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86": {
            "Shards": {
              "0": {
                "NodeId": "QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86",
                "State": "Cancelled",
                "VerificationResult": {},
                "PublishedResults": {}
              }
            }
          },
          "QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3": {
            "Shards": {
              "0": {
                "NodeId": "QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3",
                "State": "Completed",
                "Status": "Got results proposal of length: 0",
                "VerificationResult": {
                  "Complete": true,
                  "Result": true
                },
                "PublishedResults": {
                  "StorageSource": "IPFS",
                  "Name": "job-92d5d4ee-3765-4f78-8353-623f5f26df08-shard-0-host-QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3",
                  "cid": "QmWUXBndMuq2G6B6ndQCmkRHjZ6CvyJ8qLxXBG3YsSFzQG"
                },
                "RunOutput": {
                  "stdout": "",
                  "stdouttruncated": false,
                  "stderr": "",
                  "stderrtruncated": false,
                  "exitCode": 0,
                  "runnerError": ""
                }
              }
            }
          }
        }
      }
    }
  ]
}
```