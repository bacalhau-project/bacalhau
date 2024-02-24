import type { Preview } from "@storybook/react"
import { passthrough } from "msw"
import { setupWorker } from "msw/browser"
import {
  reactRouterParameters,
  withRouter,
} from "storybook-addon-react-router-v6"
import { handlers as storyBookHandlers } from "../.storybook/storybookHandlers"
import "../src/index.scss"
import { worker as importedWorker } from "../tests/msw/browser"

// const handlers = []

const MSW_FILE = "mockServiceWorker.js"
const worker = setupWorker(...importedWorker.listHandlers(), ...storyBookHandlers)
await worker.start(
  {
    serviceWorker: {
      url: MSW_FILE,
      options: {
        scope: '/',
      },
    },
    onUnhandledRequest: ({ method, url }) => {
      console.info(`Full: ${method} ${url}`)
      if (!url.includes("/api")) {
        console.info(`Passthrough: ${method} ${url}`)
        return passthrough();
      }
    },
  },
)


if (typeof global.process === "undefined") {
  const { worker } = require("../tests/mocks/browser")
  worker.start()
}

export default {
  decorators: [withRouter],
  parameters: {
    reactRouter: reactRouterParameters({}),
  },
} satisfies Preview
