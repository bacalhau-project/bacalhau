import React from "react"
import { MemoryRouter } from "react-router-dom"
import { screen, render } from "@testing-library/react"
import { JobsDashboard } from "../../src/pages/JobsDashboard/JobsDashboard"
import { server } from "../mocks/msw/server"

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

  test("renders JobsDashboard", () => {
    it("renders with right title", () => {
      render(
        <MemoryRouter>
          <JobsDashboard />
        </MemoryRouter>
      )

      console.debug()

      expect(screen.getAllByText(/Jobs Dashboard/i).length).toBeGreaterThan(0)
    })
  })
  it("renders with one job", () => {
    server.use()

    render(
      <MemoryRouter>
        <JobsDashboard />
      </MemoryRouter>
    )

    expect(screen.getAllByText(/Job 1/i).length).toBeGreaterThan(0)
  })
})
