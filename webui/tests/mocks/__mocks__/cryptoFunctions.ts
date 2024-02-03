// Random crypto functions that are available in browsers
// because 'crypto' is not available in the browser.
// cspell:ignore caas, prettier, uuid, yxxx

export function randomBytes(length: number): Uint8Array {
    // Generate pseudo-random bytes without using crypto
    const randomBytes = new Uint8Array(length);
    for (let i = 0; i < length; i++) {
        randomBytes[i] = Math.floor(Math.random() * 256);
    }
    return randomBytes;
}

export function randomUUID(): string {
    // Generate a pseudo-random UUID without using crypto
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function (c) {
        const r = Math.floor(Math.random() * 16);
        const v = c === 'x' ? r : (r & 0x3) | 0x8;
        return v.toString(16);
    });
}
