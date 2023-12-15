// @ts-nocheck
import React from 'react';
import { render, screen } from '@testing-library/react';
import JobsDashboard from '@pages/JobsDashboard/JobsDashboard';

describe('JobsDashboard', () => {
    
    beforeEach(() => {
        render(<JobsDashboard />);
    });

    it('renders JobsDashboard', () => {
        expect(screen.getByTestId('JobsDashboard')).toBeInTheDocument();
    });
      
});
