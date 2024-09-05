import React from "react"
import { act } from 'react';
import { MemoryRouter } from "react-router-dom"
import { screen, render, waitFor } from "@testing-library/react"
import { JobsDashboard } from "../JobsDashboard"
import { Job } from "../../../helpers/jobInterfaces"
import { server } from "../../../../tests/msw/server"
import { setJobs, JOBS_RETURN_LIMIT } from "../../../../tests/msw/handlers"
import { generateMockJob } from "../../../../tests/mocks/jobMock"

describe("JobsDashboard", () => {
  it("renders with right title", async () => {
    // Create a random string for pageTitle
    const pageTitle = Math.random().toString(36).substring(7)
    await act(async () => {
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
    it("with 01 job", async () => {
      await renderWithNumberOfJobs(1)
    })
    it("with JOBS_RETURN_LIMIT - 1 jobs", async () => {
      await renderWithNumberOfJobs(JOBS_RETURN_LIMIT - 1)
    })
    it("with JOBS_RETURN_LIMIT jobs", async () => {
      await renderWithNumberOfJobs(JOBS_RETURN_LIMIT)
    })
    it("with JOBS_RETURN_LIMIT + 1 jobs", async () => {
      await renderWithNumberOfJobs(JOBS_RETURN_LIMIT + 1)
    })
  })
})

async function renderWithNumberOfJobs(numberOfJobs: number) {
  const mockJobs: Job[] = []
  for (let i = 1; i <= numberOfJobs; i += 1) {
    const job = generateMockJob()
    mockJobs.push(job)
  }

  setJobs(mockJobs)

  act(() => {
    render(
      <MemoryRouter>
        <JobsDashboard />
      </MemoryRouter>
    )
  })

  await waitFor(async () => {
    // Wait for the element with the test ID 'jobsTableContainer' to be present
    const jobsTableContainer = await screen.findByTestId("jobsTableContainer")

    // Now you can check the content of the 'jobsTableContainer'
    expect(jobsTableContainer).toHaveTextContent("Job")
  })

  const firstJobName = mockJobs[0].Name
  const c1 = screen.getAllByText(firstJobName)[0]
  expect(c1.innerHTML).toContain(firstJobName)

  // Last job to be displayed is 10th job, or length of mockJobs, whatever is smaller
  const lastJobIndex = Math.min(JOBS_RETURN_LIMIT, mockJobs.length)

  // Ensure the correct number of jobs are displayed
  const numberOfJobRows = await screen.findAllByTestId("jobRow")
  expect(numberOfJobRows.length).toEqual(lastJobIndex)

  // Test to see if the last job is in the document
  const lastJobName = mockJobs[lastJobIndex - 1].Name
  const c2 = screen.getAllByText(lastJobName)[0]
  expect(c2.innerHTML).toContain(lastJobName)

  // Test to ensure tests are working
  const BAD_JOB_NAME = "BAD JOB NAME"
  const badPromise = waitFor(() => {
    const c = screen.getByText(BAD_JOB_NAME)
    expect(c).toContain(BAD_JOB_NAME)
  })
  expect(badPromise).rejects.toThrow()
}
