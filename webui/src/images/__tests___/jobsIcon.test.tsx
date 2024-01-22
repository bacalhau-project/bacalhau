import { render, screen } from "@testing-library/react"
import { ReactComponent as JobsIcon } from "../jobs-icon.svg"

export const AppTest = () => <JobsIcon />

test("renders JobsIcon", () => {
  render(<AppTest />)

  expect(screen.findAllByText("JobsIcon")).toBeTruthy()
})
