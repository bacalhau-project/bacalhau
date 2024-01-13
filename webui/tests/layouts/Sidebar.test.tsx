import React from "react"
import { MemoryRouter } from "react-router-dom"
import { render } from "@testing-library/react"
import { Sidebar } from "../../src/layout/Sidebar/Sidebar"

const toggleSidebar = () => {
  // TODO: Implement toggle sidebar
  console.log("TODO: #3239 Implement toggle sidebar")
  // document.body.classList.toggle("sidebar-collapsed")
}

describe("Sidebar", () => {
  render(
    <MemoryRouter>
      <Sidebar isCollapsed={false} toggleSidebar={toggleSidebar} />
    </MemoryRouter>
  )
})
