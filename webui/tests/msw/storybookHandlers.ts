// src/mocks/handlers.js
import { http, HttpResponse } from "msw"
import { Job } from "../../src/helpers/jobInterfaces"
import { Node } from "../../src/helpers/nodeInterfaces"

export const JOBS_RETURN_LIMIT = 10
export const NODES_RETURN_LIMIT = 10
// const BASE_URL = "https://localhost:1234"

let internalJobs: Job[] = []

export function getJobs() {
  return internalJobs
}

export function setJobs(jobs: Job[]) {
  internalJobs = jobs
}

let internalNodes: Node[] = []

export function getNodes() {
  return internalNodes
}

export function setNodes(nodes: Node[]) {
  internalNodes = nodes
}

const jobHandlers = http.get(
  "http://localhost:1234/api/v1/orchestrator/jobs",
  () => HttpResponse.json(getJobs())
)

export const storybookHandlers = [jobHandlers]
