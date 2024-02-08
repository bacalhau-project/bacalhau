// Random crypto functions that are available in browsers
// because 'crypto' is not available in the browser.
// cspell:ignore caas, prettier, uuid, yxxx

export function randomBytes(length: number): Uint8Array {
  // Generate pseudo-random bytes without using crypto
  const randomBytesReturn = new Uint8Array(length)
  for (let i = 0; i < length; i += 1) {
    randomBytesReturn[i] = Math.floor(Math.random() * 256)
  }
  return randomBytesReturn
}

export function randomUUID(): string {
  // Generate a pseudo-random UUID without using crypto
  return "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(/[xy]/g, (c) => {
    const r = Math.floor(Math.random() * 16)
    let v
    if (c === "x") {
      v = r
    } else if (r % 4 < 2) {
      v = r % 4
    } else {
      v = 8
    }
    return v.toString(16)
  })
}
