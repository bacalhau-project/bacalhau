'use strict';


/**
 * Submits a new job to the network.
 * Description:  * `client_public_key`: The base64-encoded public key of the client. * `signature`: A base64-encoded signature of the `data` attribute, signed by the client. * `data`     * `ClientID`: Request must specify a `ClientID`. To retrieve your `ClientID`, you can do the following: (1) submit a dummy job to Bacalhau (or use one you created before), (2) run `bacalhau describe <job-id>` and fetch the `ClientID` field.     * `Job`: see example below.  Example request ```json {  \"data\": {   \"ClientID\": \"ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51\",   \"Job\": {    \"APIVersion\": \"V1beta1\",    \"Spec\": {     \"Engine\": \"Docker\",     \"Verifier\": \"Noop\",     \"Publisher\": \"Estuary\",     \"Docker\": {      \"Image\": \"ubuntu\",      \"Entrypoint\": [       \"date\"      ]     },     \"Timeout\": 1800,     \"outputs\": [      {       \"StorageSource\": \"IPFS\",       \"Name\": \"outputs\",       \"path\": \"/outputs\"      }     ],     \"Sharding\": {      \"BatchSize\": 1,      \"GlobPatternBasePath\": \"/inputs\"     }    },    \"Deal\": {     \"Concurrency\": 1    }   }  },  \"signature\": \"...\",  \"client_public_key\": \"...\" } ```
 *
 * body Publicapi.submitRequest 
 * returns publicapi.submitResponse
 **/
exports.pkg/apiServer.submit = function(body) {
  return new Promise(function(resolve, reject) {
    var examples = {};
    examples['application/json'] = {
  "job" : {
    "RequesterNodeID" : "QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF",
    "ExecutionPlan" : {
      "ShardsTotal" : 5
    },
    "LocalJobEvents" : [ {
      "TargetNodeID" : "TargetNodeID",
      "ShardIndex" : 5,
      "EventName" : 5,
      "JobID" : "JobID"
    }, {
      "TargetNodeID" : "TargetNodeID",
      "ShardIndex" : 5,
      "EventName" : 5,
      "JobID" : "JobID"
    } ],
    "APIVersion" : "V1beta1",
    "CreatedAt" : "2022-11-17T13:29:01.871140291Z",
    "ClientID" : "ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51",
    "JobState" : {
      "Nodes" : {
        "key" : {
          "Shards" : {
            "key" : {
              "Status" : "Status",
              "RunOutput" : {
                "stderrtruncated" : true,
                "stdout" : "stdout",
                "exitCode" : 7,
                "runnerError" : "runnerError",
                "stdouttruncated" : true,
                "stderr" : "stderr"
              },
              "VerificationProposal" : [ 1, 1 ],
              "VerificationResult" : {
                "Complete" : true,
                "Result" : true
              },
              "ShardIndex" : 0,
              "State" : 6,
              "NodeId" : "NodeId",
              "PublishedResults" : {
                "path" : "path",
                "Metadata" : {
                  "key" : "Metadata"
                },
                "URL" : "URL",
                "CID" : "QmTVmC7JBD2ES2qGPqBNVWnX1KeEPNrPGb7rJ8cpFgtefe",
                "Name" : "job-9304c616-291f-41ad-b862-54e133c0149e-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
                "StorageSource" : 2
              }
            }
          }
        }
      }
    },
    "RequesterPublicKey" : [ 2, 2 ],
    "ID" : "92d5d4ee-3765-4f78-8353-623f5f26df08",
    "Deal" : {
      "MinBids" : 1,
      "Concurrency" : 0,
      "Confidence" : 6
    },
    "JobEvents" : [ {
      "Status" : "Got results proposal of length: 0",
      "JobExecutionPlan" : {
        "ShardsTotal" : 5
      },
      "SourceNodeID" : "QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF",
      "ShardIndex" : 3,
      "EventName" : 5,
      "Deal" : {
        "MinBids" : 1,
        "Concurrency" : 0,
        "Confidence" : 6
      },
      "PublishedResult" : {
        "path" : "path",
        "Metadata" : {
          "key" : "Metadata"
        },
        "URL" : "URL",
        "CID" : "QmTVmC7JBD2ES2qGPqBNVWnX1KeEPNrPGb7rJ8cpFgtefe",
        "Name" : "job-9304c616-291f-41ad-b862-54e133c0149e-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
        "StorageSource" : 2
      },
      "TargetNodeID" : "QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
      "RunOutput" : {
        "stderrtruncated" : true,
        "stdout" : "stdout",
        "exitCode" : 7,
        "runnerError" : "runnerError",
        "stdouttruncated" : true,
        "stderr" : "stderr"
      },
      "VerificationProposal" : [ 1, 1 ],
      "VerificationResult" : {
        "Complete" : true,
        "Result" : true
      },
      "APIVersion" : "V1beta1",
      "SenderPublicKey" : [ 9, 9 ],
      "EventTime" : "2022-11-17T13:32:55.756658941Z",
      "ClientID" : "ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51",
      "Spec" : {
        "outputs" : [ null, null ],
        "Sharding" : {
          "BatchSize" : 7,
          "GlobPattern" : "GlobPattern",
          "GlobPatternBasePath" : "GlobPatternBasePath"
        },
        "Timeout" : 1.2315135367772556,
        "inputs" : [ null, null ],
        "DoNotTrack" : true,
        "Publisher" : 4,
        "Verifier" : 1,
        "Contexts" : [ null, null ],
        "Wasm" : {
          "EnvironmentVariables" : {
            "key" : "EnvironmentVariables"
          },
          "Parameters" : [ "Parameters", "Parameters" ],
          "ImportModules" : [ null, null ],
          "EntryPoint" : "EntryPoint"
        },
        "Annotations" : [ "Annotations", "Annotations" ],
        "Docker" : {
          "WorkingDirectory" : "WorkingDirectory",
          "EnvironmentVariables" : [ "EnvironmentVariables", "EnvironmentVariables" ],
          "Entrypoint" : [ "Entrypoint", "Entrypoint" ],
          "Image" : "Image"
        },
        "Language" : {
          "RequirementsPath" : "RequirementsPath",
          "Language" : "Language",
          "Command" : "Command",
          "DeterministicExecution" : true,
          "LanguageVersion" : "LanguageVersion",
          "ProgramPath" : "ProgramPath"
        },
        "Resources" : {
          "Memory" : "Memory",
          "CPU" : "CPU",
          "Disk" : "Disk",
          "GPU" : "GPU"
        },
        "Engine" : 2
      },
      "JobID" : "9304c616-291f-41ad-b862-54e133c0149e"
    }, {
      "Status" : "Got results proposal of length: 0",
      "JobExecutionPlan" : {
        "ShardsTotal" : 5
      },
      "SourceNodeID" : "QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF",
      "ShardIndex" : 3,
      "EventName" : 5,
      "Deal" : {
        "MinBids" : 1,
        "Concurrency" : 0,
        "Confidence" : 6
      },
      "PublishedResult" : {
        "path" : "path",
        "Metadata" : {
          "key" : "Metadata"
        },
        "URL" : "URL",
        "CID" : "QmTVmC7JBD2ES2qGPqBNVWnX1KeEPNrPGb7rJ8cpFgtefe",
        "Name" : "job-9304c616-291f-41ad-b862-54e133c0149e-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
        "StorageSource" : 2
      },
      "TargetNodeID" : "QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
      "RunOutput" : {
        "stderrtruncated" : true,
        "stdout" : "stdout",
        "exitCode" : 7,
        "runnerError" : "runnerError",
        "stdouttruncated" : true,
        "stderr" : "stderr"
      },
      "VerificationProposal" : [ 1, 1 ],
      "VerificationResult" : {
        "Complete" : true,
        "Result" : true
      },
      "APIVersion" : "V1beta1",
      "SenderPublicKey" : [ 9, 9 ],
      "EventTime" : "2022-11-17T13:32:55.756658941Z",
      "ClientID" : "ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51",
      "Spec" : {
        "outputs" : [ null, null ],
        "Sharding" : {
          "BatchSize" : 7,
          "GlobPattern" : "GlobPattern",
          "GlobPatternBasePath" : "GlobPatternBasePath"
        },
        "Timeout" : 1.2315135367772556,
        "inputs" : [ null, null ],
        "DoNotTrack" : true,
        "Publisher" : 4,
        "Verifier" : 1,
        "Contexts" : [ null, null ],
        "Wasm" : {
          "EnvironmentVariables" : {
            "key" : "EnvironmentVariables"
          },
          "Parameters" : [ "Parameters", "Parameters" ],
          "ImportModules" : [ null, null ],
          "EntryPoint" : "EntryPoint"
        },
        "Annotations" : [ "Annotations", "Annotations" ],
        "Docker" : {
          "WorkingDirectory" : "WorkingDirectory",
          "EnvironmentVariables" : [ "EnvironmentVariables", "EnvironmentVariables" ],
          "Entrypoint" : [ "Entrypoint", "Entrypoint" ],
          "Image" : "Image"
        },
        "Language" : {
          "RequirementsPath" : "RequirementsPath",
          "Language" : "Language",
          "Command" : "Command",
          "DeterministicExecution" : true,
          "LanguageVersion" : "LanguageVersion",
          "ProgramPath" : "ProgramPath"
        },
        "Resources" : {
          "Memory" : "Memory",
          "CPU" : "CPU",
          "Disk" : "Disk",
          "GPU" : "GPU"
        },
        "Engine" : 2
      },
      "JobID" : "9304c616-291f-41ad-b862-54e133c0149e"
    } ],
    "Spec" : {
      "outputs" : [ null, null ],
      "Sharding" : {
        "BatchSize" : 7,
        "GlobPattern" : "GlobPattern",
        "GlobPatternBasePath" : "GlobPatternBasePath"
      },
      "Timeout" : 1.2315135367772556,
      "inputs" : [ null, null ],
      "DoNotTrack" : true,
      "Publisher" : 4,
      "Verifier" : 1,
      "Contexts" : [ null, null ],
      "Wasm" : {
        "EnvironmentVariables" : {
          "key" : "EnvironmentVariables"
        },
        "Parameters" : [ "Parameters", "Parameters" ],
        "ImportModules" : [ null, null ],
        "EntryPoint" : "EntryPoint"
      },
      "Annotations" : [ "Annotations", "Annotations" ],
      "Docker" : {
        "WorkingDirectory" : "WorkingDirectory",
        "EnvironmentVariables" : [ "EnvironmentVariables", "EnvironmentVariables" ],
        "Entrypoint" : [ "Entrypoint", "Entrypoint" ],
        "Image" : "Image"
      },
      "Language" : {
        "RequirementsPath" : "RequirementsPath",
        "Language" : "Language",
        "Command" : "Command",
        "DeterministicExecution" : true,
        "LanguageVersion" : "LanguageVersion",
        "ProgramPath" : "ProgramPath"
      },
      "Resources" : {
        "Memory" : "Memory",
        "CPU" : "CPU",
        "Disk" : "Disk",
        "GPU" : "GPU"
      },
      "Engine" : 2
    }
  }
};
    if (Object.keys(examples).length > 0) {
      resolve(examples[Object.keys(examples)[0]]);
    } else {
      resolve();
    }
  });
}


