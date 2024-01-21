/* eslint-disable @typescript-eslint/no-unsafe-argument */
import React from "react"
import { render, screen } from "@testing-library/react"
import { server } from "../mocks/msw/server"
import { testDataResponse } from "../mocks/msw/handlers"

// This is a simple file to test to make sure the configuration of msw is working
// properly. All components, types, and methods are self contained here.

// Enable request interception.
beforeAll(() => server.listen())

// Reset handlers so that each test could alter them
// without affecting other, unrelated tests.
afterEach(() => server.resetHandlers())

// Don't forget to clean up afterwards.
afterAll(() => server.close())

export type TestData = {
  userId: number
  id: number
  date: Date
  bool: boolean
}

type TestDataItemProps = {
  testData: TestData
}

function TestDataItem({ testData }: TestDataItemProps) {
  return (
    <pre>
      {`${testData.userId},${testData.id},${testData.date.toString()},${testData.bool}`}{" "}
    </pre>
  )
}

type TestDataListProps = {
  testDataArray: TestData[]
}

export default function TestDataList({ testDataArray }: TestDataListProps) {
  let content
  if (testDataArray.length === 0) {
    content = <p>No TestData</p>
  } else {
    content = (
      <>
        {testDataArray.map((testData) => (
          <TestDataItem key={testData.id} testData={testData} />
        ))}
      </>
    )
  }

  return content
}

async function fetchTestData() {
  try {
    const res = await fetch("/sampleQuery")

    const testData: TestData[] = (await res.json()) as TestData[]

    return testData
  } catch (err) {
    if (err instanceof Error) console.log(err.message)
    return []
  }
}

function MSWTestComponent(): JSX.Element {
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const [testData, setTestData] = React.useState<TestData[]>([])

  React.useEffect(() => {
    async function getTestData() {
      const testDataArray = await fetchTestData()
      if (testDataArray?.length) setTestData(testDataArray)
    }

    getTestData()
  }, [])

  return <TestDataList testDataArray={testData} setTestData={setTestData} />
}

describe("Basic tests of mocked API", () => {
  it("should GET React component backed by /sampleQuery with no test data", async () => {
    render(<MSWTestComponent />)
    // Query the sampleQuery endpoint and get a response.
    // Print the response to the console.
    expect(await screen.findByText("No TestData"))
    screen.debug()

    // expect(respData.container).toEqual({ data: { hello: "world" } })
  })
  it("should GET React component backed by /sampleQuery with two entries", async () => {
    server.use(testDataResponse)
    render(<MSWTestComponent />)
    // Query the sampleQuery endpoint and get a response.
    // Print the response to the console.
    expect(await screen.findByText("No TestData"))
    screen.debug()

    // expect(respData.container).toEqual({ data: { hello: "world" } })
  })
})
