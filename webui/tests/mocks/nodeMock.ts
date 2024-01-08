import * as faker from "faker";
import { v4 as uuidv4 } from "uuid";
import { selectRandomLabels, selectRandomElements } from "./mockUtilities";

// Generate an array with one or more of the following strings
const engine_types = ["wasm", "docker"];
const publisher_types = ["noop", "ipfs", "s3", "inline", "urldownload"];
const storage_sources = [
  "ipfs",
  "urldownload",
  "inline",
  "repoclone",
  "repoclonelfs",
  "s3",
];

export function generateMockNode() {
  const id = uuidv4();
  const cpu = Math.random() * 100;
  const memory = Math.floor(Math.random() * 1000000000000);
  const disk = Math.floor(Math.random() * 10000000000000);

  const available_cpu = Math.floor(Math.random() * cpu);
  const available_memory = Math.floor(Math.random() * memory);
  const available_disk = Math.floor(Math.random() * disk);

  const max_job_cpu = Math.floor(Math.random() * available_cpu);
  const max_job_memory = Math.floor(Math.random() * available_memory);
  const max_job_disk = Math.floor(Math.random() * available_disk);

  const major_version = faker.datatype.number({ min: 99, max: 199 }).toString();
  const minor_version = faker.datatype.number({ min: 99, max: 199 }).toString();

  return {
    PeerInfo: {
      ID: id,
      Addrs: [
        `${faker.internet.ip()}/udp/${faker.datatype.number({
          min: 2048,
          max: 65535,
        })}/quic-v1`,
      ],
    },
    NodeType: "Compute",
    Labels: {
      Architecture: faker.system.arch(),
      "Operating-System": faker.os.platform(),
      "git-lfs": faker.datatype.boolean().toString(),
      owner: faker.internet.userName(),
    },
    ComputeNodeInfo: {
      ExecutionEngines: selectRandomElements(engine_types, 1),
      Publishers: selectRandomElements(publisher_types, 1),
      StorageSources: selectRandomElements(storage_sources, 1),
      MaxCapacity: {
        CPU: cpu,
        Memory: memory,
        Disk: disk,
      },
      AvailableCapacity: {
        CPU: available_cpu,
        Memory: available_memory,
        Disk: available_disk,
      },
      MaxJobRequirements: {
        // Randomly generate a number between 0 and the available capacity
        CPU: max_job_cpu,
        Memory: max_job_memory,
        Disk: max_job_disk,
      },
      RunningExecutions: 0,
      EnqueuedExecutions: 0,
    },
    BacalhauVersion: {
      Major: major_version,
      Minor: minor_version,
      // Random semantic versioning
      GitVersion: faker.system.semver(),
      GitCommit: faker.git.shortSha(),
      BuildDate: new Date().toISOString(),
      GOOS: faker.system.platform(),
      GOARCH: faker.system.architecture(),
    },
  };
}
