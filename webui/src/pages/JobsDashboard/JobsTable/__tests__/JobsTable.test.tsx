import React from "react"
import { MemoryRouter } from "react-router-dom"
import { screen, render, waitFor, act } from "@testing-library/react"
import { JobsTable } from "../JobsTable"
import { Job } from "../../../../helpers/jobInterfaces"
import { server } from "../../../../../tests/msw/server"
import { generateSampleJob } from "../../../../../tests/mocks/jobMock"

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
    mockJobs.push(generateSampleJob())
  }

  act(() => {
    render(
      <MemoryRouter>
        <JobsTable data={mockJobs} />
      </MemoryRouter>
    )
  })

  await waitFor(() => {
    screen
      .findByDisplayValue(`/${mockJobs[0].Name}/i`)
      .then((contentRendered) => {
        // Test to see if the content is in the document
        expect(contentRendered).toBeInTheDocument()
      })
  })
}
