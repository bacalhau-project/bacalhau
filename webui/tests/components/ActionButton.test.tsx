import React from "react"
import { render, fireEvent, screen } from "@testing-library/react"
import { MemoryRouter, Routes, Route } from "react-router-dom"
import { ActionButton } from "../../src/components/ActionButton/ActionButton"

describe("ActionButton", () => {
  test("renders button with provided text", () => {
    render(
      <MemoryRouter>
        <ActionButton text="Test Button" />
      </MemoryRouter>
    )

    expect(screen.getByText("Test Button")).toBeInTheDocument()
  })

  test("calls onClick when provided and button is clicked", () => {
    const handleClick = jest.fn()

    render(
      <MemoryRouter>
        <ActionButton text="Test Button" onClick={handleClick} />
      </MemoryRouter>
    )

    fireEvent.click(screen.getByText("Test Button"))

    expect(handleClick).toHaveBeenCalled()
  })

  test("navigates to 'to' path when provided and button is clicked", () => {
    render(
      <MemoryRouter initialEntries={["/"]}>
        <Routes>
          <Route
            path="/"
            element={<ActionButton text="Test Button" to="/test-path" />}
          />
          <Route path="/test-path" element={<div>Test Page</div>} />
        </Routes>
      </MemoryRouter>
    )

    fireEvent.click(screen.getByText("Test Button"))

    // Check if the 'Test Page' content is rendered
    expect(screen.getByText("Test Page")).toBeInTheDocument()
  })
})
