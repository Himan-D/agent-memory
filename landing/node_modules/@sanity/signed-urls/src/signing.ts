import {etc, hashes, sign} from '@noble/ed25519'
import {sha512} from '@noble/hashes/sha2.js'

import type {Param, SigningOptions} from './types'

import {lexographicSort, normalizeExpiry, rfc3986, toBase64UrlWithPadding} from './utils'

hashes.sha512 = sha512

/**
 * Extracts user-defined query parameters from a URL, excluding reserved signing parameters.
 *
 * @internal
 * @param url - The URL from which to extract user parameters
 * @returns An array of [key, value] tuples for user-defined parameters
 */
export function extractUserParams(url: URL): Param[] {
  const reservedKeys = ['keyid', 'expiry', 'signature']
  return Array.from(url.searchParams).reduce<Param[]>((params, param) => {
    const key = param[0]
    if (!reservedKeys.includes(key)) params.push(param)
    return params
  }, [])
}

/**
 * Generates an Ed25519 signature for a given URL.
 *
 * @public
 * @param url - The URL to sign
 * @param privateKey - The private key to use for signing
 * @returns The base64url-encoded signature
 */
export function generateSignature(url: string, privateKey: string): string {
  // Encode the URL as bytes
  const urlBytes = new TextEncoder().encode(url)
  // Encode the private key as bytes
  const privateKeyBytes = etc.hexToBytes(privateKey)
  // Get the signed URL as bytes
  const signatureBytes = sign(urlBytes, privateKeyBytes)
  // Convert the signature to a URL-safe base64 string
  const urlSafeSignature = toBase64UrlWithPadding(signatureBytes)
  // Return the URL-safe signature, encoded per RFC 3986
  return urlSafeSignature
}

/**
 * Generates a lexicographically sorted canonical query string from parameter tuples.
 *
 * The canonical query string is formed by:
 *   - Encoding each key and value per RFC 3986
 *   - Sorting parameters lexicographically by key, then by value
 *   - Joining as key=value pairs with '&' separation
 *
 * @internal
 * @param params - An array of [key, value] parameter tuples
 * @returns The canonical query string of user parameters
 */
export function getCanonicalQuery(params: Param[]): string {
  // Encode each key and value
  const encodedParams = params.map((param) => param.map(rfc3986) as Param)
  // Sort params lexicographically by key, then by value
  encodedParams.sort(lexographicSort)
  // Join as key=value pairs with '&' separation
  return encodedParams.map((param) => param.join('=')).join('&')
}

/**
 * Signs a URL with Ed25519 signature, adding keyid, expiry, and signature parameters.
 *
 * @public
 * @param url - The URL to sign
 * @param options - The signing options to use
 * @returns The signed URL
 */
export function signUrl(url: string | URL, options: SigningOptions): string {
  validateSigningOptions(options)

  const urlObj = new URL(url)
  // Extract user-defined query parameters, excluding reserved signing parameters
  const userParams = extractUserParams(urlObj)
  // Canonicalize the query string from the user parameters
  const canonicalQuery = getCanonicalQuery(userParams)
  // Get the URL with signing parameters (keyid and expiry)
  const urlToSign = urlWithSigningParams(urlObj, canonicalQuery, options)
  // Generate a signature from the fully canonicalized URL
  const signature = generateSignature(urlToSign, options.privateKey)
  // Append the RFC 3986 encoded signature
  const urlWithSignature = `${urlToSign}&signature=${rfc3986(signature)}`
  return urlWithSignature
}

/**
 * Returns a ready-to-sign URL object with signing parameters (keyid and expiry).
 *
 * @public
 * @param url - The base URL to which signing parameters will be added
 * @param query - The canonical query string of user parameters
 * @param options - The signing options containing keyId and expiry
 * @returns A "signable" URL string with keyid and expiry parameters
 */
export function urlWithSigningParams(
  url: string | URL,
  query: string,
  signingOptions: Omit<SigningOptions, 'privateKey'>,
): string {
  const {origin, pathname} = new URL(url)
  const parts = [origin, pathname, '?']

  if (query.length > 0) {
    parts.push(query)
    parts.push('&')
  }

  parts.push(`keyid=${rfc3986(signingOptions.keyId)}`)
  parts.push(`&expiry=${rfc3986(normalizeExpiry(signingOptions.expiry))}`)

  return parts.join('')
}

/**
 * Validates signing options parameters.
 *
 * @internal
 * @param options - The signing options to validate
 * @throws When required parameters are missing or empty
 */
function validateSigningOptions(options: SigningOptions): void {
  // Validate required parameters
  if (!options.keyId) {
    throw new Error('Missing required parameter: keyId')
  }
  if (!options.privateKey) {
    throw new Error('Missing required parameter: privateKey')
  }
  if (!options.expiry) {
    throw new Error('Missing required parameter: expiry')
  }

  // Validate strings are not empty
  if (typeof options.keyId === 'string' && options.keyId.trim() === '') {
    throw new Error('keyId cannot be empty')
  }
  if (typeof options.privateKey === 'string' && options.privateKey.trim() === '') {
    throw new Error('privateKey cannot be empty')
  }
}