/**
 * Simply lists jobs.
 * Returns the first (sorted) #`max_jobs` jobs that belong to the `client_id` passed in the body payload (by default). If `return_all` is set to true, it returns all jobs on the Bacalhau network.  If `id` is set, it returns only the job with that ID.  Example response: ```json {   \"jobs\": [     {       \"APIVersion\": \"V1beta1\",       \"ID\": \"9304c616-291f-41ad-b862-54e133c0149e\",       \"RequesterNodeID\": \"QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF\",       \"RequesterPublicKey\": \"...\",       \"ClientID\": \"ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51\",       \"Spec\": {         \"Engine\": \"Docker\",         \"Verifier\": \"Noop\",         \"Publisher\": \"Estuary\",         \"Docker\": {           \"Image\": \"ubuntu\",           \"Entrypoint\": [             \"date\"           ]         },         \"Language\": {           \"JobContext\": {}         },         \"Wasm\": {},         \"Resources\": {           \"GPU\": \"\"         },         \"Timeout\": 1800,         \"outputs\": [           {             \"StorageSource\": \"IPFS\",             \"Name\": \"outputs\",             \"path\": \"/outputs\"           }         ],         \"Sharding\": {           \"BatchSize\": 1,           \"GlobPatternBasePath\": \"/inputs\"         }       },       \"Deal\": {         \"Concurrency\": 1       },       \"ExecutionPlan\": {         \"ShardsTotal\": 1       },       \"CreatedAt\": \"2022-11-17T13:32:55.33837275Z\",       \"JobState\": {         \"Nodes\": {           \"QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86\": {             \"Shards\": {               \"0\": {                 \"NodeId\": \"QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86\",                 \"State\": \"Cancelled\",                 \"VerificationResult\": {},                 \"PublishedResults\": {}               }             }           },           \"QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3\": {             \"Shards\": {               \"0\": {                 \"NodeId\": \"QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3\",                 \"State\": \"Cancelled\",                 \"VerificationResult\": {},                 \"PublishedResults\": {}               }             }           },           \"QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\": {             \"Shards\": {               \"0\": {                 \"NodeId\": \"QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\",                 \"State\": \"Completed\",                 \"Status\": \"Got results proposal of length: 0\",                 \"VerificationResult\": {                   \"Complete\": true,                   \"Result\": true                 },                 \"PublishedResults\": {                   \"StorageSource\": \"IPFS\",                   \"Name\": \"job-9304c616-291f-41ad-b862-54e133c0149e-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\",                   \"CID\": \"QmTVmC7JBD2ES2qGPqBNVWnX1KeEPNrPGb7rJ8cpFgtefe\"                 },                 \"RunOutput\": {                   \"stdout\": \"Thu Nov 17 13:32:55 UTC 2022\\n\",                   \"stdouttruncated\": false,                   \"stderr\": \"\",                   \"stderrtruncated\": false,                   \"exitCode\": 0,                   \"runnerError\": \"\"                 }               }             }           }         }       }     },     {       \"APIVersion\": \"V1beta1\",       \"ID\": \"92d5d4ee-3765-4f78-8353-623f5f26df08\",       \"RequesterNodeID\": \"QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF\",       \"RequesterPublicKey\": \"...\",       \"ClientID\": \"ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51\",       \"Spec\": {         \"Engine\": \"Docker\",         \"Verifier\": \"Noop\",         \"Publisher\": \"Estuary\",         \"Docker\": {           \"Image\": \"ubuntu\",           \"Entrypoint\": [             \"sleep\",             \"4\"           ]         },         \"Language\": {           \"JobContext\": {}         },         \"Wasm\": {},         \"Resources\": {           \"GPU\": \"\"         },         \"Timeout\": 1800,         \"outputs\": [           {             \"StorageSource\": \"IPFS\",             \"Name\": \"outputs\",             \"path\": \"/outputs\"           }         ],         \"Sharding\": {           \"BatchSize\": 1,           \"GlobPatternBasePath\": \"/inputs\"         }       },       \"Deal\": {         \"Concurrency\": 1       },       \"ExecutionPlan\": {         \"ShardsTotal\": 1       },       \"CreatedAt\": \"2022-11-17T13:29:01.871140291Z\",       \"JobState\": {         \"Nodes\": {           \"QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86\": {             \"Shards\": {               \"0\": {                 \"NodeId\": \"QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86\",                 \"State\": \"Cancelled\",                 \"VerificationResult\": {},                 \"PublishedResults\": {}               }             }           },           \"QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3\": {             \"Shards\": {               \"0\": {                 \"NodeId\": \"QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3\",                 \"State\": \"Completed\",                 \"Status\": \"Got results proposal of length: 0\",                 \"VerificationResult\": {                   \"Complete\": true,                   \"Result\": true                 },                 \"PublishedResults\": {                   \"StorageSource\": \"IPFS\",                   \"Name\": \"job-92d5d4ee-3765-4f78-8353-623f5f26df08-shard-0-host-QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3\",                   \"CID\": \"QmWUXBndMuq2G6B6ndQCmkRHjZ6CvyJ8qLxXBG3YsSFzQG\"                 },                 \"RunOutput\": {                   \"stdout\": \"\",                   \"stdouttruncated\": false,                   \"stderr\": \"\",                   \"stderrtruncated\": false,                   \"exitCode\": 0,                   \"runnerError\": \"\"                 }               }             }           }         }       }     }   ] } ```
 *
 * body Publicapi.listRequest Set `return_all` to `true` to return all jobs on the network (may degrade performance, use with care!).
 * returns publicapi.listResponse
 **/
