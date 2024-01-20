/* eslint-disable @typescript-eslint/no-unsafe-argument */
import { jobsDashboardResponse } from "../../../tests/mocks/msw/handlers"
import { server as mswServer } from "../../../tests/mocks/msw/server"
import { bacalhauAPI } from "../bacalhau"

// This is a simple file to test to make sure the configuration of msw is working
// properly. All components, types, and methods are self contained here.

// Enable request interception.
beforeAll(() => mswServer.listen())

// Reset handlers so that each test could alter them
// without affecting other, unrelated tests.
afterEach(() => mswServer.resetHandlers())

// Don't forget to clean up afterwards.
afterAll(() => mswServer.close())

describe("Basic fetch of mocked API", () => {
    it("should GET /orchestrator/jobs", async () => {
        const jobsResponse = { a: 2 }

        mswServer.use(jobsDashboardResponse)
        mswServer.listHandlers() // on printing handlers, I see the sampleUrl printed

        const jobs = await bacalhauAPI.listJobs()

        expect(jobs).toEqual(jobsResponse)
    })
})
