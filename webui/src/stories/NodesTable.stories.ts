import { generateMockNode } from "../../tests/mocks/nodeMock"
import { Node } from "../helpers/nodeInterfaces"
import { NodesTable } from "../pages/NodesDashboard/NodesTable/NodesTable"

export default {
    component: NodesTable,
    title: "NodesTable",
    tags: ["autodocs"],
}

const fullData = (): Node[] => {
    // Create a list of 10 jobs
    const nodes: Node[] = []
    for (let i = 0; i < 10; i += 1) {
        nodes.push(generateMockNode())
    }
    return nodes
}

export const Default = {
    args: { data: [] },
}

export const FullData = {
    args: { data: fullData() },
}
