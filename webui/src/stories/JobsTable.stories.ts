import type { Meta, StoryObj } from '@storybook/react';
import { HttpResponse, http } from "msw";
import { JobsTable } from "../pages/JobsDashboard/JobsTable/JobsTable";

const meta: Meta<typeof JobsTable> = {
  title: 'Jobs Table',
  component: JobsTable,
};

export default meta;

type Story = StoryObj<typeof JobsTable>;

export const Default: Story = {
  args: { data: [] },
  tags: ["autodocs"],
}

export const OneJob: Story = {
  args: {},
  tags: ["autodocs"],
  parameters: {
    msw: [
      http.get('/api/v1/orchestrator/jobs', ({ request }) => {
        return HttpResponse.json({ data: [{ id: "1", name: "Job 1" }] });
      }),
    ],
  },
}
