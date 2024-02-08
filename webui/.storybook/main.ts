import type { StorybookConfig } from "@storybook/react-webpack5";

import { config as baseConfig } from '../webpack.config';

const config: StorybookConfig = {
  stories: [
    "../src/**/*.mdx",
    "../src/**/*.stories.@(js|jsx|mjs|ts|tsx)",
  ],
  addons: [
    "@storybook/addon-links",
    "@storybook/addon-essentials",
    "@storybook/addon-onboarding",
    "@storybook/addon-interactions",
    '@storybook/addon-storysource',
    '@storybook/addon-controls',
    '@storybook/addon-actions',
    '@storybook/addon-viewport',
    '@storybook/addon-a11y',
    '@storybook/react',
    {
      name: '@storybook/addon-coverage',
      options: {
        istanbul: {
          exclude: ['**/components/**/index.ts'],
        },
      },
    },
    'storybook-addon-module-mock',
    'storybook-addon-react-router-v6',
  ],
  framework: {
    name: "@storybook/react-webpack5",
    options: {
      builder: {
        useSWC: true,
      },
    },
  },
  docs: {
    autodocs: "tag",
  },
  typescript: {
    reactDocgen: 'react-docgen',
  },
  webpackFinal: async (config) => {
    const webpackConfig = config;
    return {
      ...webpackConfig,
      module: {
        ...webpackConfig.module,
        rules: [...(webpackConfig.module?.rules ?? []), ...(baseConfig.module?.rules ?? [])]
      },
    };
  },
}

export default config