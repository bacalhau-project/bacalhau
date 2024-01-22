import React from "react"
import { render, fireEvent, screen, act, waitFor } from "@testing-library/react"
import { MemoryRouter, Routes, Route } from "react-router-dom"
import { ActionButton } from "../ActionButton"

describe("ActionButton", () => {
  test("renders button with provided text", async () => {
    // Generate random string for button text
    const buttonText = Math.random().toString(36).substring(7)

    await act(async () => {
      render(
        <MemoryRouter>
          <ActionButton text={buttonText} />
        </MemoryRouter>
      )
    })

    // Test to see if the content is in the document
    await waitFor(() => {
      screen.findByDisplayValue(`/${buttonText}/i`).then((contentRendered) => {
        expect(contentRendered).toBeInTheDocument()
      })
    })
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
