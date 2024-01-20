import { render, screen } from "@testing-library/react"
import { ReactComponent as JobsIcon } from "../../src/images/jobs-icon.svg"

export const AppTest = () => <JobsIcon />

test("renders JobsIcon", () => {
  render(<AppTest />)

  screen.debug()

  expect(screen.findAllByText("JobsIcon")).toBeTruthy()
})
