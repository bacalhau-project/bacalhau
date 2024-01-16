/* eslint-disable @typescript-eslint/no-unsafe-argument */
import React from "react"
import { render } from "@testing-library/react"
import { server } from "../mocks/msw/server"

// Enable request interception.
beforeAll(() => server.listen())

// Reset handlers so that each test could alter them
// without affecting other, unrelated tests.
afterEach(() => server.resetHandlers())

// Don't forget to clean up afterwards.
afterAll(() => server.close())

function MSWTestComponent() {
  const [data, setData] = React.useState(null)

  React.useEffect(() => {
    fetch("http://localhost:1234/sampleQuery")
      .then((response) => response.json())
      .then((responseData) => setData(responseData))
  }, [])

  return data
}

test("should GET /sampleQuery", () => {
  const respData = render(<MSWTestComponent />)
  // Query the sampleQuery endpoint and get a response.
  // Print the response to the console.
  // console.log(respData)

  // expect(respData.container).toEqual({ data: { hello: "world" } })
})
