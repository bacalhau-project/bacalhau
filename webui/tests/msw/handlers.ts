/* tslint:disable */
/* eslint-disable */
import { http, HttpResponse, RequestHandler, RequestHandlerOptions } from "msw";
import { Job, JobsResponse } from "../../src/helpers/jobInterfaces";
import { Node, NodesResponse } from "../../src/helpers/nodeInterfaces";
import { RETURN_DATA_PARAMETER, TestData } from "./__tests__/msw.test";

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

export function setJobs(jobs: Job[]) {
  internalJobs = jobs
}

let internalNodes: Node[] = []

export function setNodes(nodes: Node[]) {
  internalNodes = nodes
}

export const jobsResponse = http.get('http://localhost:1234/api/v1/orchestrator/jobs', ({ request }) => {
  let jobsListResponse: JobsResponse = { Jobs: internalJobs, NextToken: "" }
  return HttpResponse.json(jobsListResponse)
})

export const nodesResponse = http.get('http://localhost:1234/api/v1/orchestrator/nodes', ({ request }) => {
  let nodesListResponse: NodesResponse = { Nodes: internalNodes, NextToken: "" }
  return HttpResponse.json(nodesListResponse)
})


export const rootResponse = http.get('http://localhost:1234/', ({ cookies }) => {
  // Placeholders for messing around with cookies
  const { v } = cookies

  return HttpResponse.json(v === 'a' ? { foo: 'a' } : { bar: 'b' })
})

export const handlers: RequestHandler<any, any, any, RequestHandlerOptions>[] = [testDataResponse, rootResponse, jobsResponse, nodesResponse]

// export const sampResp = http.get<never, RequestBody, { foo: 'a' } | { bar: 'b' }>('/', resolver)

// export const fetchTasksEmptyResponse: HttpResponseResolver = async (_req: MockedRequest, res: ResponseComposition, ctx: Context) => await res(ctx.status(200), ctx.json([]))

// export const saveTasksEmptyResponse: HttpResponseResolver = async (_req: http.MockedRequest, res: http.ResponseComposition, ctx: http.Context) => await res(ctx.status(200), ctx.json([]))

// export const handlers = [
//   fetchTasksEmptyResponse,
//   saveTasks_empty_response,
// ]
// export const loadOneJob = http.get(BASE_URL, async (req, res, ctx) =>
//   res(ctx.status(200), ctx.json([]))
// )

// export const handlers = [
//   http.get("http://localhost:1234/api/v1/*", () => passthrough()),
// ]
