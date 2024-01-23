import React from "react"
import { MemoryRouter } from "react-router-dom"
import { screen, render, waitFor, act } from "@testing-library/react"
import { NodesDashboard } from "../NodesDashboard"
import { Node } from "../../../helpers/nodeInterfaces"
import { server } from "../../../../tests/msw/server"
import { setNodes } from "../../../../tests/msw/handlers"
import { generateMockNode } from "../../../../tests/mocks/nodeMock"

describe("NodesDashboard", () => {
  beforeEach(() => {
    jest.clearAllMocks()
  })
  it("renders with right title", () => {
    // Create a random string for pageTitle
    const pageTitle = Math.random().toString(36).substring(7)
    act(() => {
      render(
        <MemoryRouter>
          <NodesDashboard pageTitle={pageTitle} />
        </MemoryRouter>
      )
    })

    expect(screen.getByRole("heading", { level: 1 }).innerHTML).toContain(
      pageTitle
    )
  })
  describe("integration test that renders", () => {
    beforeEach(() => {
      server.resetHandlers()
    })
    it("with one nodes", async () => {
      await act(async () => {
        await renderWithNumberOfNodes(1)
      })
    })
    it("with multiple nodes", async () => {
      await act(async () => {
        await renderWithNumberOfNodes(10)
      })
    })
    it("with 11+ nodes", async () => {
      await act(async () => {
        await renderWithNumberOfNodes(11)
      })
    })
  })
})

async function renderWithNumberOfNodes(numberOfNodes: number) {
  const mockNodes: Node[] = []

  act(() => {
    for (let i = 0; i < numberOfNodes; i += 1) {
      mockNodes.push(generateMockNode())
    }

    setNodes(mockNodes)
    render(
      <MemoryRouter>
        <NodesDashboard />
      </MemoryRouter>
    )
  })

  await waitFor(() => {
    screen
      .findByDisplayValue(`/${mockNodes[0].PeerInfo.ID}/i`)
      .then((contentRendered) => {
        // Test to see if the content is in the document
        expect(contentRendered).toBeInTheDocument()
      })
  })

  // Last job to be displayed is 10th job, or length of mockJobs, whatever is smaller
  const lastJobIndex = Math.min(10, mockNodes.length - 1)

  // Test to see if the last job is in the document
  screen
    .findByDisplayValue(`/${mockNodes[lastJobIndex].PeerInfo.ID}/i`)
    .then((contentRendered) => {
      // Test to see if the content is in the document
      expect(contentRendered).toBeInTheDocument()
    })
}
