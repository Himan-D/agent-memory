# @sanity/signed-urls

A low-level library for generating Ed25519-signed URLs.

If you're working with images, you'll likely want to use [`@sanity/image-url`](https://github.com/sanity-io/image-url) instead, which provides a more convenient way to build and sign image URLs. Use this library directly if you need to sign other types of URLs or want full control over the signing process.

## Installation

```bash
npm install @sanity/signed-urls
# or
pnpm add @sanity/signed-urls
# or
yarn add @sanity/signed-urls
```

## Usage

### Basic URL Signing

```typescript
import {signUrl} from '@sanity/signed-urls'

const signedUrl = signUrl('https://cdn.sanity.io/images/project/dataset/image-id-100x100.jpg', {
  keyId: 'your-key-id',
  privateKey: 'your-ed25519-private-key-as-hex',
  expiry: '2026-12-31T23:59:59Z',
})

console.log(signedUrl)
// https://cdn.sanity.io/images/project/dataset/image-id-100x100.jpg?keyid=your-key-id&expiry=2026-12-31T23%3A59%3A59Z&signature=base64url-encoded-signature
```

### Signing with Date Objects

```typescript
import {signUrl} from '@sanity/signed-urls'

const expiryDate = new Date()
expiryDate.setHours(expiryDate.getHours() + 24) // Expires in 24 hours

const signedUrl = signUrl('https://cdn.sanity.io/images/project/dataset/image-id-100x100.jpg', {
  keyId: 'your-key-id',
  privateKey: 'your-ed25519-private-key-as-hex',
  expiry: expiryDate,
})
```

### Generating Signatures Only

```typescript
import {generateSignature} from '@sanity/signed-urls'

const signature = generateSignature(
  'https://cdn.sanity.io/images/project/dataset/image-id-100x100.jpg?keyid=your-key-id&expiry=2026-12-31T23%3A59%3A59Z',
  'your-ed25519-private-key-as-hex',
)

console.log(signature) // base64url-encoded signature
```

### URL Objects

The library accepts both string URLs and URL objects:

```typescript
import {signUrl} from '@sanity/signed-urls'

const url = new URL('https://cdn.sanity.io/images/project/dataset/image-id-100x100.jpg')
const signedUrl = signUrl(url, {
  keyId: 'your-key-id',
  privateKey: 'your-ed25519-private-key-as-hex',
  expiry: '2026-12-31T23:59:59Z',
})
```

## API Reference

### `signUrl(url, options)`

Signs a URL with Ed25519 signature.

**Parameters:**

- `url` (`string | URL`) - The URL to sign
- `options` (`SigningOptions`) - The signing options

**Returns:** `string` - The signed URL with keyid, expiry (if provided), and signature parameters

### `generateSignature(url, privateKey)`

Generates an Ed25519 signature for a given URL.

**Parameters:**

- `url` (`string | URL`) - The URL to sign
- `privateKey` (`string`) - The Ed25519 private key as a hex string

**Returns:** `string` - The base64url-encoded signature

### `SigningOptions`

```typescript
interface SigningOptions {
  /**
   * The key ID to use for signing the URL
   */
  keyId: string

  /**
   * The Ed25519 private key (as hex string) to use for signing the URL
   */
  privateKey: string

  /**
   * Expiry time for the signed URL, either as a Date object or ISO string
   */
  expiry: Date | string
}
```

## License

MIT © [Sanity.io](https://sanity.io)
