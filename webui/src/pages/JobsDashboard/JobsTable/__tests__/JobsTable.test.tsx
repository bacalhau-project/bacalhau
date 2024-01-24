import React from "react"
import { MemoryRouter } from "react-router-dom"
import { screen, render, waitFor, act } from "@testing-library/react"
import { JobsTable } from "../JobsTable"
import { Job } from "../../../../helpers/jobInterfaces"
import { server } from "../../../../../tests/msw/server"
import { generateMockJob } from "../../../../../tests/mocks/jobMock"

// Enable request interception.
beforeAll(() => server.listen())

// Reset handlers so that each test could alter them
// without affecting other, unrelated tests.
afterEach(() => server.resetHandlers())

// Don't forget to clean up afterwards.
afterAll(() => server.close())

describe("JobsTable", () => {
  beforeEach(() => {
    jest.clearAllMocks()
  })

  describe("Unit test that renders", () => {
    server.use()
    it("with 01 job", async () => {
      await renderWithNumberOfJobs(1)
    })
    it("with 10 jobs", async () => {
      await renderWithNumberOfJobs(10)
    })
    it("with 11 jobs", async () => {
      await renderWithNumberOfJobs(11)
    })
  })
})

async function renderWithNumberOfJobs(numberOfJobs: number) {
  const mockJobs: Job[] = []
  for (let i = 0; i < numberOfJobs; i += 1) {
    mockJobs.push(generateMockJob())
  }

  act(() => {
    render(
      <MemoryRouter>
        <JobsTable data={mockJobs} />
      </MemoryRouter>
    )
  })

  await waitFor(() => {
    const firstJobName = mockJobs[0].Name
    const c = screen.getByText(firstJobName)
    expect(c.innerHTML).toContain(firstJobName)
  })

  // Last job to be displayed is 10th job, or length of mockJobs, whatever is smaller
  const lastJobIndex = Math.min(10, mockJobs.length - 1)

  // Test to see if the last job is in the document
  await waitFor(() => {
    const lastJobName = mockJobs[lastJobIndex].Name
    const c = screen.getByText(lastJobName)
    expect(c.innerHTML).toContain(lastJobName)
  })

  // Test to ensure tests are working
  const BAD_JOB_NAME = "BAD JOB NAME"
  const badPromise = waitFor(() => {
    const c = screen.getByText(BAD_JOB_NAME)
    expect(c).toContain(BAD_JOB_NAME)
  })
  expect(badPromise).rejects.toThrow()
}
