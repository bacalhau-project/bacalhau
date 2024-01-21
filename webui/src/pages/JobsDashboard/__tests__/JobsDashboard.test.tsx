import React from "react"
import { MemoryRouter } from "react-router-dom"
import { screen, render } from "@testing-library/react"
import { JobsDashboard } from "../JobsDashboard"
import { Job } from "../../../helpers/jobInterfaces"
import { server } from "../../../../tests/mocks/msw/server"
import { setJobs } from "../../../../tests/mocks/msw/handlers"
import { generateSampleJob } from "../../../../tests/mocks/jobMock"

// Enable request interception.
beforeAll(() => server.listen())

// Reset handlers so that each test could alter them
// without affecting other, unrelated tests.
afterEach(() => server.resetHandlers())

// Don't forget to clean up afterwards.
afterAll(() => server.close())

describe("JobsDashboard", () => {
  beforeEach(() => {
    jest.clearAllMocks()
  })
  it("renders with right title", () => {
    // Create a random string for pageTitle
    const pageTitle = Math.random().toString(36).substring(7)
    render(
      <MemoryRouter>
        <JobsDashboard pageTitle={pageTitle} />
      </MemoryRouter>
    )
    console.log(screen.debug())
    expect(screen.getByRole('heading', {level: 1}).innerHTML).toContain(pageTitle)
  })
  it("renders with one job", () => {
    server.use()
    
    const mockJobs: Job[] = [generateSampleJob()]
    setJobs(mockJobs)

    render(
      <MemoryRouter>
        <JobsDashboard />
      </MemoryRouter>
    )
    
    console.log(screen.debug())

    expect(screen.getAllByText(/Job 1/i).length).toBeGreaterThan(0)
  })
})
