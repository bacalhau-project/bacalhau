// Exclude from coverage reports, and linting because this is a script
/* istanbul ignore file */
/* eslint-disable */
import { fullDataGenerator } from "../NodesTable.stories"

const path = require("node:path")
const fs = require("node:fs")

// Take as parameter the filename to write to, and the array of objects to write
export const writeData = (filename: string, data: any[]) => {
  const fullPath = path.resolve(__dirname, filename)
  const dataString = JSON.stringify(data, null, 2)
  console.log(`Writing to ${fullPath}`)
  console.log(dataString)
  fs.writeFileSync(fullPath, dataString)
}

// Paste (or import) the function to output below

// Write the jobs to a file using the writeData function from writeData.ts
const filename = "100-nodes.json"
writeData(filename, fullDataGenerator(100))
