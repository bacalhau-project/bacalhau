import { render, screen, act } from "@testing-library/react"
import { server as mswServer } from "../tests/msw/server"
import App from "./App"

// Enable request interception.
beforeAll(() => mswServer.listen())

// Reset handlers so that each test could alter them
// without affecting other, unrelated tests.
afterEach(() => mswServer.resetHandlers())

// Don't forget to clean up afterwards.
afterAll(() => mswServer.close())

describe("Root Page", () => {
  describe("Static tests", () => {
    it("should render home page", () => {
      const pageTitle = "Jobs Dashboard"
      act(() => {
        render(<App />)
      })

      expect(screen.getByRole("heading", { level: 1 }).innerHTML).toContain(
        pageTitle
      )
    })
  })
})
