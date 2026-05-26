import {etc} from '@noble/ed25519'

import {base64ToBytes, ed25519SeedFromPkcs8, extractPemContents} from './utils'

/**
 * Converts a PEM-formatted PKCS#8 Ed25519 private key to its 32-byte seed (Uint8Array).
 * @public
 * @param pem - The PEM-formatted PKCS#8 key (e.g. "BEGIN PRIVATE KEY")
 * @returns The 32-byte Ed25519 seed.
 * @throws If the PEM/DER is invalid or a 32-byte seed cannot be found.
 */
export function pemToEd25519Bytes(pem: string): Uint8Array {
  // Extract the DER from the PEM
  const der = base64ToBytes(extractPemContents(pem))
  // Return the 32-byte Ed25519 seed from the DER
  return ed25519SeedFromPkcs8(der)
}

/**
 * Converts a PEM-formatted PKCS#8 Ed25519 private key to its 32-byte seed (hex string).
 * @public
 * @param pem - The PEM-formatted PKCS#8 key (e.g. "BEGIN PRIVATE KEY")
 * @returns The 32-byte Ed25519 seed as a lowercase hex string.
 * @throws If the PEM/DER is invalid or a 32-byte seed cannot be found.
 */
export function pemToEd25519Hex(pem: string): string {
  return etc.bytesToHex(pemToEd25519Bytes(pem))
}
