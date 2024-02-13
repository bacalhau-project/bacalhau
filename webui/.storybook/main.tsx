// Replace your-framework with the framework you are using (e.g., react-vite, vue3-vite)
import type { StorybookConfig } from "@storybook/react-vite"
import { mergeConfig } from "vite"
import path from "path"

const config: StorybookConfig = {
  stories: [
    "../src/stories/*.mdx",
    "../src/**/*.mdx",
    "../src/**/*.stories.@(js|jsx|mjs|ts|tsx)",
  ],

  framework: {
    name: "@storybook/react-vite",
    options: {},
  },
  staticDirs: ['../public'],

  addons: [
    "@storybook/addon-links",
  ],
  core: {
    builder: "@storybook/builder-vite",
  },
  docs: {
    autodocs: true,
  },
  async viteFinal(config, { configType }) {
    if (!config.resolve) {
      config.resolve = {}
    }
    if (!config.resolve.alias) {
      config.resolve.alias = {}
    }
    if (configType === "DEVELOPMENT") {
      config.resolve.alias["./cryptoFunctions"] = path.resolve(
        __dirname,
        "../tests/mocks/__mocks__/cryptoFunctions.ts"
      )

    }
    if (configType === "PRODUCTION") {
      // Prodcution Specific Config
    }
    return mergeConfig(config, {
      optimizeDeps: {
        include: ["storybook-dark-mode"],
      },
    })
  },
}

export default config
