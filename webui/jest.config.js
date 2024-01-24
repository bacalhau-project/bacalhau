module.exports = {
  roots: ["<rootDir>/src", "<rootDir>/tests"],
  collectCoverageFrom: [
    "src/**/*.{js,jsx,ts,tsx}",
    "!src/**/*.d.ts"
  ],
  testMatch: [
    "**/__tests__/**/*.[jt]s?(x)",
    "**/?(*.)+(spec|test).[jt]s?(x)"
  ],
  transform: {
    "^.+\\.(ts|tsx)$": "ts-jest",
    "^.+\\.css$": "<rootDir>/config/jest/cssTransform.js",
    "^.+\\.(js|jsx|mjs|cjs|ts|tsx)$": "<rootDir>/config/jest/babelTransform.js",
    "^(?!.*\\.(js|jsx|mjs|cjs|ts|tsx|css|json)$)": "<rootDir>/config/jest/fileTransform.js",
  },
  transformIgnorePatterns: [
    "[/\\\\]node_modules[/\\\\].+\\.(js|jsx|mjs|cjs|ts|tsx)$",
    "^.+\\.module\\.(css|sass|scss)$"
  ],
  modulePaths: [],
  moduleFileExtensions: ["ts",
    "tsx",
    "js",
    "jsx",
    "json",
    "node",
    "web.js",
    "web.ts",
    "web.tsx",
    "web.jsx",
  ],
  setupFiles: ["react-app-polyfill/jsdom", '<rootDir>/jest.polyfills.js'],
  setupFilesAfterEnv: ['<rootDir>/src/setupTests.ts'],
  testPathIgnorePatterns: [
    "<rootDir>/tests/mocks",
    "<rootDir>/tests/setupTests.ts",
  ],
  moduleNameMapper: {
    "^@pages/(.*)$": "<rootDir>/src/pages/$1",
    "^@components/(.*)$": "<rootDir>/src/components/$1",
    '\\.svg$': '<rootDir>/tests/mocks/svgMock.js',
    "^react-native$": "react-native-web",
    "^.+\\.module\\.(css|sass|scss)$": "identity-obj-proxy"
  },
  testEnvironment: 'jsdom',
  testEnvironmentOptions: {
    customExportConditions: [''],
  },
  runner: 'jest-runner',
  watchPlugins: ['jest-watch-select-projects', 'jest-watch-typeahead/filename', 'jest-watch-typeahead/testname'],
  watchman: false,
  maxWorkers: 1,
  resetMocks: true,
}
