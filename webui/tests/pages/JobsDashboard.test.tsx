// @ts-nocheck
import React from "react";
import { MemoryRouter } from "react-router-dom";
import { render, screen } from "@testing-library/react";
import axios from "axios";
import { JobsDashboard } from "../../src/pages/JobsDashboard/JobsDashboard";

jest.mock('axios', () => {
  return {
    create: jest.fn(() => ({
      get: jest.fn().mockResolvedValue({
        data: {
          Jobs: [
            { id: 1, name: "Job1" },
            { id: 2, name: "Job2" },
          ],
        },
      }),
    })),
  };
});


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

  test("renders JobsTable with data on successful API call", async () => {
    render(
      <MemoryRouter>
        <JobsDashboard />
      </MemoryRouter>
    );
  
    expect(await screen.findByText("Job1")).toBeInTheDocument();
    expect(await screen.findByText("Job2")).toBeInTheDocument();
  });
  
  test("handles API call failure gracefully", async () => {
    axios.create().get.mockRejectedValueOnce(new Error("API Error"));
    render(
      <MemoryRouter>
        <JobsDashboard />
      </MemoryRouter>
    );
  
    // Replace this with how your component shows errors
    expect(await screen.findByText(/error/i)).toBeInTheDocument();
  });
});