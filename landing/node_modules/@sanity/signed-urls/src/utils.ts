import type {Param} from './types'

/**
 * Converts a base64 string to a byte array.
 *
 * @internal
 * @param base64 - The base64 string to convert.
 * @returns The converted byte array.
 */
export function base64ToBytes(base64: string): Uint8Array {
  if (typeof Buffer !== 'undefined') {
    // Node: base64 → Buffer → Uint8Array
    return new Uint8Array(Buffer.from(base64, 'base64'))
  }
  // Browser: base64 → atob → Uint8Array
  const bin = atob(base64)
  return Uint8Array.from(bin, (c) => c.charCodeAt(0))
}

/**
 * Converts a byte array to a base64 string.
 *
 * @internal
 * @param bytes - The byte array to convert.
 * @returns The converted base64-encoded string.
 */
export function bytesToBase64(bytes: Uint8Array): string {
  if (typeof Buffer !== 'undefined') {
    // Node: Uint8Array → Buffer → base64
    return Buffer.from(bytes).toString('base64')
  }
  // Browser: Uint8Array → binary string → btoa
  const bin = String.fromCharCode(...bytes)
  return btoa(bin)
}

/**
 * Extract the 32-byte Ed25519 private seed from a PKCS#8 DER-encoded key.
 *
 * PKCS#8 is a standard format for storing private keys. Internally it uses a
 * binary encoding called ASN.1 DER, which structures data as TLV blocks:
 *   [Tag][Length][Value]
 *
 * For Ed25519 keys, the actual private seed we need is stored as:
 *   - Tag = 0x04 (this means "OCTET STRING" = just raw bytes)
 *   - Length = 0x20 (32 in decimal)
 *   - Value = the 32 raw seed bytes
 *
 * Multiple OCTET STRINGs can appear in the file, so we scan **backwards**
 * from the end of the buffer to make sure we find the last 32-byte one,
 * which is the real seed.
 *
 * @internal
 * @param der - The DER-encoded PKCS#8 key as a byte array
 * @returns A Uint8Array of the 32-byte seed
 * @throws If a 32-byte OCTET STRING cannot be found
 */
export function ed25519SeedFromPkcs8(der: Uint8Array): Uint8Array {
  const TAG_OCTET_STRING = 0x04 // ASN.1 tag for OCTET STRING / "raw bytes"
  const SEED_LEN = 32 // Ed25519 private seeds are exactly 32 bytes

  // Each block has at least 2 bytes before the actual value:
  //   1 byte for Tag + 1 byte for Length
  // So start scanning from the end minus (32 + 2) to ensure there's room.
  for (let i = der.length - (SEED_LEN + 2); i >= 0; i--) {
    // Look for a block where:
    //   - Tag byte = OCTET STRING
    //   - Length byte = 32
    if (der[i] === TAG_OCTET_STRING && der[i + 1] === SEED_LEN) {
      // Return just the 32 value bytes that follow
      return der.subarray(i + 2, i + 2 + SEED_LEN)
    }
  }
  throw new Error('Ed25519 32-byte seed not found in PKCS#8')
}

/**
 * Extracts the base64-encoded contents from a PEM-formatted key.
 *
 * The PEM format wraps base64-encoded DER data with header and footer lines.
 * This function removes those lines and any whitespace to return just the
 * base64-encoded contents.
 *
 * @internal
 * @param pem - The PEM-formatted key
 * @returns The base64-encoded contents between the header and footer
 */
export function extractPemContents(pem: string): string {
  return pem
    .replace(/^-----BEGIN [^-]+-----$/gim, '') // Remove the header
    .replace(/^-----END [^-]+-----$/gim, '') // Remove the footer
    .replace(/\s+/g, '') // Remove all whitespace
}

/**
 * Lexicographically sorts two [key, value] parameter tuples.
 *
 * Sorting is first by key, then by value if keys are identical.
 *
 * @internal
 * @param a - The first parameter tuple to compare
 * @param b - The second parameter tuple to compare
 * @returns A negative number if a < b, positive if a > b, or 0 if equal
 */
export function lexographicSort(a: Param, b: Param): number {
  const [keyA, valueA] = a
  const [keyB, valueB] = b

  // First compare the keys
  if (keyA < keyB) return -1
  if (keyA > keyB) return 1

  // Keys are equal → compare the values
  if (valueA < valueB) return -1
  if (valueA > valueB) return 1

  // Keys and values are identical
  return 0
}

/**
 * Normalizes base64url encoding to match RFC 4648 "URL and Filename Safe"
 * Base64 Alphabet, but with padding. Replaces '+' with '-', and '/' with '_',
 * but maintains '=' padding.
 *
 * @internal
 * @param base64 - The base64 string to normalize
 * @returns The normalized base64url string
 */
export function normalizeBase64Url(base64: string): string {
  return base64.replace(/\+/g, '-').replace(/\//g, '_')
}

/**
 * Normalizes expiry to ISO string format with validation.
 *
 * @internal
 * @param expiry - The expiry date to normalize
 * @returns The normalized expiry date as an ISO string
 */
export function normalizeExpiry(expiry: Date | number | string): string {
  let date: Date
  if (expiry instanceof Date) {
    date = expiry
  } else {
    date = new Date(expiry)
    if (isNaN(date.getTime())) {
      throw new Error('Invalid expiry date format')
    }
  }

  const now = new Date()
  if (date.getTime() <= now.getTime()) {
    throw new Error('Expiry date must be in the future')
  }

  // Format as 'YYYY-MM-DDTHH:mm:ssZ' (strip milliseconds)
  return date.toISOString().replace(/\.\d{3}Z$/, 'Z')
}

/**
 * Encodes a string per RFC 3986 for use in URLs.
 *
 * This function uses `encodeURIComponent` and additionally encodes
 * characters that are not encoded by it: `!'()*`
 *
 * @internal
 * @param str - The string to encode
 * @returns The RFC 3986 encoded string
 */
export function rfc3986(str: string) {
  return encodeURIComponent(str).replace(
    /[!'()*]/g,
    (c) => `%${c.charCodeAt(0).toString(16).toUpperCase()}`,
  )
}

/**
 * Converts a byte array to base64url with padding.
 *
 * @internal
 * @param bytes - The byte array to convert
 * @returns The base64url-encoded string
 */
export function toBase64UrlWithPadding(bytes: Uint8Array): string {
  let b64 = bytesToBase64(bytes)

  // Ensure padding, although likely unnecessary as an Ed25519 signature is
  // always 64 bytes / 88 chars when encoded in b64
  if (b64.length % 4) b64 += '='.repeat(4 - (b64.length % 4))

  return normalizeBase64Url(b64)
}
