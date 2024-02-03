// Creating an external file for importing crypto functions so that 
// we can mock these functions where crypto isn't available (like the browser).
// Mocks are in __mocks__/cryptoFunctions.ts
import { randomBytes as randomBytesFn, randomUUID as randomUUIDFn } from "crypto"

export const randomBytes = randomBytesFn
export const randomUUID = randomUUIDFn