exports.pkg/publicapi.list = function(body) {
  return new Promise(function(resolve, reject) {
    var examples = {};
    examples['application/json'] = {
  "jobs" : [ {
    "RequesterNodeID" : "QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF",
    "ExecutionPlan" : {
      "ShardsTotal" : 5
    },
    "LocalJobEvents" : [ {
      "TargetNodeID" : "TargetNodeID",
      "ShardIndex" : 5,
      "EventName" : 5,
      "JobID" : "JobID"
    }, {
      "TargetNodeID" : "TargetNodeID",
      "ShardIndex" : 5,
      "EventName" : 5,
      "JobID" : "JobID"
    } ],
    "APIVersion" : "V1beta1",
    "CreatedAt" : "2022-11-17T13:29:01.871140291Z",
    "ClientID" : "ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51",
    "JobState" : {
      "Nodes" : {
        "key" : {
          "Shards" : {
            "key" : {
              "Status" : "Status",
              "RunOutput" : {
                "stderrtruncated" : true,
                "stdout" : "stdout",
                "exitCode" : 7,
                "runnerError" : "runnerError",
                "stdouttruncated" : true,
                "stderr" : "stderr"
              },
              "VerificationProposal" : [ 1, 1 ],
              "VerificationResult" : {
                "Complete" : true,
                "Result" : true
              },
              "ShardIndex" : 0,
              "State" : 6,
              "NodeId" : "NodeId",
              "PublishedResults" : {
                "path" : "path",
                "Metadata" : {
                  "key" : "Metadata"
                },
                "URL" : "URL",
                "CID" : "QmTVmC7JBD2ES2qGPqBNVWnX1KeEPNrPGb7rJ8cpFgtefe",
                "Name" : "job-9304c616-291f-41ad-b862-54e133c0149e-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
                "StorageSource" : 2
              }
            }
          }
        }
      }
    },
    "RequesterPublicKey" : [ 2, 2 ],
    "ID" : "92d5d4ee-3765-4f78-8353-623f5f26df08",
    "Deal" : {
      "MinBids" : 1,
      "Concurrency" : 0,
      "Confidence" : 6
    },
    "JobEvents" : [ {
      "Status" : "Got results proposal of length: 0",
      "JobExecutionPlan" : {
        "ShardsTotal" : 5
      },
      "SourceNodeID" : "QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF",
      "ShardIndex" : 3,
      "EventName" : 5,
      "Deal" : {
        "MinBids" : 1,
        "Concurrency" : 0,
        "Confidence" : 6
      },
      "PublishedResult" : {
        "path" : "path",
        "Metadata" : {
          "key" : "Metadata"
        },
        "URL" : "URL",
        "CID" : "QmTVmC7JBD2ES2qGPqBNVWnX1KeEPNrPGb7rJ8cpFgtefe",
        "Name" : "job-9304c616-291f-41ad-b862-54e133c0149e-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
        "StorageSource" : 2
      },
      "TargetNodeID" : "QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
      "RunOutput" : {
        "stderrtruncated" : true,
        "stdout" : "stdout",
        "exitCode" : 7,
        "runnerError" : "runnerError",
        "stdouttruncated" : true,
        "stderr" : "stderr"
      },
      "VerificationProposal" : [ 1, 1 ],
      "VerificationResult" : {
        "Complete" : true,
        "Result" : true
      },
      "APIVersion" : "V1beta1",
      "SenderPublicKey" : [ 9, 9 ],
      "EventTime" : "2022-11-17T13:32:55.756658941Z",
      "ClientID" : "ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51",
      "Spec" : {
        "outputs" : [ null, null ],
        "Sharding" : {
          "BatchSize" : 7,
          "GlobPattern" : "GlobPattern",
          "GlobPatternBasePath" : "GlobPatternBasePath"
        },
        "Timeout" : 1.2315135367772556,
        "inputs" : [ null, null ],
        "DoNotTrack" : true,
        "Publisher" : 4,
        "Verifier" : 1,
        "Contexts" : [ null, null ],
        "Wasm" : {
          "EnvironmentVariables" : {
            "key" : "EnvironmentVariables"
          },
          "Parameters" : [ "Parameters", "Parameters" ],
          "ImportModules" : [ null, null ],
          "EntryPoint" : "EntryPoint"
        },
        "Annotations" : [ "Annotations", "Annotations" ],
        "Docker" : {
          "WorkingDirectory" : "WorkingDirectory",
          "EnvironmentVariables" : [ "EnvironmentVariables", "EnvironmentVariables" ],
          "Entrypoint" : [ "Entrypoint", "Entrypoint" ],
          "Image" : "Image"
        },
        "Language" : {
          "RequirementsPath" : "RequirementsPath",
          "Language" : "Language",
          "Command" : "Command",
          "DeterministicExecution" : true,
          "LanguageVersion" : "LanguageVersion",
          "ProgramPath" : "ProgramPath"
        },
        "Resources" : {
          "Memory" : "Memory",
          "CPU" : "CPU",
          "Disk" : "Disk",
          "GPU" : "GPU"
        },
        "Engine" : 2
      },
      "JobID" : "9304c616-291f-41ad-b862-54e133c0149e"
    }, {
      "Status" : "Got results proposal of length: 0",
      "JobExecutionPlan" : {
        "ShardsTotal" : 5
      },
      "SourceNodeID" : "QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF",
      "ShardIndex" : 3,
      "EventName" : 5,
      "Deal" : {
        "MinBids" : 1,
        "Concurrency" : 0,
        "Confidence" : 6
      },
      "PublishedResult" : {
        "path" : "path",
        "Metadata" : {
          "key" : "Metadata"
        },
        "URL" : "URL",
        "CID" : "QmTVmC7JBD2ES2qGPqBNVWnX1KeEPNrPGb7rJ8cpFgtefe",
        "Name" : "job-9304c616-291f-41ad-b862-54e133c0149e-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
        "StorageSource" : 2
      },
      "TargetNodeID" : "QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
      "RunOutput" : {
        "stderrtruncated" : true,
        "stdout" : "stdout",
        "exitCode" : 7,
        "runnerError" : "runnerError",
        "stdouttruncated" : true,
        "stderr" : "stderr"
      },
      "VerificationProposal" : [ 1, 1 ],
      "VerificationResult" : {
        "Complete" : true,
        "Result" : true
      },
      "APIVersion" : "V1beta1",
      "SenderPublicKey" : [ 9, 9 ],
      "EventTime" : "2022-11-17T13:32:55.756658941Z",
      "ClientID" : "ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51",
      "Spec" : {
        "outputs" : [ null, null ],
        "Sharding" : {
          "BatchSize" : 7,
          "GlobPattern" : "GlobPattern",
          "GlobPatternBasePath" : "GlobPatternBasePath"
        },
        "Timeout" : 1.2315135367772556,
        "inputs" : [ null, null ],
        "DoNotTrack" : true,
        "Publisher" : 4,
        "Verifier" : 1,
        "Contexts" : [ null, null ],
        "Wasm" : {
          "EnvironmentVariables" : {
            "key" : "EnvironmentVariables"
          },
          "Parameters" : [ "Parameters", "Parameters" ],
          "ImportModules" : [ null, null ],
          "EntryPoint" : "EntryPoint"
        },
        "Annotations" : [ "Annotations", "Annotations" ],
        "Docker" : {
          "WorkingDirectory" : "WorkingDirectory",
          "EnvironmentVariables" : [ "EnvironmentVariables", "EnvironmentVariables" ],
          "Entrypoint" : [ "Entrypoint", "Entrypoint" ],
          "Image" : "Image"
        },
        "Language" : {
          "RequirementsPath" : "RequirementsPath",
          "Language" : "Language",
          "Command" : "Command",
          "DeterministicExecution" : true,
          "LanguageVersion" : "LanguageVersion",
          "ProgramPath" : "ProgramPath"
        },
        "Resources" : {
          "Memory" : "Memory",
          "CPU" : "CPU",
          "Disk" : "Disk",
          "GPU" : "GPU"
        },
        "Engine" : 2
      },
      "JobID" : "9304c616-291f-41ad-b862-54e133c0149e"
    } ],
    "Spec" : {
      "outputs" : [ null, null ],
      "Sharding" : {
        "BatchSize" : 7,
        "GlobPattern" : "GlobPattern",
        "GlobPatternBasePath" : "GlobPatternBasePath"
      },
      "Timeout" : 1.2315135367772556,
      "inputs" : [ null, null ],
      "DoNotTrack" : true,
      "Publisher" : 4,
      "Verifier" : 1,
      "Contexts" : [ null, null ],
      "Wasm" : {
        "EnvironmentVariables" : {
          "key" : "EnvironmentVariables"
        },
        "Parameters" : [ "Parameters", "Parameters" ],
        "ImportModules" : [ null, null ],
        "EntryPoint" : "EntryPoint"
      },
      "Annotations" : [ "Annotations", "Annotations" ],
      "Docker" : {
        "WorkingDirectory" : "WorkingDirectory",
        "EnvironmentVariables" : [ "EnvironmentVariables", "EnvironmentVariables" ],
        "Entrypoint" : [ "Entrypoint", "Entrypoint" ],
        "Image" : "Image"
      },
      "Language" : {
        "RequirementsPath" : "RequirementsPath",
        "Language" : "Language",
        "Command" : "Command",
        "DeterministicExecution" : true,
        "LanguageVersion" : "LanguageVersion",
        "ProgramPath" : "ProgramPath"
      },
      "Resources" : {
        "Memory" : "Memory",
        "CPU" : "CPU",
        "Disk" : "Disk",
        "GPU" : "GPU"
      },
      "Engine" : 2
    }
  }, {
    "RequesterNodeID" : "QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF",
    "ExecutionPlan" : {
      "ShardsTotal" : 5
    },
    "LocalJobEvents" : [ {
      "TargetNodeID" : "TargetNodeID",
      "ShardIndex" : 5,
      "EventName" : 5,
      "JobID" : "JobID"
    }, {
      "TargetNodeID" : "TargetNodeID",
      "ShardIndex" : 5,
      "EventName" : 5,
      "JobID" : "JobID"
    } ],
    "APIVersion" : "V1beta1",
    "CreatedAt" : "2022-11-17T13:29:01.871140291Z",
    "ClientID" : "ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51",
    "JobState" : {
      "Nodes" : {
        "key" : {
          "Shards" : {
            "key" : {
              "Status" : "Status",
              "RunOutput" : {
                "stderrtruncated" : true,
                "stdout" : "stdout",
                "exitCode" : 7,
                "runnerError" : "runnerError",
                "stdouttruncated" : true,
                "stderr" : "stderr"
              },
              "VerificationProposal" : [ 1, 1 ],
              "VerificationResult" : {
                "Complete" : true,
                "Result" : true
              },
              "ShardIndex" : 0,
              "State" : 6,
              "NodeId" : "NodeId",
              "PublishedResults" : {
                "path" : "path",
                "Metadata" : {
                  "key" : "Metadata"
                },
                "URL" : "URL",
                "CID" : "QmTVmC7JBD2ES2qGPqBNVWnX1KeEPNrPGb7rJ8cpFgtefe",
                "Name" : "job-9304c616-291f-41ad-b862-54e133c0149e-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
                "StorageSource" : 2
              }
            }
          }
        }
      }
    },
    "RequesterPublicKey" : [ 2, 2 ],
    "ID" : "92d5d4ee-3765-4f78-8353-623f5f26df08",
    "Deal" : {
      "MinBids" : 1,
      "Concurrency" : 0,
      "Confidence" : 6
    },
    "JobEvents" : [ {
      "Status" : "Got results proposal of length: 0",
      "JobExecutionPlan" : {
        "ShardsTotal" : 5
      },
      "SourceNodeID" : "QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF",
      "ShardIndex" : 3,
      "EventName" : 5,
      "Deal" : {
        "MinBids" : 1,
        "Concurrency" : 0,
        "Confidence" : 6
      },
      "PublishedResult" : {
        "path" : "path",
        "Metadata" : {
          "key" : "Metadata"
        },
        "URL" : "URL",
        "CID" : "QmTVmC7JBD2ES2qGPqBNVWnX1KeEPNrPGb7rJ8cpFgtefe",
        "Name" : "job-9304c616-291f-41ad-b862-54e133c0149e-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
        "StorageSource" : 2
      },
      "TargetNodeID" : "QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
      "RunOutput" : {
        "stderrtruncated" : true,
        "stdout" : "stdout",
        "exitCode" : 7,
        "runnerError" : "runnerError",
        "stdouttruncated" : true,
        "stderr" : "stderr"
      },
      "VerificationProposal" : [ 1, 1 ],
      "VerificationResult" : {
        "Complete" : true,
        "Result" : true
      },
      "APIVersion" : "V1beta1",
      "SenderPublicKey" : [ 9, 9 ],
      "EventTime" : "2022-11-17T13:32:55.756658941Z",
      "ClientID" : "ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51",
      "Spec" : {
        "outputs" : [ null, null ],
        "Sharding" : {
          "BatchSize" : 7,
          "GlobPattern" : "GlobPattern",
          "GlobPatternBasePath" : "GlobPatternBasePath"
        },
        "Timeout" : 1.2315135367772556,
        "inputs" : [ null, null ],
        "DoNotTrack" : true,
        "Publisher" : 4,
        "Verifier" : 1,
        "Contexts" : [ null, null ],
        "Wasm" : {
          "EnvironmentVariables" : {
            "key" : "EnvironmentVariables"
          },
          "Parameters" : [ "Parameters", "Parameters" ],
          "ImportModules" : [ null, null ],
          "EntryPoint" : "EntryPoint"
        },
        "Annotations" : [ "Annotations", "Annotations" ],
        "Docker" : {
          "WorkingDirectory" : "WorkingDirectory",
          "EnvironmentVariables" : [ "EnvironmentVariables", "EnvironmentVariables" ],
          "Entrypoint" : [ "Entrypoint", "Entrypoint" ],
          "Image" : "Image"
        },
        "Language" : {
          "RequirementsPath" : "RequirementsPath",
          "Language" : "Language",
          "Command" : "Command",
          "DeterministicExecution" : true,
          "LanguageVersion" : "LanguageVersion",
          "ProgramPath" : "ProgramPath"
        },
        "Resources" : {
          "Memory" : "Memory",
          "CPU" : "CPU",
          "Disk" : "Disk",
          "GPU" : "GPU"
        },
        "Engine" : 2
      },
      "JobID" : "9304c616-291f-41ad-b862-54e133c0149e"
    }, {
      "Status" : "Got results proposal of length: 0",
      "JobExecutionPlan" : {
        "ShardsTotal" : 5
      },
      "SourceNodeID" : "QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF",
      "ShardIndex" : 3,
      "EventName" : 5,
      "Deal" : {
        "MinBids" : 1,
        "Concurrency" : 0,
        "Confidence" : 6
      },
      "PublishedResult" : {
        "path" : "path",
        "Metadata" : {
          "key" : "Metadata"
        },
        "URL" : "URL",
        "CID" : "QmTVmC7JBD2ES2qGPqBNVWnX1KeEPNrPGb7rJ8cpFgtefe",
        "Name" : "job-9304c616-291f-41ad-b862-54e133c0149e-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
        "StorageSource" : 2
      },
      "TargetNodeID" : "QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
      "RunOutput" : {
        "stderrtruncated" : true,
        "stdout" : "stdout",
        "exitCode" : 7,
        "runnerError" : "runnerError",
        "stdouttruncated" : true,
        "stderr" : "stderr"
      },
      "VerificationProposal" : [ 1, 1 ],
      "VerificationResult" : {
        "Complete" : true,
        "Result" : true
      },
      "APIVersion" : "V1beta1",
      "SenderPublicKey" : [ 9, 9 ],
      "EventTime" : "2022-11-17T13:32:55.756658941Z",
      "ClientID" : "ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51",
      "Spec" : {
        "outputs" : [ null, null ],
        "Sharding" : {
          "BatchSize" : 7,
          "GlobPattern" : "GlobPattern",
          "GlobPatternBasePath" : "GlobPatternBasePath"
        },
        "Timeout" : 1.2315135367772556,
        "inputs" : [ null, null ],
        "DoNotTrack" : true,
        "Publisher" : 4,
        "Verifier" : 1,
        "Contexts" : [ null, null ],
        "Wasm" : {
          "EnvironmentVariables" : {
            "key" : "EnvironmentVariables"
          },
          "Parameters" : [ "Parameters", "Parameters" ],
          "ImportModules" : [ null, null ],
          "EntryPoint" : "EntryPoint"
        },
        "Annotations" : [ "Annotations", "Annotations" ],
        "Docker" : {
          "WorkingDirectory" : "WorkingDirectory",
          "EnvironmentVariables" : [ "EnvironmentVariables", "EnvironmentVariables" ],
          "Entrypoint" : [ "Entrypoint", "Entrypoint" ],
          "Image" : "Image"
        },
        "Language" : {
          "RequirementsPath" : "RequirementsPath",
          "Language" : "Language",
          "Command" : "Command",
          "DeterministicExecution" : true,
          "LanguageVersion" : "LanguageVersion",
          "ProgramPath" : "ProgramPath"
        },
        "Resources" : {
          "Memory" : "Memory",
          "CPU" : "CPU",
          "Disk" : "Disk",
          "GPU" : "GPU"
        },
        "Engine" : 2
      },
      "JobID" : "9304c616-291f-41ad-b862-54e133c0149e"
    } ],
    "Spec" : {
      "outputs" : [ null, null ],
      "Sharding" : {
        "BatchSize" : 7,
        "GlobPattern" : "GlobPattern",
        "GlobPatternBasePath" : "GlobPatternBasePath"
      },
      "Timeout" : 1.2315135367772556,
      "inputs" : [ null, null ],
      "DoNotTrack" : true,
      "Publisher" : 4,
      "Verifier" : 1,
      "Contexts" : [ null, null ],
      "Wasm" : {
        "EnvironmentVariables" : {
          "key" : "EnvironmentVariables"
        },
        "Parameters" : [ "Parameters", "Parameters" ],
        "ImportModules" : [ null, null ],
        "EntryPoint" : "EntryPoint"
      },
      "Annotations" : [ "Annotations", "Annotations" ],
      "Docker" : {
        "WorkingDirectory" : "WorkingDirectory",
        "EnvironmentVariables" : [ "EnvironmentVariables", "EnvironmentVariables" ],
        "Entrypoint" : [ "Entrypoint", "Entrypoint" ],
        "Image" : "Image"
      },
      "Language" : {
        "RequirementsPath" : "RequirementsPath",
        "Language" : "Language",
        "Command" : "Command",
        "DeterministicExecution" : true,
        "LanguageVersion" : "LanguageVersion",
        "ProgramPath" : "ProgramPath"
      },
      "Resources" : {
        "Memory" : "Memory",
        "CPU" : "CPU",
        "Disk" : "Disk",
        "GPU" : "GPU"
      },
      "Engine" : 2
    }
  } ]
};
    if (Object.keys(examples).length > 0) {
      resolve(examples[Object.keys(examples)[0]]);
    } else {
      resolve();
    }
  });
}


