/* tslint:disable */
/* eslint-disable */
import { http, HttpResponse, RequestHandler, RequestHandlerOptions } from "msw";
import { Job, JobsResponse } from "../../src/helpers/jobInterfaces";
import { RETURN_DATA_PARAMETER, TestData } from "../../src/helpers/mswInterfaces";
import { Node, NodesResponse } from "../../src/helpers/nodeInterfaces";

export const JOBS_RETURN_LIMIT = 10
export const NODES_RETURN_LIMIT = 10
const BASE_URL = "https://localhost:1234"

export const mockTestDataArray: TestData[] = [
  {
    "userId": 1234,
    "id": 1,
    "date": new Date("1970-01-01"),
    "bool": true
  },
  {
    "userId": 9876,
    "id": 2,
    "date": new Date("2023-12-31"),
    "bool": false
  },
]

// This does not have a route in the production app - it's for testing that tests are working
export const testDataResponse = http.get('/testData', ({ request }) => {
  const url = new URL(request.url)

  let returnDataArray: TestData[] = []
  if (url.searchParams.get(RETURN_DATA_PARAMETER)) {
    returnDataArray = mockTestDataArray;
  }

  return HttpResponse.json(returnDataArray);

})

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

// TODO: #3304 Implement testing for pagination

// TODO: #3303 Build a go generator that generates constants / types from the Go code - examples: Job, Node type, default limits, etc.
export const jobsResponse = http.get('http://localhost:1234/api/v1/orchestrator/jobs', ({ request }) => {
  // return number of jobs based on limit
  const limitedJobs = internalJobs.slice(0, JOBS_RETURN_LIMIT)
  const jobsListResponse: JobsResponse = { Jobs: limitedJobs, NextToken: "" }
  return HttpResponse.json(jobsListResponse)
})

export const nodesResponse = http.get('http://localhost:1234/api/v1/orchestrator/nodes', ({ request }) => {
  // return number of nodes based on limit
  const limitedNodes = internalNodes.slice(0, NODES_RETURN_LIMIT)
  const nodesListResponse: NodesResponse = { Nodes: limitedNodes, NextToken: "" }
  return HttpResponse.json(nodesListResponse)
})


export const rootResponse = http.get('http://localhost:1234/', ({ cookies }) => {
  // Placeholders for messing around with cookies
  const { v } = cookies

  return HttpResponse.json(v === 'a' ? { foo: 'a' } : { bar: 'b' })
})

export const handlers: RequestHandler<any, any, any, RequestHandlerOptions>[] = [testDataResponse, rootResponse, jobsResponse, nodesResponse]
