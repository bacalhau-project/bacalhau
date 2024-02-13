/* tslint:disable */
/* eslint-disable */
import { http, HttpResponse, RequestHandler, RequestHandlerOptions } from "msw";
import { Job, JobsResponse } from "../src/helpers/jobInterfaces";
import { Node, NodesResponse } from "../src/helpers/nodeInterfaces";
import { el } from "@faker-js/faker";



export const JOBS_RETURN_LIMIT = 10
export const NODES_RETURN_LIMIT = 10
const BASE_URL = "https://localhost:1234"
const storyDataPath = '../src/stories/storyData';

let internalJobs: Record<string, Job[]> = {};
let internalNodes: Record<string, Node[]> = {};

console.log("storyDataPath", storyDataPath)

// List all files in storyData directory as an array of paths
// Import all the content of the file as a string without using fs.readFileSync
const fileList: Record<string, string> = {
    "0-jobs": "../src/stories/storyData/jobsTable/0-jobs.json",
    "10-jobs": "../src/stories/storyData/jobsTable/10-jobs.json",
    "100-jobs": "../src/stories/storyData/jobsTable/100-jobs.json",
}

// Import the contents from filelist using import
// and store them in internalJobs
async function loadData() {
    for (const [key, value] of Object.entries(fileList)) {
        // If filename contains "jobs" then store in internalJobs
        if (key.includes("jobs")) {
            internalJobs[key] = await import(value)
        }
        // If filename contains "nodes" then store in internalNodes
        if (key.includes("nodes")) {
            internalNodes[key] = await import(value)
        }
        else {
            // Filename does not match any of the above - and list filename
            console.log("Filename does not match jobs or nodes:", key)
        }
    }
}

export function getJobs(key: string = "") : Job[] {
    if (!internalJobs[key]) {
        key = Object.keys(internalJobs)[1]
    }
    return internalJobs[key]
}
export function getNodes(key: string = "") : Node[] {
    if (!internalNodes[key]) {
        key = Object.keys(internalNodes)[1]
    }
    return internalNodes[key]
}

export const jobsResponse = http.get('http://localhost:1234/api/v1/orchestrator/jobs', ({ request }) => {
    console.log("jobsResponse request", request)
    // return number of jobs based on limit
    const limitedJobs = getJobs().slice(0, JOBS_RETURN_LIMIT)
    const jobsListResponse: JobsResponse = { Jobs: limitedJobs, NextToken: "" }
    return HttpResponse.json(jobsListResponse)
})

export const nodesResponse = http.get('http://localhost:1234/api/v1/orchestrator/nodes', ({ request }) => {
    // return number of nodes based on limit
    const limitedNodes = getNodes().slice(0, NODES_RETURN_LIMIT)
    const nodesListResponse: NodesResponse = { Nodes: limitedNodes, NextToken: "" }
    return HttpResponse.json(nodesListResponse)
})

export const handlers: RequestHandler<any, any, any, RequestHandlerOptions>[] = [jobsResponse, nodesResponse]