/**
 * Returns the events related to the job-id passed in the body payload. Useful for troubleshooting.
 * Events (e.g. Created, Bid, BidAccepted, ..., ResultsAccepted, ResultsPublished) are useful to track the progress of a job.  Example response (truncated): ```json {   \"events\": [     {       \"APIVersion\": \"V1beta1\",       \"JobID\": \"9304c616-291f-41ad-b862-54e133c0149e\",       \"ClientID\": \"ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51\",       \"SourceNodeID\": \"QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF\",       \"EventName\": \"Created\",       \"Spec\": {         \"Engine\": \"Docker\",         \"Verifier\": \"Noop\",         \"Publisher\": \"Estuary\",         \"Docker\": {           \"Image\": \"ubuntu\",           \"Entrypoint\": [             \"date\"           ]         },         \"Language\": {           \"JobContext\": {}         },         \"Wasm\": {},         \"Resources\": {           \"GPU\": \"\"         },         \"Timeout\": 1800,         \"outputs\": [           {             \"StorageSource\": \"IPFS\",             \"Name\": \"outputs\",             \"path\": \"/outputs\"           }         ],         \"Sharding\": {           \"BatchSize\": 1,           \"GlobPatternBasePath\": \"/inputs\"         }       },       \"JobExecutionPlan\": {         \"ShardsTotal\": 1       },       \"Deal\": {         \"Concurrency\": 1       },       \"VerificationResult\": {},       \"PublishedResult\": {},       \"EventTime\": \"2022-11-17T13:32:55.331375351Z\",       \"SenderPublicKey\": \"...\"     },     ...     {       \"JobID\": \"9304c616-291f-41ad-b862-54e133c0149e\",       \"SourceNodeID\": \"QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF\",       \"TargetNodeID\": \"QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\",       \"EventName\": \"ResultsAccepted\",       \"Spec\": {         \"Docker\": {},         \"Language\": {           \"JobContext\": {}         },         \"Wasm\": {},         \"Resources\": {           \"GPU\": \"\"         },         \"Sharding\": {}       },       \"JobExecutionPlan\": {},       \"Deal\": {},       \"VerificationResult\": {         \"Complete\": true,         \"Result\": true       },       \"PublishedResult\": {},       \"EventTime\": \"2022-11-17T13:32:55.707825569Z\",       \"SenderPublicKey\": \"...\"     },     {       \"JobID\": \"9304c616-291f-41ad-b862-54e133c0149e\",       \"SourceNodeID\": \"QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\",       \"EventName\": \"ResultsPublished\",       \"Spec\": {         \"Docker\": {},         \"Language\": {           \"JobContext\": {}         },         \"Wasm\": {},         \"Resources\": {           \"GPU\": \"\"         },         \"Sharding\": {}       },       \"JobExecutionPlan\": {},       \"Deal\": {},       \"VerificationResult\": {},       \"PublishedResult\": {         \"StorageSource\": \"IPFS\",         \"Name\": \"job-9304c616-291f-41ad-b862-54e133c0149e-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\",         \"CID\": \"QmTVmC7JBD2ES2qGPqBNVWnX1KeEPNrPGb7rJ8cpFgtefe\"       },       \"EventTime\": \"2022-11-17T13:32:55.756658941Z\",       \"SenderPublicKey\": \"...\"     }   ] } ```
 *
 * body Publicapi.eventsRequest Request must specify a `client_id`. To retrieve your `client_id`, you can do the following: (1) submit a dummy job to Bacalhau (or use one you created before), (2) run `bacalhau describe <job-id>` and fetch the `ClientID` field.
 * returns publicapi.eventsResponse
 **/
