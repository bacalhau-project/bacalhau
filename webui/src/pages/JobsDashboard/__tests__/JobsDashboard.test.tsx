import React from "react"
import { MemoryRouter } from "react-router-dom"
import { screen, render, waitFor, act } from "@testing-library/react"
import { JobsDashboard } from "../JobsDashboard"
import { Job } from "../../../helpers/jobInterfaces"
import { server } from "../../../../tests/msw/server"
import { setJobs } from "../../../../tests/msw/handlers"
import { generateSampleJob } from "../../../../tests/mocks/jobMock"

describe("JobsDashboard", () => {
  beforeEach(() => {
    jest.clearAllMocks()
  })
  it("renders with right title", () => {
    // Create a random string for pageTitle
    const pageTitle = Math.random().toString(36).substring(7)
    act(() => {
      render(
        <MemoryRouter>
          <JobsDashboard pageTitle={pageTitle} />
        </MemoryRouter>
      )
    })

    expect(screen.getByRole("heading", { level: 1 }).innerHTML).toContain(
      pageTitle
    )
  })
  describe("integration test that renders", () => {
    beforeEach(() => {
      server.resetHandlers()
    })
    it("with one job", () => {
      act(async () => {
        await renderWithNumberOfJobs(1)
      })
    })
    it("with multiple jobs", () => {
      act(async () => {
        await renderWithNumberOfJobs(10)
      })
    })
    it("with 11+ jobs", () => {
      act(async () => {
        await renderWithNumberOfJobs(11)
      })
    })
  })
})

async function renderWithNumberOfJobs(numberOfJobs: number) {
  const mockJobs: Job[] = []

  act(() => {
    for (let i = 0; i < numberOfJobs; i += 1) {
      mockJobs.push(generateSampleJob())
    }

    setJobs(mockJobs)
    render(
      <MemoryRouter>
        <JobsDashboard />
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

  // Last job to be displayed is 10th job, or length of mockJobs, whatever is smaller
  const lastJobIndex = Math.min(10, mockJobs.length - 1)

  // Test to see if the last job is in the document
  screen
    .findByDisplayValue(`/${mockJobs[lastJobIndex].Name}/i`)
    .then((contentRendered) => {
      // Test to see if the content is in the document
      expect(contentRendered).toBeInTheDocument()
    })
}
