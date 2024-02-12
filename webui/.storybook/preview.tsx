import type { Preview } from "@storybook/react";
import { reactRouterParameters, withRouter } from 'storybook-addon-react-router-v6';
import '../src/index.scss';
import { initialize, mswLoader } from 'msw-storybook-addon';

// Initialize MSW
initialize();

// Provide the MSW addon loader globally. A loader runs before a story renders, avoiding potential race conditions.
export const loaders = [mswLoader];

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
}

if (typeof global.process === 'undefined') {
  const { worker } = require('../tests/mocks/browser')
  worker.start()
}

export default {
  decorators: [withRouter],
  parameters: {
    reactRouter: reactRouterParameters({}),
  }
} satisfies Preview;