// src/mocks/handlers.js
import { http, HttpResponse } from "msw";
import { Job } from "../../src/helpers/jobInterfaces";
import { Node, NodesResponse } from "../../src/helpers/nodeInterfaces";

export const JOBS_RETURN_LIMIT = 10
export const NODES_RETURN_LIMIT = 10
const BASE_URL = "https://localhost:1234"

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

const jobHandlers = http.get('http://localhost:1234/api/v1/orchestrator/jobs', ({ request }) => {
    // return number of nodes based on limit
    const limitedNodes = internalNodes.slice(0, NODES_RETURN_LIMIT)
    const nodesListResponse: NodesResponse = { Nodes: limitedNodes, NextToken: "" }
    return HttpResponse.json(nodesListResponse)
})

export const storybookHandlers = [jobHandlers]