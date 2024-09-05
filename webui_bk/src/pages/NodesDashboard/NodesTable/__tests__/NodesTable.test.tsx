import React from "react"
import { MemoryRouter } from "react-router-dom"
import { screen, render, waitFor, act } from "@testing-library/react"
import { NodesTable } from "../NodesTable"
import { Node } from "../../../../helpers/nodeInterfaces"
import { server } from "../../../../../tests/msw/server"
import { generateMockNode } from "../../../../../tests/mocks/nodeMock"

// Enable request interception.
beforeAll(() => server.listen())

// Reset handlers so that each test could alter them
// without affecting other, unrelated tests.
afterEach(() => server.resetHandlers())

// Don't forget to clean up afterwards.
afterAll(() => server.close())

describe("NodesTable", () => {
  beforeEach(() => {
    jest.clearAllMocks()
  })

  describe("Unit test that renders", () => {
    server.use()
    it("with 01 node", async () => {
      await renderWithNumberOfNodes(1)
    })
    it("with 10 node", async () => {
      await renderWithNumberOfNodes(10)
    })
    it("with 11 node", async () => {
      await renderWithNumberOfNodes(11)
    })
  })
})

async function renderWithNumberOfNodes(numberOfNodes: number) {
  const mockNodes: Node[] = []
  for (let i = 0; i < numberOfNodes; i += 1) {
    mockNodes.push(generateMockNode())
  }

  act(() => {
    render(
      <MemoryRouter>
        <NodesTable data={mockNodes} />
      </MemoryRouter>
    )
  })

  await waitFor(() => {
    screen
      .findByDisplayValue(`/${mockNodes[0].NodeID}/i`)
      .then((contentRendered) => {
        // Test to see if the content is in the document
        expect(contentRendered).toBeInTheDocument()
      })
  })
}
