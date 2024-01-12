import React from "react"
import { MemoryRouter } from "react-router-dom"
import { screen, render } from "@testing-library/react"
import { JobsDashboard } from "../../src/pages/JobsDashboard/JobsDashboard"

describe("JobsDashboard", () => {
  beforeEach(() => {
    jest.clearAllMocks()
  })

  test("renders JobsDashboard", () => {
    render(
      <MemoryRouter>
        <JobsDashboard />
      </MemoryRouter>
    )

    expect(screen.getAllByText(/Jobs Dashboard/i).length).toBeGreaterThan(0)
  })
})
