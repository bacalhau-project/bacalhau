import type { Meta, StoryObj } from '@storybook/react';
import { Job } from "../helpers/jobInterfaces";
import { JobsTable } from "../pages/JobsDashboard/JobsTable/JobsTable";
import HundredJobsData from "./storyData/jobsTable/100-jobs.json";


const meta: Meta<typeof JobsTable> = {
  title: 'Jobs Table',
  component: JobsTable,
};

export default meta;

type Story = StoryObj<typeof JobsTable>;

export const Default: Story = {
  args: { data: [] },
  argTypes: { data: { control: { type: 'array' } } },
  tags: ["autodocs"],
}

function getData(numOfJobs: number): Job[] {
  // Slice 100 jobs to the number of jobs requested
  return HundredJobsData.slice(0, numOfJobs);
}

export const OneJob: Story = {
  args: { data: getData(1) },
  argTypes: {},
  tags: ["autodocs"],
}

export const TenJobs: Story = {
  args: { data: getData(10) },
  argTypes: {},
  tags: ["autodocs"],
}

export const HundredJobs: Story = {
  args: { data: getData(100).reverse() },
  argTypes: {},
  tags: ["autodocs"],
}
