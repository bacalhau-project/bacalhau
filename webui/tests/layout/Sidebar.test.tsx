// @ts-nocheck
import React from "react"
import { render, fireEvent, screen } from "@testing-library/react"
import { MemoryRouter, Route } from "react-router-dom"
import Sidebar from "../../src/layout/Sidebar/Sidebar"

describe("Sidebar", () => {
  const setup = (isCollapsed = false) => {
    const toggleSidebar = jest.fn()
    render(
      <MemoryRouter initialEntries={["/"]}>
        <Sidebar isCollapsed={isCollapsed} toggleSidebar={toggleSidebar} />
        <Route path="/:page" children={<div />} />
      </MemoryRouter>
    )
    return { toggleSidebar }
  }

  it("renders correctly", () => {
    setup()
    expect(screen.getByTitle("Jobs Dashboard")).toBeInTheDocument()
    expect(screen.getByTitle("Nodes Dashboard")).toBeInTheDocument()
    expect(screen.getByTitle("Settings")).toBeInTheDocument()
  })

  it("toggles sidebar on button click", () => {
    const { toggleSidebar } = setup()
    fireEvent.click(screen.getByRole("button"))
    expect(toggleSidebar).toHaveBeenCalledTimes(1)
  })

  it("displays links with correct paths and titles", () => {
    setup()
    expect(screen.getByTitle("Jobs Dashboard").getAttribute("href")).toBe(
      "/JobsDashboard"
    )
    expect(screen.getByTitle("Nodes Dashboard").getAttribute("href")).toBe(
      "/NodesDashboard"
    )
    expect(screen.getByTitle("Settings").getAttribute("href")).toBe("/Settings")
  })

  it("indicates the selected link based on the current path", () => {
    render(
      <MemoryRouter initialEntries={["/JobsDashboard"]}>
        <Sidebar isCollapsed={false} toggleSidebar={() => {}} />
      </MemoryRouter>
    )
    expect(screen.getByTitle("Jobs Dashboard")).toHaveAttribute(
      "data-selected",
      "true"
    )
  })
})
