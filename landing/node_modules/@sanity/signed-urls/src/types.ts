export type Param = [string, string]

/**
 * Options for signing URLs with Ed25519
 *
 * @public
 */
export interface SigningOptions {
  /**
   * Expiry time for the signed URL, either as a Date object or ISO string
   */
  expiry: Date | string
  /**
   * The key ID to use for signing the URL
   */
  keyId: string
  /**
   * The Ed25519 private key (as hex string) to use for signing the URL
   */
  privateKey: string
}
