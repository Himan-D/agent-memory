import { urlForImage, createBuilder, ImageUrlBuilderImpl, constructNewOptions, defineDeprecated } from "../_chunks-es/compat.js";
import { signUrl } from "@sanity/signed-urls";
function signedUrlForImage(options, signingOptions) {
  const baseUrl = urlForImage(options);
  return signUrl(baseUrl, signingOptions);
}
function assertValidSignedOptions(opts) {
  if (!opts.keyId || typeof opts.keyId != "string")
    throw new Error("Cannot call `signedUrl()` without `keyId`");
  if (!opts.privateKey || typeof opts.privateKey != "string")
    throw new Error("Cannot call `signedUrl()` without `privateKey`");
  if (!opts.expiry || typeof opts.expiry != "string" && !(opts.expiry instanceof Date))
    throw new Error("Cannot call `signedUrl()` without `expiry`");
}
class SignedImageUrlBuilderImpl extends ImageUrlBuilderImpl {
  constructor(parent, options) {
    super(parent, options);
  }
  withOptions(options) {
    const newOptions = constructNewOptions(this.options, options);
    return new SignedImageUrlBuilderImpl(this, { ...newOptions });
  }
  expiry(expiry) {
    return this.withOptions({ expiry });
  }
  signingKey(keyId, privateKey) {
    return this.withOptions({ keyId, privateKey });
  }
  signedUrl() {
    const { expiry, keyId, privateKey, ...rest } = this.options, signedOptions = { expiry, keyId, privateKey };
    return assertValidSignedOptions(signedOptions), signedUrlForImage(rest, signedOptions);
  }
}
function createImageUrlBuilder(options) {
  return createBuilder(SignedImageUrlBuilderImpl, options);
}
const deprecatedcreateImageUrlBuilder = defineDeprecated(createImageUrlBuilder);
export {
  createImageUrlBuilder,
  deprecatedcreateImageUrlBuilder as default
};
//# sourceMappingURL=index.js.map
