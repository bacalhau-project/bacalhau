import { render, screen } from "@testing-library/react"
import React from "react"
import { server as mswServer } from "./mocks/msw/server"
import App from "../src/App"

import { rootResponse } from "./mocks/msw/handlers"

describe("Root Page", () => {
  describe("Static tests", () => {
    it("should render home page", () => {
      mswServer.use(rootResponse)

      render(<App />)

      console.debug(screen.debug())

      // Should redirect to the jobs dashboard, so that's the page title
      expect(screen.getByText(/Jobs Dashboard/)).resolves.toBeInTheDocument()
    })
  })
})

//   describe("HTTP test", () => {
//     afterEach(() => jest.restoreAllMocks())

//     it("should fetch tasks from backend", async () => {
//       const spy = jest.spyOn(TaskClient, "fetchTasks").mockResolvedValue([])

//       await act(() => {
//         render(<App />)
//       })

//       expect(spy).toHaveBeenCalledTimes(1)
//     })

//     it("should render fetched tasks", async () => {
//       mswServer.use(fetchTasks_incompleteTask_response)

//       render(<App />)

//       expect(await screen.findByText(/Finish course/)).toBeInTheDocument()
//     })

//     it("should send http post on submit", async () => {
//       mswServer.use(fetchTasks_empty_response, saveTasks_empty_response)
//       const putSpy = jest.spyOn(TaskClient, "saveTasks")

//       const user = userEvent.setup()
//       render(<App />)

//       await user.type(
//         screen.getByPlaceholderText(/Add a task/),
//         "Finish course"
//       )
//       await user.click(screen.getByText(/Add/))

//       expect(
//         within(screen.getByText("Incomplete Tasks").parentElement!).getByText(
//           /Finish course/
//         )
//       ).toBeInTheDocument()

//       expect(putSpy).toHaveBeenNthCalledWith(1, [
//         expect.objectContaining({
//           id: expect.any(String),
//           name: "Finish course",
//           createdOn: expect.any(Number),
//           status: TaskStatus.INCOMPLETE,
//         }),
//       ])
//     })

//     it("should send http post on status change", async () => {
//       mswServer.use(
//         fetchTasks_incompleteTask_response,
//         saveTasks_empty_response
//       )
//       const putSpy = jest.spyOn(TaskClient, "saveTasks")

//       const user = userEvent.setup()
//       render(<App />)

//       expect(await screen.findByText("Finish course")).toBeInTheDocument()

//       await user.click(
//         within(screen.getByText("Incomplete Tasks").parentElement!).getByText(
//           /Finish course/
//         )
//       )
//       expect(putSpy).toHaveBeenNthCalledWith(1, [
//         expect.objectContaining({
//           id: "1",
//           name: "Finish course",
//           createdOn: expect.any(Number),
//           completedOn: expect.any(Number),
//           status: TaskStatus.COMPLETE,
//         }),
//       ])

//       await user.click(
//         within(screen.getByText("Completed Tasks").parentElement!).getByText(
//           /Finish course/
//         )
//       )
//       expect(putSpy).toHaveBeenNthCalledWith(2, [
//         expect.objectContaining({
//           id: expect.any(String),
//           name: expect.any(String),
//           createdOn: expect.any(Number),
//           status: TaskStatus.INCOMPLETE,
//         }),
//       ])
//     })
//   })

//   describe("Input Validation", () => {
//     let user: UserEvent
//     beforeEach(() => {
//       user = userEvent.setup()
//       render(<App />)
//     })

//     it("should show error when empty input is submitted", async () => {
//       await user.click(screen.getByText(/Add/))
//       expect(screen.getByText(/Invalid input/)).toBeInTheDocument()
//     })

//     it("should clear error message once valid input is submitted", async () => {
//       await user.click(screen.getByText(/Add/))
//       expect(screen.getByText(/Invalid input/)).toBeInTheDocument()

//       const input = screen.getByPlaceholderText(/Add a task/)
//       await user.type(input, "Finish course")
//       await user.click(screen.getByText(/Add/))
//       expect(screen.queryByText(/Invalid input/)).toBeNull()
//     })
//   })
// })

// test("renders App component with routes", () => {
//   render(<App />)

//   const jobsDashboardElement = screen.getAllByText(/Jobs Dashboard/i)
//   expect(jobsDashboardElement.length).toBeGreaterThan(0)

//   const nodesDashboardElement = screen.getAllByText(/Nodes Dashboard/i)
//   expect(nodesDashboardElement.length).toBeGreaterThan(0)

//   const settingsElement = screen.getAllByText(/Settings/i)
//   expect(settingsElement.length).toBeGreaterThan(0)
// })

// describe("Main Page", () => {
//   describe("Static tests", () => {
//     it("should render text input and submit button", async () => {
//       await act(() => {
//         render(<MainPage />)
//       })

//       expect(screen.getByPlaceholderText(/Add a task/)).toBeInTheDocument()
//       expect(screen.getByText(/Add/)).toBeInTheDocument()
//     })

//     it("should clear input on submit", async () => {
//       const user = userEvent.setup()
//       render(<MainPage />)
//       const input = screen.getByPlaceholderText(/Add a task/)
//       await user.type(input, "Finish course")
//       await user.click(screen.getByText(/Add/))

//       expect(input.value).toBe("")
//     })

