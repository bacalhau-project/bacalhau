import { generateMockNode } from "../../tests/mocks/nodeMock"
import { Node } from "../helpers/nodeInterfaces"
import { NodesTable } from "../pages/NodesDashboard/NodesTable/NodesTable"

export default {
  component: NodesTable,
  title: "NodesTable",
  tags: ["autodocs"],
}

export const fullDataGenerator = (numberToGenerate: number): Node[] => {
  const nodes: Node[] = []
  for (let i = 0; i < numberToGenerate; i += 1) {
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
