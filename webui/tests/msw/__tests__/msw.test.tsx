/* eslint-disable @typescript-eslint/no-unsafe-argument */
import { render, screen, act, waitFor } from "@testing-library/react"
import { useState, useEffect } from "react"
import { server } from "../server"
import { mockTestDataArray } from "../handlers"

// This is a simple file to test to make sure the configuration of msw is working
// properly. All components, types, and methods are self contained here.

export const RETURN_DATA_PARAMETER = "returnData"
export const TEST_DATA_ID = "testDataId"

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

async function fetchTestData(returnData: boolean = false) {
  try {
    const appendString = returnData ? `?${RETURN_DATA_PARAMETER}=true` : ""
    const res = await fetch(`http://localhost/testData${appendString}`)

    const testData: TestData[] = (await res.json()) as TestData[]

    return testData
  } catch (err) {
    if (err instanceof Error) console.log(err.message)
    return []
  }
}

type MSWProps = {
  ShouldReturnData?: boolean
}

export const MSWTestComponent: React.FC<MSWProps> = ({
  ShouldReturnData = false,
}) => {
  const [testDataArray, setTestDataArray] = useState<TestData[]>([])
  const [loadingData, setLoadingData] = useState<boolean>(true)

  useEffect(() => {
    async function fetchTestDataAsync() {
      const fetchedTestDataArray = await fetchTestData(ShouldReturnData)
      setTestDataArray(fetchedTestDataArray)
      setLoadingData(false)
    }
    fetchTestDataAsync()
  }, [ShouldReturnData])

  return (
    <div className="App">
      {loadingData ? <div>loading...</div> : renderTestData(testDataArray)}
    </div>
  )
}

function renderTestData(testDataArray: TestData[]) {
  return testDataArray.length === 0 ? (
    <p>No TestData</p>
  ) : (
    <ul data-testid={TEST_DATA_ID}>
      {testDataArray.map((testDataItem, i) => (
        <li key={i}>{convertTestItemToText(testDataItem)}</li>
      ))}
    </ul>
  )
}

describe("Basic tests of mocked API", () => {
  it("should GET React component backed by /testData with no test data", async () => {
    act(() => {
      render(<MSWTestComponent />)
    })

    await waitFor(() => {
      // Query the sampleQuery endpoint and get a response.
      // Print the response to the console.
      expect(screen.getByText("No TestData")).toBeInTheDocument()
    })
  })
  it("should GET React component backed by /testData with two entries", async () => {
    render(<MSWTestComponent ShouldReturnData />)
    const loading = await screen.findByText("loading...")
    expect(loading).toBeInTheDocument()

    await waitFor(async () => {
      expect(loading).not.toBeInTheDocument()

      const listNode = await screen.findByTestId(TEST_DATA_ID)

      expect(listNode.childNodes.length).toBe(2)

      for (let i = 0; i < listNode.childNodes.length; i += 1) {
        expect(listNode.childNodes[i]).toHaveTextContent(
          convertTestItemToText(mockTestDataArray[i])
        )
      }
    })
  })
})

function convertTestItemToText(testItem: TestData) {
  const dateString = new Date(testItem.date)
  return `${testItem.userId} ${testItem.id} ${dateString.toISOString()} ${testItem.bool}`
}
