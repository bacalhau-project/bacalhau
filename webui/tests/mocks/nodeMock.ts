/* eslint-disable @typescript-eslint/no-unsafe-member-access */
import { faker } from "@faker-js/faker"
import { Node } from "../../src/helpers/nodeInterfaces"
import { randomUUID } from "./cryptoFunctions"
import { selectRandomElements } from "./mockUtilities"

// Generate an array with one or more of the following strings
const engineTypes = ["wasm", "docker"]
const publisherTypes = ["noop", "ipfs", "s3", "inline", "urldownload"]
const storageSources = [
  "ipfs",
  "urldownload",
  "inline",
  "repoclone",
  "repoclonelfs",
  "s3",
]

export function generateMockNode(): Node {
  const id = randomUUID()
  const cpu = Math.random() * 100
  const memory = Math.floor(Math.random() * 1000000000000)
  const disk = Math.floor(Math.random() * 10000000000000)

  const availableCPU = Math.floor(Math.random() * cpu)
  const availableMemory = Math.floor(Math.random() * memory)
  const availableDisk = Math.floor(Math.random() * disk)

  const maxJobCPU = Math.floor(Math.random() * availableCPU)
  const maxJobMemory = Math.floor(Math.random() * availableMemory)
  const maxJobDisk = Math.floor(Math.random() * availableDisk)

  const majorVersion: string = faker.number
    .int({ min: 99, max: 199 })
    .toString()
  const minorVersion: string = faker.number
    .int({ min: 99, max: 199 })
    .toString()

  return {
    PeerInfo: {
      ID: id,
      Addrs: [
        `${faker.internet.ip()}/udp/${faker.number.int({
          min: 2048,
          max: 65535,
        })}/quic-v1`,
      ],
    },
    NodeType: "Compute",
    Labels: {
      Architecture: `arch-${faker.string.alphanumeric(10)}`,
      "Operating-System": `os-${faker.string.alphanumeric(10)}`,
      "git-lfs": faker.datatype.boolean().toString(),
      name: faker.internet.userName(),
      env: `env-${faker.string.alphanumeric(100)}`,
    },
    ComputeNodeInfo: {
      ExecutionEngines: selectRandomElements(engineTypes, 1),
      Publishers: selectRandomElements(publisherTypes, 1),
      StorageSources: selectRandomElements(storageSources, 1),
      MaxCapacity: {
        CPU: cpu,
        Memory: memory,
        Disk: disk,
      },
      AvailableCapacity: {
        CPU: availableCPU,
        Memory: availableMemory,
        Disk: availableDisk,
      },
      MaxJobRequirements: {
        // Randomly generate a number between 0 and the available capacity
        CPU: maxJobCPU,
        Memory: maxJobMemory,
        Disk: maxJobDisk,
      },
      RunningExecutions: 0,
      EnqueuedExecutions: 0,
    },
    BacalhauVersion: {
      Major: majorVersion,
      Minor: minorVersion,
      // Random semantic versioning
      GitVersion: faker.system.semver(),
      GitCommit: faker.git.commitSha({ length: 7 }),
      BuildDate: new Date().toISOString(),
      GOOS: `GOOS-${faker.string.alphanumeric(10)}`,
      GOARCH: `GOARCH-${faker.string.alphanumeric(10)}`,
    },
  }
}
