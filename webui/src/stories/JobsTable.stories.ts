import { generateMockJob } from '../../tests/mocks/jobMock';
import { Job } from '../helpers/jobInterfaces';
import { JobsTable } from '../pages/JobsDashboard/JobsTable/JobsTable';

export default {
    component: JobsTable,
    title: 'JobsTable',
    tags: ['autodocs'],
};

const fullData = (): Job[] => {
    // Create a list of 10 jobs
    const jobs: Job[] = []
    for (let i = 0; i < 10; i++) {
        jobs.push(generateMockJob())
    }
    return jobs
}

export const Default = {
    args: { data: [] },
};

export const FullData = {
    args: { data: fullData() },
};