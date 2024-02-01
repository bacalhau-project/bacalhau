import React from "react"
import { MemoryRouter } from "react-router-dom"
import { screen, render, waitFor, act } from "@testing-library/react"
import { NodesDashboard } from "../NodesDashboard"
import { Node } from "../../../helpers/nodeInterfaces"
import { server } from "../../../../tests/msw/server"
import { setNodes, NODES_RETURN_LIMIT } from "../../../../tests/msw/handlers"
import { generateMockNode } from "../../../../tests/mocks/nodeMock"

describe("NodesDashboard", () => {
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
    it("with NODES_RETURN_LIMIT - 1 jobs", async () => {
      await renderWithNumberOfNodes(NODES_RETURN_LIMIT - 1)
    })
    it("with NODES_RETURN_LIMIT jobs", async () => {
      await renderWithNumberOfNodes(NODES_RETURN_LIMIT)
    })
    it("with NODES_RETURN_LIMIT + 1 jobs", async () => {
      await renderWithNumberOfNodes(NODES_RETURN_LIMIT + 1)
    })
  })
})

async function renderWithNumberOfNodes(numberOfNodes: number) {
  const mockNodes: Node[] = []
  for (let i = 1; i <= numberOfNodes; i += 1) {
    const node = generateMockNode()
    mockNodes.push(node)
  }

  setNodes(mockNodes)

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
    // Wait for the element with the test ID 'nodesTableContainer' to be present
    const nodesTableContainer = screen.getAllByTestId("nodeRow")

    // Now you can check the content of the 'nodesTableContainer'
    expect(nodesTableContainer.length).toBeGreaterThan(0)
  })

  const firstNodeID = mockNodes[0].PeerInfo.ID
  const c1 = screen.getAllByText(firstNodeID)[0]
  expect(c1.innerHTML).toContain(firstNodeID)

  // Last node to be displayed is 10th node, or length of mockNode, whatever is smaller
  const lastNodeIndex = Math.min(10, mockNodes.length)

  // Count number of nodes displayed
  const nodesDisplayed = screen.getAllByTestId("nodeRow")
  expect(nodesDisplayed.length).toEqual(lastNodeIndex)

  // Test to see if the last node is in the document
  const lastNodeID = mockNodes[lastNodeIndex - 1].PeerInfo.ID
  const c2 = screen.getAllByText(lastNodeID)[0]
  expect(c2.innerHTML).toContain(lastNodeID)

  // Test to ensure tests are working
  const BAD_NODE_NAME = "BAD NODE NAME"
  const badPromise = waitFor(() => {
    const c = screen.getByText(BAD_NODE_NAME)
    expect(c).toContain(BAD_NODE_NAME)
  })
  expect(badPromise).rejects.toThrow()
}
