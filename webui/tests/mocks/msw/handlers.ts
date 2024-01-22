import { http, HttpResponse, passthrough } from "msw"

export const handlers = [
  http.get("http://localhost:1234/api/v1/orchestrator/jobs", () =>
    HttpResponse.json({}, { status: 200 })
  ),
  http.get("http://localhost:1234/api/v1/*", () => passthrough()),
]
