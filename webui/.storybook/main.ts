import type { StorybookConfig } from "@storybook/react-webpack5";
const { resolve } = require('path');
const path = require('path');
const { merge } = require('webpack-merge');

import MiniCssExtractPlugin from 'mini-css-extract-plugin';

const stylesHandler = MiniCssExtractPlugin.loader;

const baseConfig = require("../webpack.config");

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
    'storybook-addon-module-mock',
    'storybook-addon-react-router-v6',
    '@storybook/addon-mdx-gfm'
  ],
  framework: {
    name: "@storybook/react-webpack5",
    options: {
      builder: {
        useSWC: true,
        name: "@storybook/builder-webpack5",
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

    if (!storybookWebpackConfig.resolve) {
      storybookWebpackConfig.resolve = {};
    }
    if (!storybookWebpackConfig.resolve.alias) {
      storybookWebpackConfig.resolve.alias = {};
    }

    if (!storybookWebpackConfig.module) {
      storybookWebpackConfig.module = {};
    }
    if (!storybookWebpackConfig.module.rules) {
      storybookWebpackConfig.module.rules = [];
    }

    storybookWebpackConfig.resolve.alias['./cryptoFunctions'] = path.resolve(__dirname, '../tests/mocks/__mocks__/cryptoFunctions.ts')

    storybookWebpackConfig.resolve.alias['react-router-dom'] = require.resolve('react-router-dom');

    storybookWebpackConfig.module.rules.push({
      include: path.resolve(__dirname, '../src'),
    });

    return merge(storybookWebpackConfig, baseConfig);

    // return {
    //   ...storybookWebpackConfig,
    //   module: {
    //     ...storybookWebpackConfig.module,
    //     rules: [...(baseConfig.module?.rules ?? []), ...(storybookWebpackConfig.module?.rules ?? [])]
    //   },
    // };
  },
}

export default config
