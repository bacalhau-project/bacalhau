import type { Preview } from "@storybook/react"
import {
  reactRouterParameters,
  withRouter,
} from "storybook-addon-react-router-v6"
import "../src/index.scss"
import { passthrough } from "msw"
// import { setupWorker } from "msw/browser"
import { initialize, mswDecorator } from 'msw-storybook-addon';
import { handlers as storyBookHandlers } from "../.storybook/storybookHandlers"

// const handlers = []

const MSW_FILE = "mockServiceWorker.js"
// const worker = setupWorker(...handlers)
// await worker.start(
//   {
//     serviceWorker: {
//       url: MSW_FILE,
//       options: {
//         scope: '/',
//       },
//     },
//     onUnhandledRequest: ({ method, url }) => {
//       console.info(`Full: ${method} ${url}`)
//       if (!url.includes("/api")) {
//         console.info(`Passthrough: ${method} ${url}`)
//         return passthrough();
//       }
//     },
//   },
// )

initialize({
  onUnhandledRequest: ({ method, url }) => {
    console.info(`Full: ${method} ${url}`)
    if (!url.includes("/api")) {
      console.info(`Passthrough: ${method} ${url}`)
      return passthrough();
    }
  },
  serviceWorker: {
    url: MSW_FILE,
    options: {
      scope: '/',
    },
  },
},
  storyBookHandlers,
)

export const preview: Preview = {
  // decorators: [(storyFn, context) => withConsole()(storyFn)(context)],
  parameters: {
    actions: { argTypesRegex: "^on[A-Z].*" },
    controls: {
      matchers: {
        color: /(background|color)$/i,
        date: /Date$/i,
      },
    },
  },
  decorators: [mswDecorator],
}

// if (typeof global.process === "undefined") {
//   const { worker } = require("../tests/mocks/browser")
//   worker.start()
// }

export default {
  decorators: [withRouter],
  parameters: {
    reactRouter: reactRouterParameters({}),
  },
} satisfies Preview