//     it("should add to list on submit", async () => {
//       const user = userEvent.setup()
//       render(<MainPage />)
//       await user.type(
//         screen.getByPlaceholderText(/Add a task/),
//         "Finish course"
//       )
//       await user.click(screen.getByText(/Add/))

//       expect(screen.getByText(/Finish course/)).toBeInTheDocument()
//     })

//     it("should render incomplete and complete sections", async () => {
//       await act(() => {
//         render(<MainPage />)
//       })

//       expect(screen.getByText(/Incomplete Tasks/)).toBeInTheDocument()
//       expect(screen.getByText(/Completed Tasks/)).toBeInTheDocument()
//     })

//     it("should change task status on click", async () => {
//       const user = userEvent.setup()
//       render(<MainPage />)
//       await user.type(
//         screen.getByPlaceholderText(/Add a task/),
//         "Finish course"
//       )
//       await user.click(screen.getByText(/Add/))
//       await user.click(screen.getByText(/Finish course/))

//       await waitFor(() => {
//         expect(
//           screen.getByText(/Completed Tasks/).nextSibling!.firstChild
//         ).not.toBeNull()
//       })
//     })

//     it("should render completed date and time on completed tasks", async () => {
//       const date = new Date("2022-08-30T09:00:00.135").getTime()
//       jest.spyOn(global.Date, "now").mockImplementation(() => date)

//       const user = userEvent.setup()
//       render(<MainPage />)
//       await user.type(
//         screen.getByPlaceholderText(/Add a task/),
//         "Finish course"
//       )
//       await user.click(screen.getByText(/Add/))
//       await user.click(screen.getByText("Finish course"))

//       expect(screen.getByText(/2022/)).toBeInTheDocument()
//       expect(screen.getByText(/9:00:00/)).toBeInTheDocument()
//       jest.restoreAllMocks()
//     })
//   })

//   describe("HTTP test", () => {
//     afterEach(() => jest.restoreAllMocks())

//     it("should fetch tasks from backend", async () => {
//       const spy = jest.spyOn(TaskClient, "fetchTasks").mockResolvedValue([])

//       await act(() => {
//         render(<App />)
//       })

//       expect(spy).toHaveBeenCalledTimes(1)
//     })

//     it("should render fetched tasks", async () => {
//       mswServer.use(fetchTasks_incompleteTask_response)

//       render(<App />)

//       expect(await screen.findByText(/Finish course/)).toBeInTheDocument()
//     })

//     it("should send http post on submit", async () => {
//       mswServer.use(fetchTasks_empty_response, saveTasks_empty_response)
//       const putSpy = jest.spyOn(TaskClient, "saveTasks")

//       const user = userEvent.setup()
//       render(<App />)

//       await user.type(
//         screen.getByPlaceholderText(/Add a task/),
//         "Finish course"
//       )
//       await user.click(screen.getByText(/Add/))

//       expect(
//         within(screen.getByText("Incomplete Tasks").parentElement!).getByText(
//           /Finish course/
//         )
//       ).toBeInTheDocument()

//       expect(putSpy).toHaveBeenNthCalledWith(1, [
//         expect.objectContaining({
//           id: expect.any(String),
//           name: "Finish course",
//           createdOn: expect.any(Number),
//           status: TaskStatus.INCOMPLETE,
//         }),
//       ])
//     })

//     it("should send http post on status change", async () => {
//       mswServer.use(
//         fetchTasks_incompleteTask_response,
//         saveTasks_empty_response
//       )
//       const putSpy = jest.spyOn(TaskClient, "saveTasks")

//       const user = userEvent.setup()
//       render(<App />)

//       expect(await screen.findByText("Finish course")).toBeInTheDocument()

//       await user.click(
//         within(screen.getByText("Incomplete Tasks").parentElement!).getByText(
//           /Finish course/
//         )
//       )
//       expect(putSpy).toHaveBeenNthCalledWith(1, [
//         expect.objectContaining({
//           id: "1",
//           name: "Finish course",
//           createdOn: expect.any(Number),
//           completedOn: expect.any(Number),
//           status: TaskStatus.COMPLETE,
//         }),
//       ])

//       await user.click(
//         within(screen.getByText("Completed Tasks").parentElement!).getByText(
//           /Finish course/
//         )
//       )
//       expect(putSpy).toHaveBeenNthCalledWith(2, [
//         expect.objectContaining({
//           id: expect.any(String),
//           name: expect.any(String),
//           createdOn: expect.any(Number),
//           status: TaskStatus.INCOMPLETE,
//         }),
//       ])
//     })
//   })

//   describe("Input Validation", () => {
//     let user: UserEvent
//     beforeEach(() => {
//       user = userEvent.setup()
//       render(<App />)
//     })

//     it("should show error when empty input is submitted", async () => {
//       await user.click(screen.getByText(/Add/))
//       expect(screen.getByText(/Invalid input/)).toBeInTheDocument()
//     })

//     it("should clear error message once valid input is submitted", async () => {
//       await user.click(screen.getByText(/Add/))
//       expect(screen.getByText(/Invalid input/)).toBeInTheDocument()

//       const input = screen.getByPlaceholderText(/Add a task/)
//       await user.type(input, "Finish course")
//       await user.click(screen.getByText(/Add/))
//       expect(screen.queryByText(/Invalid input/)).toBeNull()
//     })
//   })
// })
