import React from "react"
import { render } from "@testing-library/react"
import { MemoryRouter } from "react-router-dom"
import { Sidebar, toggleSidebarFn } from "../../src/layout/Sidebar/Sidebar"

describe("Sidebar", () => {
  test("renders", () => {
    render(
      <MemoryRouter>
        <Sidebar isCollapsed={false} toggleSidebar={toggleSidebarFn} />
      </MemoryRouter>
    )
  })
})
