/* tslint:disable */
/* eslint-disable */
import { http, HttpResponse, RequestHandler, RequestHandlerOptions } from "msw";
import { Job, JobsResponse } from "../src/helpers/jobInterfaces";
import { Node, NodesResponse } from "../src/helpers/nodeInterfaces";

export const JOBS_RETURN_LIMIT = 10
export const NODES_RETURN_LIMIT = 10
const BASE_URL = "https://localhost:1234"
const storyDataPath = '../src/stories/storyData';

console.log("storyDataPath", storyDataPath)

import * as internalJobs from '../src/stories/storyData/jobsTable/100-jobs.json'
import * as internalNodes from '../src/stories/storyData/nodesTable/100-nodes.json'

export function getJobs() : Job[] {
    return internalJobs
}
export function getNodes(): Node[] {
    return internalNodes
}

export const jobsResponse = http.get('/api/v1/orchestrator/jobs', ({ request }) => {
    console.log("jobsResponse request", request)
    const limitedJobs = getJobs().slice(0, JOBS_RETURN_LIMIT)
    const jobsListResponse: JobsResponse = { Jobs: limitedJobs, NextToken: "" }
    return HttpResponse.json(jobsListResponse)
})

export const nodesResponse = http.get('/api/v1/orchestrator/nodes', ({ request }) => {
    // return number of nodes based on limit
    const limitedNodes = getNodes().slice(0, NODES_RETURN_LIMIT)
    const nodesListResponse: NodesResponse = { Nodes: limitedNodes, NextToken: "" }
    return HttpResponse.json(nodesListResponse)
})

export const handlers: RequestHandler<any, any, any, RequestHandlerOptions>[] = [jobsResponse, nodesResponse]
