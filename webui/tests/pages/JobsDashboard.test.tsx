// @ts-nocheck
import React from "react";
import { MemoryRouter } from "react-router-dom";
import { screen, render } from "@testing-library/react";
import { JobsDashboard } from "../../src/pages/JobsDashboard/JobsDashboard";
import axios from "axios"; // Import axios module

jest.mock("axios");

axios.get.mockResolvedValueOnce({ data: { Jobs: [{ id: 1, name: "test" }] } });

describe("JobsDashboard", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  test("renders JobsDashboard", () => {
    render(
      <MemoryRouter>
        <JobsDashboard />
      </MemoryRouter>
    );

    expect(screen.getAllByText(/Jobs Dashboard/i).length).toBeGreaterThan(0);
  });
});
