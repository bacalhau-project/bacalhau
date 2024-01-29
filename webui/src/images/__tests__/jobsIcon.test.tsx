import { render, screen } from "@testing-library/react"
import { SVGImage } from "../svg-image"

export const AppTest = () => (
  <SVGImage src="./jobs-icon.svg" alt="jobs icon alt" />
)

test("renders JobsIcon", () => {
  render(<AppTest />)

  expect(screen.findAllByText("JobsIcon")).toBeTruthy()
})
