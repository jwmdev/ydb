/* eslint-env browser */

export const Uint8Array_ = Uint8Array

export const createUint8ArrayByLen = len => new Uint8Array_(len)

/**
 * Create Uint8Array with initial content from buffer
 */
export const createUint8ArrayByBuffer = (buffer, byteOffset, length) => new Uint8Array_(buffer, byteOffset, length)

/**
 * Create Uint8Array with initial content from buffer
 */
export const createUint8ArrayByArrayBuffer = arraybuffer => new Uint8Array_(arraybuffer)

export const createPromise = f => new Promise(f)
/**
 * `Promise.all` wait for all promises in the array to resolve and return the result
 * @param {Array<Promise<any>>} arrp
 * @return {any}
 */
export const pall = arrp => Promise.all(arrp)
export const preject = reason => Promise.reject(reason)
export const presolve = res => Promise.resolve(res)

export const error = description => new Error(description)

export const max = (a, b) => a > b ? a : b
