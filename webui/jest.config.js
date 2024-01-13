module.exports = {
  roots: ["<rootDir>/src", "<rootDir>/tests"],
  testMatch: [
    "**/tests/**/*.+(ts|tsx|js)",
    "**/?(*.)+(spec|test).+(ts|tsx|js)",
  ],
  transform: {
    "^.+\\.(ts|tsx)$": "ts-jest",
    "\\.(css|scss)$": "jest-css-modules-transform",
    "^.+\\.(js|jsx)$": "babel-jest",
  },
  moduleFileExtensions: ["ts", "tsx", "js", "jsx", "json", "node"],
  setupFilesAfterEnv: ["<rootDir>/tests/setupTests.ts"],
  testPathIgnorePatterns: [
    "<rootDir>/tests/mocks",
    "<rootDir>/tests/setupTests.ts",
  ],
  moduleNameMapper: {
    "^@pages/(.*)$": "<rootDir>/src/pages/$1",
    "^@components/(.*)$": "<rootDir>/src/components/$1",
    '\\.svg$': '<rootDir>/tests/mocks/svgMock.js',
  },
  testEnvironment: "jsdom",
}
