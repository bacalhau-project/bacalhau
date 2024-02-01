import { render, screen } from "@testing-library/react"
import { ReactComponent as JobsIcon } from "../jobs-icon.svg"

export const AppTest = () => <JobsIcon />

test("renders JobsIcon", () => {
  render(<AppTest />)

  const screenContent = screen
    .findAllByAltText("JobsIcon")
    .then((content) => content)
  expect(screenContent).toBeTruthy()
})
