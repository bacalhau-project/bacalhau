/* tslint:disable */
/* eslint-disable */
import { http, HttpResponse } from "msw"

const BASE_URL = "https://localhost:1234/"

// export const fetchTasksIncompleteTaskResponse = http.get(
//   BASE_URL,
//   async (req, res, ctx) =>
//     res(
//       ctx.status(200),
//       ctx.json([
//         {
//           id: "1",
//           name: "Finish course",
//           createdOn: Date.now(),
//           status: TaskStatus.INCOMPLETE,
//         },
//       ])
//     ) as any
// )

export const sampleQueryResponse = http.get('http://localhost:1234/sampleQuery', ({ cookies }) => {
  // Placeholders for messing around with cookies
  const { v } = cookies

  return new HttpResponse('Hello world', { status: 201 })
})


export const rootResponse = http.get('/', ({ cookies }) => {
  // Placeholders for messing around with cookies
  const { v } = cookies

  return HttpResponse.json(v === 'a' ? { foo: 'a' } : { bar: 'b' })
})

export const jobsDashboardResponse = http.get('/api/v1/orchestrator/jobs', ({ cookies }) => {
  // Placeholders for messing around with cookies
  const { v } = cookies

  return HttpResponse.json(v === 'a' ? { foo: 'a' } : { bar: 'b' })
})

export const handlers = [sampleQueryResponse, rootResponse, jobsDashboardResponse]

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
