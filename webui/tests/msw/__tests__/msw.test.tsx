/* eslint-disable @typescript-eslint/no-unsafe-argument */
import React from "react"
import { render, screen, act, waitFor, queryByAttribute, RenderResult } from "@testing-library/react"
import { server } from "../server"
import { mockTestDataArray } from "../handlers"
import { useState, useEffect } from "react"
import crypto from "crypto"

// This is a simple file to test to make sure the configuration of msw is working
// properly. All components, types, and methods are self contained here.

export const RETURN_DATA_PARAMETER = "returnData"

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
  return (<>
    {`${testData.userId},${testData.id},${testData.date.toString()},${testData.bool}`}{" "}
    </>
  )
}

async function fetchTestData(returnData:boolean = false) {
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

export const MSWTestComponent: React.FC<MSWProps>  = ({
  ShouldReturnData = false,
}) => {
  const [testDataArray, setTestDataArray] = useState<TestData[]>([]);

  useEffect(() => {
    async function fetchTestDataAsync() {
      const fetchedTestDataArray = await fetchTestData(ShouldReturnData)
      setTestDataArray(fetchedTestDataArray)
    }
    fetchTestDataAsync();
  }, []);

  return (
    <div id="dataList">
      {testDataArray?.length ? (
        testDataArray.map((testDataItem) => (
          <div key={crypto.randomBytes(16).toString("hex")}>
            <TestDataItem testData={testDataItem} />
          </div>
        ))
      ) : (
        <p>No TestData</p>
      )}
    </div>
  )
}

describe("Basic tests of mocked API", () => {
  it("should GET React component backed by /testData with no test data", async () => {
    await act(() => {
      render(<MSWTestComponent />)
    })
    
    await waitFor(() => {
      // Query the sampleQuery endpoint and get a response.
      // Print the response to the console.
      expect(screen.getByText("No TestData")).toBeInTheDocument()
    })

  })
  it("should GET React component backed by /testData with two entries", async () => {
    let result: RenderResult;
    result = render(<MSWTestComponent ShouldReturnData={true} />);
    const dom = result.container;
    
    await waitFor(() => {
      // Get the expect for the div with id "dataList"
      const dataList = dom.querySelector("#dataList");
      
      expect(dataList).toBeInTheDocument();
      
      // Query the sampleQuery endpoint and get a response.
      // Print the response to the console.
      expect(dataList).toHaveTextContent(mockTestDataArray[0].id.toString())
      expect(dataList).toHaveTextContent(mockTestDataArray[1].id.toString())
    })

  })
})
