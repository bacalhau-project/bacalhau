// Disable eslint for this file
/* eslint-disable */
// jest-dom adds custom jest matchers for asserting on DOM nodes.
// allows you to do things like:
// expect(element).toHaveTextContent(/react/i)
// learn more: https://github.com/testing-library/jest-dom
import "@testing-library/jest-dom"
import "@testing-library/react"

import { server as mswServer } from "../tests/msw/server"

// Establish API mocking before all tests.
beforeAll(() => {
  // Very annoying error message that is not relevant to our tests
  //  Warning: The current testing environment is not configured to support act(...)
  // Disabled with the following code according to this - https://stackoverflow.com/questions/72003409/the-current-testing-environment-is-not-configured-to-support-act-testing
  const originalConsoleError = console.error
  console.error = (...args) => {
    const firstArg = args[0]
    if (
      typeof args[0] === "string" &&
      (args[0].startsWith(
        "Warning: It looks like you're using the wrong act()"
      ) ||
        firstArg.startsWith(
          "Warning: The current testing environment is not configured to support act"
        ) ||
        firstArg.startsWith(
          "Warning: You seem to have overlapping act() calls"
        ))
    ) {
      return
    }
    originalConsoleError.apply(console, args)
  }

  mswServer.listen()
})

// Reset any request handlers that we may add during the tests,
// so they don't affect other tests.
afterEach(() => mswServer.resetHandlers())

// Clean up after the tests are finished.
afterAll(() => mswServer.close())
