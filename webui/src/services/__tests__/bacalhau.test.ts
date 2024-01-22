/* eslint-disable @typescript-eslint/no-unsafe-argument */
import { generateSampleJob } from "../../../tests/mocks/jobMock"
import { jobsResponse, setJobs } from "../../../tests/msw/handlers"
import { server as mswServer } from "../../../tests/msw/server"
import { Job } from "../../helpers/jobInterfaces"
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
  it("should GET /orchestrator/jobs with no data", async () => {
    mswServer.use(jobsResponse)
    const jobs = await bacalhauAPI.listJobs()
    expect(jobs.Jobs).toHaveLength(0)
  })
  it("should GET /orchestrator/jobs with two jobs", async () => {
    const mockJobList: Job[] = [generateSampleJob(), generateSampleJob()]
    setJobs(mockJobList)

    mswServer.use(jobsResponse)
    mswServer.listHandlers() // on printing handlers, I see the sampleUrl printed
    const returnJobs = await bacalhauAPI.listJobs(["returnData"])
    expect(returnJobs.Jobs).toHaveLength(2)
    expect(returnJobs.Jobs).toEqual(mockJobList)
  })
})
