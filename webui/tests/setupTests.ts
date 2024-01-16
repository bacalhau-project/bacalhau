// jest-dom adds custom jest matchers for asserting on DOM nodes.
// allows you to do things like:
// expect(element).toHaveTextContent(/react/i)
// learn more: https://github.com/testing-library/jest-dom
import "@testing-library/jest-dom"

import { server as mswServer } from "./mocks/msw/server"
// Establish API mocking before all tests.
beforeAll(() => mswServer.listen())

// Reset any request handlers that we may add during the tests,
// so they don't affect other tests.
afterEach(() => mswServer.resetHandlers())

// Clean up after the tests are finished.
afterAll(() => mswServer.close())
