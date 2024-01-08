// @ts-nocheck
import React from "react";
import { render, fireEvent } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { Sidebar } from "../../src/layout/Sidebar/Sidebar";

describe("Sidebar", () => {
  test("renders", () => {
    render(
      <MemoryRouter>
        <Sidebar isCollapsed="false" />
      </MemoryRouter>,
    );
  });
});
