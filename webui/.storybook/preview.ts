import type { Preview } from "@storybook/react";
import { reactRouterParameters, withRouter } from 'storybook-addon-react-router-v6';
import '../src/index.scss';

export const preview: Preview = {
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
