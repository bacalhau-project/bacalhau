import React from "react"
import { render } from "@testing-library/react"
import { MemoryRouter } from "react-router-dom"
import { Sidebar } from "../../src/layout/Sidebar/Sidebar"

const toggleSidebar = () => {
  // TODO: Implement toggle sidebar
  console.log("TODO: #3239 Implement toggle sidebar")
  // document.body.classList.toggle("sidebar-collapsed")
}

describe("Sidebar", () => {
  test("renders", () => {
    render(
      <MemoryRouter>
        <Sidebar isCollapsed={false} toggleSidebar={toggleSidebar} />
      </MemoryRouter>
    )
  })
})
