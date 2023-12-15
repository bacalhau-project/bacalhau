import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import App from "./App";

test("renders App component with routes", () => {
  render(
    <MemoryRouter initialEntries={["/"]} initialIndex={0}>
      <App />
    </MemoryRouter>
  );

  const homeElement = screen.getByText(/Home/i);
  expect(homeElement).toBeInTheDocument();

  const jobsDashboardElement = screen.getByText(/Jobs Dashboard/i);
  expect(jobsDashboardElement).toBeInTheDocument();

  const nodesDashboardElement = screen.getByText(/Nodes Dashboard/i);
  expect(nodesDashboardElement).toBeInTheDocument();

  const settingsElement = screen.getByText(/Settings/i);
  expect(settingsElement).toBeInTheDocument();
});