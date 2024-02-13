import { defineConfig, loadEnv } from "vite"
import { fileURLToPath } from "url"
import react from "@vitejs/plugin-react"
import viteTsconfigPaths from "vite-tsconfig-paths"
import browserslistToEsbuild from "browserslist-to-esbuild"
import dotenv from "dotenv"
import path from "path"
import { createRequire } from 'node:module';
const require = createRequire(import.meta.url);

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
      middleware: [
        (req, res, next) => {
          if (req.url === "/mockServiceWorker.js") {
            res.setHeader('Service-Worker-Allowed', '/')
            res.setHeader('Content-Type', 'application/javascript')
          }
        },
      ],
    },
    target: browserslistToEsbuild([">0.2%", "not dead", "not op_mini all"]),
    resolve: {
      alias: {
        "@": fileURLToPath(new URL("./src", import.meta.url)),
        'msw/native': require.resolve(path.resolve(__dirname, './node_modules/msw/lib/native/index.mjs')),
      },
      mainFields: ["browser"],
    },
    build: {
      outDir: path.resolve(__dirname, "build"),
    },
    publicDir: "./public",

  }
})
