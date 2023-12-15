import React from 'react';
import { render, fireEvent } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { createMemoryHistory } from 'history';
// import ActionButton from '@components/ActionButton/ActionButton';
import ActionButton from '../../src/components/ActionButton/ActionButton';


describe("ActionButton", () => {
  test("renders button with provided text", () => {
    const { getByText } = render(
      <MemoryRouter>
        <ActionButton text="Test Button" />
      </MemoryRouter>
    );

    expect(getByText("Test Button")).toBeInTheDocument();
  });

  test("calls onClick when provided and button is clicked", () => {
    const handleClick = jest.fn();

    const { getByText } = render(
      <MemoryRouter>
        <ActionButton text="Test Button" onClick={handleClick} />
      </MemoryRouter>
    );

    fireEvent.click(getByText("Test Button"));

    expect(handleClick).toHaveBeenCalled();
  });

  test("navigates to 'to' path when provided and button is clicked", () => {
    const { getByText } = render(
      <MemoryRouter initialEntries={['/']}>
        <Routes>
          <Route path="/" element={<ActionButton text="Test Button" to="/test-path" />} />
          <Route path="/test-path" element={<div>Test Page</div>} />
        </Routes>
      </MemoryRouter>
    );
  
    fireEvent.click(getByText("Test Button"));
  
    // Check if the 'Test Page' content is rendered
    expect(getByText("Test Page")).toBeInTheDocument();
  });

  test("navigates to 'to/id' path when 'to' and 'id' are provided and button is clicked", () => {
    const history = createMemoryHistory();

    const { getByText } = render(
      <MemoryRouter initialEntries={['/']}>
        <Routes>
          <Route path="/" element={<ActionButton text="Test Button" to="/test-path" id="123" />} />
          <Route path="/test-path/:id" element={<div>Test Page</div>} />
        </Routes>
      </MemoryRouter>
    );

    fireEvent.click(getByText("Test Button"));

    expect(history.location.pathname).toBe("/test-path/123");
  });
});