exports.pkg/publicapi/events = function(body) {
  return new Promise(function(resolve, reject) {
    var examples = {};
    examples['application/json'] = {
  "events" : [ {
    "Status" : "Got results proposal of length: 0",
    "JobExecutionPlan" : {
      "ShardsTotal" : 5
    },
    "SourceNodeID" : "QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF",
    "ShardIndex" : 3,
    "EventName" : 5,
    "Deal" : {
      "MinBids" : 1,
      "Concurrency" : 0,
      "Confidence" : 6
    },
    "PublishedResult" : {
      "path" : "path",
      "Metadata" : {
        "key" : "Metadata"
      },
      "URL" : "URL",
      "CID" : "QmTVmC7JBD2ES2qGPqBNVWnX1KeEPNrPGb7rJ8cpFgtefe",
      "Name" : "job-9304c616-291f-41ad-b862-54e133c0149e-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
      "StorageSource" : 2
    },
    "TargetNodeID" : "QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
    "RunOutput" : {
      "stderrtruncated" : true,
      "stdout" : "stdout",
      "exitCode" : 7,
      "runnerError" : "runnerError",
      "stdouttruncated" : true,
      "stderr" : "stderr"
    },
    "VerificationProposal" : [ 1, 1 ],
    "VerificationResult" : {
      "Complete" : true,
      "Result" : true
    },
    "APIVersion" : "V1beta1",
    "SenderPublicKey" : [ 9, 9 ],
    "EventTime" : "2022-11-17T13:32:55.756658941Z",
    "ClientID" : "ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51",
    "Spec" : {
      "outputs" : [ null, null ],
      "Sharding" : {
        "BatchSize" : 7,
        "GlobPattern" : "GlobPattern",
        "GlobPatternBasePath" : "GlobPatternBasePath"
      },
      "Timeout" : 1.2315135367772556,
      "inputs" : [ null, null ],
      "DoNotTrack" : true,
      "Publisher" : 4,
      "Verifier" : 1,
      "Contexts" : [ null, null ],
      "Wasm" : {
        "EnvironmentVariables" : {
          "key" : "EnvironmentVariables"
        },
        "Parameters" : [ "Parameters", "Parameters" ],
        "ImportModules" : [ null, null ],
        "EntryPoint" : "EntryPoint"
      },
      "Annotations" : [ "Annotations", "Annotations" ],
      "Docker" : {
        "WorkingDirectory" : "WorkingDirectory",
        "EnvironmentVariables" : [ "EnvironmentVariables", "EnvironmentVariables" ],
        "Entrypoint" : [ "Entrypoint", "Entrypoint" ],
        "Image" : "Image"
      },
      "Language" : {
        "RequirementsPath" : "RequirementsPath",
        "Language" : "Language",
        "Command" : "Command",
        "DeterministicExecution" : true,
        "LanguageVersion" : "LanguageVersion",
        "ProgramPath" : "ProgramPath"
      },
      "Resources" : {
        "Memory" : "Memory",
        "CPU" : "CPU",
        "Disk" : "Disk",
        "GPU" : "GPU"
      },
      "Engine" : 2
    },
    "JobID" : "9304c616-291f-41ad-b862-54e133c0149e"
  }, {
    "Status" : "Got results proposal of length: 0",
    "JobExecutionPlan" : {
      "ShardsTotal" : 5
    },
    "SourceNodeID" : "QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF",
    "ShardIndex" : 3,
    "EventName" : 5,
    "Deal" : {
      "MinBids" : 1,
      "Concurrency" : 0,
      "Confidence" : 6
    },
    "PublishedResult" : {
      "path" : "path",
      "Metadata" : {
        "key" : "Metadata"
      },
      "URL" : "URL",
      "CID" : "QmTVmC7JBD2ES2qGPqBNVWnX1KeEPNrPGb7rJ8cpFgtefe",
      "Name" : "job-9304c616-291f-41ad-b862-54e133c0149e-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
      "StorageSource" : 2
    },
    "TargetNodeID" : "QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
    "RunOutput" : {
      "stderrtruncated" : true,
      "stdout" : "stdout",
      "exitCode" : 7,
      "runnerError" : "runnerError",
      "stdouttruncated" : true,
      "stderr" : "stderr"
    },
    "VerificationProposal" : [ 1, 1 ],
    "VerificationResult" : {
      "Complete" : true,
      "Result" : true
    },
    "APIVersion" : "V1beta1",
    "SenderPublicKey" : [ 9, 9 ],
    "EventTime" : "2022-11-17T13:32:55.756658941Z",
    "ClientID" : "ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51",
    "Spec" : {
      "outputs" : [ null, null ],
      "Sharding" : {
        "BatchSize" : 7,
        "GlobPattern" : "GlobPattern",
        "GlobPatternBasePath" : "GlobPatternBasePath"
      },
      "Timeout" : 1.2315135367772556,
      "inputs" : [ null, null ],
      "DoNotTrack" : true,
      "Publisher" : 4,
      "Verifier" : 1,
      "Contexts" : [ null, null ],
      "Wasm" : {
        "EnvironmentVariables" : {
          "key" : "EnvironmentVariables"
        },
        "Parameters" : [ "Parameters", "Parameters" ],
        "ImportModules" : [ null, null ],
        "EntryPoint" : "EntryPoint"
      },
      "Annotations" : [ "Annotations", "Annotations" ],
      "Docker" : {
        "WorkingDirectory" : "WorkingDirectory",
        "EnvironmentVariables" : [ "EnvironmentVariables", "EnvironmentVariables" ],
        "Entrypoint" : [ "Entrypoint", "Entrypoint" ],
        "Image" : "Image"
      },
      "Language" : {
        "RequirementsPath" : "RequirementsPath",
        "Language" : "Language",
        "Command" : "Command",
        "DeterministicExecution" : true,
        "LanguageVersion" : "LanguageVersion",
        "ProgramPath" : "ProgramPath"
      },
      "Resources" : {
        "Memory" : "Memory",
        "CPU" : "CPU",
        "Disk" : "Disk",
        "GPU" : "GPU"
      },
      "Engine" : 2
    },
    "JobID" : "9304c616-291f-41ad-b862-54e133c0149e"
  } ]
};
    if (Object.keys(examples).length > 0) {
      resolve(examples[Object.keys(examples)[0]]);
    } else {
      resolve();
    }
  });
}


