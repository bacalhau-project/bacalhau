import { defineConfig, loadEnv } from "vite"
import { fileURLToPath } from "url"
import react from "@vitejs/plugin-react"
import viteTsconfigPaths from "vite-tsconfig-paths"
import browserslistToEsbuild from "browserslist-to-esbuild"
import dotenv from "dotenv"

export default defineConfig(({ mode }) => {
  dotenv.config() // load env vars from .env
  const env = loadEnv(mode, process.cwd(), "")
  return {
    // depending on your application, base can also be "/"
    base: "",
    plugins: [react(), viteTsconfigPaths()],
    envDir: "vite/env",
    define: {
      "process.env": env,
      __VALUE__: `"${process.env.VALUE}"`, // wrapping in "" since it's a string
      _global: {},
    },
    server: {
      // this ensures that the browser opens upon server start
      open: true,
      // this sets a default port to 3000
      port: 3000,
    },
    target: browserslistToEsbuild([">0.2%", "not dead", "not op_mini all"]),
    resolve: {
      alias: {
        "@": fileURLToPath(new URL("./src", import.meta.url)),
      },
      mainFields: ["browser"],
    },
  }
})
