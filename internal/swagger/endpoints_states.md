Example response:

```json
{
  "state": {
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
              "CID": "QmTVmC7JBD2ES2qGPqBNVWnX1KeEPNrPGb7rJ8cpFgtefe"
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
}
```