/**
 * Returns the node's local events related to the job-id passed in the body payload. Useful for troubleshooting.
 * Local events (e.g. Selected, BidAccepted, Verified) are useful to track the progress of a job.
 *
 * body Publicapi.localEventsRequest 
 * returns publicapi.localEventsResponse
 **/
exports.pkg/publicapi/localEvents = function(body) {
  return new Promise(function(resolve, reject) {
    var examples = {};
    examples['application/json'] = {
  "localEvents" : [ {
    "TargetNodeID" : "TargetNodeID",
    "ShardIndex" : 5,
    "EventName" : 5,
    "JobID" : "JobID"
  }, {
    "TargetNodeID" : "TargetNodeID",
    "ShardIndex" : 5,
    "EventName" : 5,
    "JobID" : "JobID"
  } ]
};
    if (Object.keys(examples).length > 0) {
      resolve(examples[Object.keys(examples)[0]]);
    } else {
      resolve();
    }
  });
}


/**
 * Returns the results of the job-id specified in the body payload.
 * Example response:  ```json {   \"results\": [     {       \"NodeID\": \"QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\",       \"Data\": {         \"StorageSource\": \"IPFS\",         \"Name\": \"job-9304c616-291f-41ad-b862-54e133c0149e-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\",         \"CID\": \"QmTVmC7JBD2ES2qGPqBNVWnX1KeEPNrPGb7rJ8cpFgtefe\"       }     }   ] } ```
 *
 * body Publicapi.stateRequest 
 * returns publicapi.resultsResponse
 **/
