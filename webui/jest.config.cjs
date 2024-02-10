module.exports = {
  roots: ["<rootDir>/src", "<rootDir>/tests"],
  testMatch: ["**/__tests__/**/*.[jt]s?(x)", "**/?(*.)+(spec|test).[jt]s?(x)"],
  transform: {
    "^.+\\.(ts|tsx)$": "ts-jest",
    "\\.(css|scss)$": "jest-css-modules-transform",
    "^.+\\.(js|jsx)$": "babel-jest",
  },
  moduleFileExtensions: ["ts", "tsx", "js", "jsx", "json", "node"],
  setupFiles: ["<rootDir>/jest.polyfills.cjs"],
  setupFilesAfterEnv: ["<rootDir>/src/setupTests.ts"],
  testPathIgnorePatterns: [
    "<rootDir>/tests/mocks",
    "<rootDir>/tests/setupTests.ts",
  ],
  moduleNameMapper: {
    "^@pages/(.*)$": "<rootDir>/src/pages/$1",
    "^@components/(.*)$": "<rootDir>/src/components/$1",
    "\\.svg$": "<rootDir>/tests/mocks/svgMock.mjs",
    uuid: require.resolve("uuid"),
  },
  testEnvironment: "jsdom",
  testEnvironmentOptions: {
    customExportConditions: [""],
  },
  runner: "jest-runner",
  watchPlugins: [
    "jest-watch-select-projects",
    "jest-watch-typeahead/filename",
    "jest-watch-typeahead/testname",
  ],
  watchman: false,
  maxWorkers: 1,
}
