Events (e.g. Created, Bid, BidAccepted, ..., ResultsAccepted, ResultsPublished) are useful to track the progress of a job.

Example response (truncated):
```json
{
  "events": [
    {
      "APIVersion": "V1beta1",
      "JobID": "9304c616-291f-41ad-b862-54e133c0149e",
      "ClientID": "ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51",
      "SourceNodeID": "QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF",
      "EventName": "Created",
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
      "JobExecutionPlan": {
        "ShardsTotal": 1
      },
      "Deal": {
        "Concurrency": 1
      },
      "VerificationResult": {},
      "PublishedResult": {},
      "EventTime": "2022-11-17T13:32:55.331375351Z",
      "SenderPublicKey": "..."
    },
    ...
    {
      "JobID": "9304c616-291f-41ad-b862-54e133c0149e",
      "SourceNodeID": "QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF",
      "TargetNodeID": "QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
      "EventName": "ResultsAccepted",
      "Spec": {
        "Docker": {},
        "Language": {
          "JobContext": {}
        },
        "Wasm": {},
        "Resources": {
          "gpu": ""
        },
        "Sharding": {}
      },
      "JobExecutionPlan": {},
      "Deal": {},
      "VerificationResult": {
        "Complete": true,
        "Result": true
      },
      "PublishedResult": {},
      "EventTime": "2022-11-17T13:32:55.707825569Z",
      "SenderPublicKey": "..."
    },
    {
      "JobID": "9304c616-291f-41ad-b862-54e133c0149e",
      "SourceNodeID": "QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
      "EventName": "ResultsPublished",
      "Spec": {
        "Docker": {},
        "Language": {
          "JobContext": {}
        },
        "Wasm": {},
        "Resources": {
          "gpu": ""
        },
        "Sharding": {}
      },
      "JobExecutionPlan": {},
      "Deal": {},
      "VerificationResult": {},
      "PublishedResult": {
        "StorageSource": "IPFS",
        "Name": "job-9304c616-291f-41ad-b862-54e133c0149e-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
        "cid": "QmTVmC7JBD2ES2qGPqBNVWnX1KeEPNrPGb7rJ8cpFgtefe"
      },
      "EventTime": "2022-11-17T13:32:55.756658941Z",
      "SenderPublicKey": "..."
    }
  ]
}
```