exports.pkg/publicapi/results = function(body) {
  return new Promise(function(resolve, reject) {
    var examples = {};
    examples['application/json'] = {
  "results" : [ {
    "ShardIndex" : 0,
    "NodeID" : "NodeID",
    "Data" : {
      "path" : "path",
      "Metadata" : {
        "key" : "Metadata"
      },
      "URL" : "URL",
      "CID" : "QmTVmC7JBD2ES2qGPqBNVWnX1KeEPNrPGb7rJ8cpFgtefe",
      "Name" : "job-9304c616-291f-41ad-b862-54e133c0149e-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
      "StorageSource" : 2
    }
  }, {
    "ShardIndex" : 0,
    "NodeID" : "NodeID",
    "Data" : {
      "path" : "path",
      "Metadata" : {
        "key" : "Metadata"
      },
      "URL" : "URL",
      "CID" : "QmTVmC7JBD2ES2qGPqBNVWnX1KeEPNrPGb7rJ8cpFgtefe",
      "Name" : "job-9304c616-291f-41ad-b862-54e133c0149e-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
      "StorageSource" : 2
    }
  } ]
};
    if (Object.keys(examples).length > 0) {
      resolve(examples[Object.keys(examples)[0]]);
    } else {
      resolve();
    }
  });
}


