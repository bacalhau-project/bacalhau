import React from "react"
import { render, screen } from "@testing-library/react"
import { SVGImage } from "../svg-image"

export const AppTest = () => (
  <SVGImage src="/images/jobs-icon.svg" alt="Jobs Icon" />
)

test("renders JobsIcon", () => {
  render(<AppTest />)

  const screenContent = screen
    .findAllByAltText("JobsIcon")
    .then((content) => content)
  expect(screenContent).toBeTruthy()
})
