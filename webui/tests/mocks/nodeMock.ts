/* eslint-disable @typescript-eslint/no-unsafe-member-access */
/* eslint-disable @typescript-eslint/no-unsafe-call */
/* eslint-disable @typescript-eslint/no-unsafe-assignment */
import { datatype, git, internet, os, system } from "@faker-js/faker"

import { v4 as uuidv4 } from "uuid"
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

export function generateMockNode() {
  const id = uuidv4()
  const cpu = Math.random() * 100
  const memory = Math.floor(Math.random() * 1000000000000)
  const disk = Math.floor(Math.random() * 10000000000000)

  const availableCPU = Math.floor(Math.random() * cpu)
  const availableMemory = Math.floor(Math.random() * memory)
  const availableDisk = Math.floor(Math.random() * disk)

  const maxJobCPU = Math.floor(Math.random() * availableCPU)
  const maxJobMemory = Math.floor(Math.random() * availableMemory)
  const maxJobDisk = Math.floor(Math.random() * availableDisk)

  const majorVersion: string = datatype.number({ min: 99, max: 199 }).toString()
  const minorVersion: string = datatype.number({ min: 99, max: 199 }).toString()

  return {
    PeerInfo: {
      ID: id,
      Addrs: [
        `${internet.ip()}/udp/${datatype.number({
          min: 2048,
          max: 65535,
        })}/quic-v1`,
      ],
    },
    NodeType: "Compute",
    Labels: {
      Architecture: system.arch(),
      "Operating-System": os.platform(),
      "git-lfs": datatype.boolean().toString(),
      owner: internet.userName(),
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
      GitVersion: system.semver(),
      GitCommit: git.shortSha(),
      BuildDate: new Date().toISOString(),
      GOOS: system.platform(),
      GOARCH: system.architecture(),
    },
  }
}
