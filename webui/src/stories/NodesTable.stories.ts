import { generateMockNode } from "../../tests/mocks/nodeMock"
import { Node } from "../helpers/nodeInterfaces"
import { NodesTable } from "../pages/NodesDashboard/NodesTable/NodesTable"

export default {
    component: NodesTable,
    title: "NodesTable",
    tags: ["autodocs"],
}

export const fullDataGenerator = (numberToGenerate: number): Node[] => {
    // Create a list of 10 jobs
    const jobs: Node[] = []
    for (let i = 0; i < numberToGenerate; i += 1) {
        jobs.push(generateMockNode())
    }
    return jobs
}

export const Default = {
    args: { data: [] },
}

export const FullData = {
    args: { data: fullDataGenerator(10) },
}
