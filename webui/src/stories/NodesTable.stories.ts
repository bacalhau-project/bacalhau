import { generateMockNode } from "../../tests/mocks/nodeMock"
import { Node } from "../helpers/nodeInterfaces"
import { NodesTable } from "../pages/NodesDashboard/NodesTable/NodesTable"

export default {
  component: NodesTable,
  title: "NodesTable",
  tags: ["autodocs"],
}

export const fullDataGenerator = (numNodes: number = 10): Node[] => {
  // Create a list of 10 jobs
  const nodes: Node[] = []
  for (let i = 0; i < numNodes; i += 1) {
    nodes.push(generateMockNode())
  }
  return nodes
}

export const Default = {
  args: { data: [] },
}

export const FullData = {
    args: { data: fullDataGenerator(10) },
}
