import { Meta, StoryObj } from "@storybook/react";
import { handlers as storybookHandlers } from "../../.storybook/storybookHandlers";
import { JobsDashboard } from "../pages/JobsDashboard/JobsDashboard";

const meta: Meta<typeof JobsDashboard> = {
  component: JobsDashboard,
  args: {
    pageTitle: "Jobs Dashboard",
  },
};

export default meta;
type Story = StoryObj<typeof JobsDashboard>;

export const Simple: Story = {
  parameters: {
    msw: {
      handlers: storybookHandlers,
    },
  },
  args: {
    pageTitle: "foobaz",
  },
};
