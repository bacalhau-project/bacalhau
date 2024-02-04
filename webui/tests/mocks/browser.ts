import { setupWorker } from "msw/browser"
import { storybookHandlers } from "../msw/storybookHandlers"

export const worker = setupWorker(...storybookHandlers)