/**
 * Returns the state of the job-id specified in the body payload.
 * Example response:  ```json {   \"state\": {     \"Nodes\": {       \"QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86\": {         \"Shards\": {           \"0\": {             \"NodeId\": \"QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86\",             \"State\": \"Cancelled\",             \"VerificationResult\": {},             \"PublishedResults\": {}           }         }       },       \"QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3\": {         \"Shards\": {           \"0\": {             \"NodeId\": \"QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3\",             \"State\": \"Cancelled\",             \"VerificationResult\": {},             \"PublishedResults\": {}           }         }       },       \"QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\": {         \"Shards\": {           \"0\": {             \"NodeId\": \"QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\",             \"State\": \"Completed\",             \"Status\": \"Got results proposal of length: 0\",             \"VerificationResult\": {               \"Complete\": true,               \"Result\": true             },             \"PublishedResults\": {               \"StorageSource\": \"IPFS\",               \"Name\": \"job-9304c616-291f-41ad-b862-54e133c0149e-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\",               \"CID\": \"QmTVmC7JBD2ES2qGPqBNVWnX1KeEPNrPGb7rJ8cpFgtefe\"             },             \"RunOutput\": {               \"stdout\": \"Thu Nov 17 13:32:55 UTC 2022\\n\",               \"stdouttruncated\": false,               \"stderr\": \"\",               \"stderrtruncated\": false,               \"exitCode\": 0,               \"runnerError\": \"\"             }           }         }       }     }   } } ```
 *
 * body Publicapi.stateRequest 
 * returns publicapi.stateResponse
 **/
exports.pkg/publicapi/states = function(body) {
  return new Promise(function(resolve, reject) {
    var examples = {};
    examples['application/json'] = {
  "state" : {
    "Nodes" : {
      "key" : {
        "Shards" : {
          "key" : {
            "Status" : "Status",
            "RunOutput" : {
              "stderrtruncated" : true,
              "stdout" : "stdout",
              "exitCode" : 7,
              "runnerError" : "runnerError",
              "stdouttruncated" : true,
              "stderr" : "stderr"
            },
            "VerificationProposal" : [ 1, 1 ],
            "VerificationResult" : {
              "Complete" : true,
              "Result" : true
            },
            "ShardIndex" : 0,
            "State" : 6,
            "NodeId" : "NodeId",
            "PublishedResults" : {
              "path" : "path",
              "Metadata" : {
                "key" : "Metadata"
              },
              "URL" : "URL",
              "CID" : "QmTVmC7JBD2ES2qGPqBNVWnX1KeEPNrPGb7rJ8cpFgtefe",
              "Name" : "job-9304c616-291f-41ad-b862-54e133c0149e-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
              "StorageSource" : 2
            }
          }
        }
      }
    }
  }
};
    if (Object.keys(examples).length > 0) {
      resolve(examples[Object.keys(examples)[0]]);
    } else {
      resolve();
    }
  });
}

