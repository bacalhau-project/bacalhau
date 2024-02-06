import type { StorybookConfig } from "@storybook/react-webpack5";

const baseConfig = require("../webpack.config");

const config: StorybookConfig = {
  stories: [
    "../src/**/*.mdx",
    "../src/**/*.stories.ts",
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
    'storybook-addon-module-mock',
    'storybook-addon-react-router-v6',
    '@storybook/addon-mdx-gfm'
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
    const storybookWebpackConfig = config;
    return {
      ...storybookWebpackConfig,
      module: {
        ...storybookWebpackConfig.module,
        rules: [...(storybookWebpackConfig.module?.rules ?? []), ...(baseConfig.module?.rules ?? [])]
      },
    };
  },
}

export default config