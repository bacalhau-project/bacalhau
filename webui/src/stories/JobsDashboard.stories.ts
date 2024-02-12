import type { Meta, StoryObj } from "@storybook/react"
import { HttpResponse, http } from "msw"
import { JobsDashboard } from "../pages/JobsDashboard/JobsDashboard"

const BASE_URL = "http://localhost:1234"
const meta: Meta<typeof JobsDashboard> = {
  component: JobsDashboard,
  title: "JobsDashboard",
  tags: ["autodocs"],
  parameters: { msw: true },
}

export default meta
type Story = StoryObj<typeof JobsDashboard>

// export const Simple: Story = {
//   args: {
//     user: PageLayout.Simple.args.user,
//     document: DocumentHeader.Simple.args.document,
//     subdocuments: DocumentList.Simple.args.documents,
//   },
// }

export const MockedSuccess: Story = {
  parameters: {
    msw: {
      handlers: [
        http.get(`${BASE_URL}/api/v1/orchestrator/jobs`, () =>
          // ?order_by=created_at&reverse=true&limit=10
          HttpResponse.json({})
        ),
      ],
    },
  },
}

// export const MockedError: Story = {
//   parameters: {
//     msw: [
//       rest.get('https://your-restful-endpoint', (_req, res, ctx) => {
//         return res(ctx.delay(800), ctx.status(403));
//       }),
//     ],
//   },
// };
