const example = "image-Tb9Ew8CXIwaY6R1kjMvI0uRR-2000x3000-jpg";
function parseAssetId(ref) {
  const [, id, dimensionString, format] = ref.split("-");
  if (!id || !dimensionString || !format)
    throw new Error(`Malformed asset _ref '${ref}'. Expected an id like "${example}".`);
  const [imgWidthStr, imgHeightStr] = dimensionString.split("x"), width = +imgWidthStr, height = +imgHeightStr;
  if (!(isFinite(width) && isFinite(height)))
    throw new Error(`Malformed asset _ref '${ref}'. Expected an id like "${example}".`);
  return { id, width, height, format };
}
const isRef = (src) => {
  const source = src;
  return source ? typeof source._ref == "string" : !1;
}, isAsset = (src) => {
  const source = src;
  return source ? typeof source._id == "string" : !1;
}, isAssetStub = (src) => {
  const source = src;
  return source && source.asset ? typeof source.asset.url == "string" : !1;
}, isInProgressUpload = (src) => {
  if (typeof src == "object" && src !== null) {
    const obj = src;
    return obj._upload && (!obj.asset || !obj.asset._ref);
  }
  return !1;
};
function parseSource(source) {
  if (!source)
    return null;
  let image;
  if (typeof source == "string" && isUrl(source))
    image = {
      asset: { _ref: urlToId(source) }
    };
  else if (typeof source == "string")
    image = {
      asset: { _ref: source }
    };
  else if (isRef(source))
    image = {
      asset: source
    };
  else if (isAsset(source))
    image = {
      asset: {
        _ref: source._id || ""
      }
    };
  else if (isAssetStub(source))
    image = {
      asset: {
        _ref: urlToId(source.asset.url)
      }
    };
  else if (typeof source.asset == "object")
    image = { ...source };
  else
    return null;
  const img = source;
  return img.crop && (image.crop = img.crop), img.hotspot && (image.hotspot = img.hotspot), applyDefaults(image);
}
function isUrl(url) {
  return /^https?:\/\//.test(`${url}`);
}
function urlToId(url) {
  return `image-${url.split("/").slice(-1)[0]}`.replace(/\.([a-z]+)$/, "-$1");
}
function applyDefaults(image) {
  if (image.crop && image.hotspot)
    return image;
  const result = { ...image };
  return result.crop || (result.crop = {
    left: 0,
    top: 0,
    bottom: 0,
    right: 0
  }), result.hotspot || (result.hotspot = {
    x: 0.5,
    y: 0.5,
    height: 1,
    width: 1
  }), result;
}
const SPEC_NAME_TO_URL_NAME_MAPPINGS = [
  ["width", "w"],
  ["height", "h"],
  ["format", "fm"],
  ["download", "dl"],
  ["blur", "blur"],
  ["sharpen", "sharp"],
  ["invert", "invert"],
  ["orientation", "or"],
  ["minHeight", "min-h"],
  ["maxHeight", "max-h"],
  ["minWidth", "min-w"],
  ["maxWidth", "max-w"],
  ["quality", "q"],
  ["fit", "fit"],
  ["crop", "crop"],
  ["saturation", "sat"],
  ["auto", "auto"],
  ["dpr", "dpr"],
  ["pad", "pad"],
  ["frame", "frame"]
];
function urlForImage(options) {
  let spec = { ...options || {} };
  const source = spec.source;
  delete spec.source;
  const image = parseSource(source);
  if (!image) {
    if (source && isInProgressUpload(source))
      return "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8HwQACfsD/QNViZkAAAAASUVORK5CYII=";
    throw new Error(`Unable to resolve image URL from source (${JSON.stringify(source)})`);
  }
  const id = image.asset._ref || image.asset._id || "", asset = parseAssetId(id), cropLeft = Math.round(image.crop.left * asset.width), cropTop = Math.round(image.crop.top * asset.height), crop = {
    left: cropLeft,
    top: cropTop,
    width: Math.round(asset.width - image.crop.right * asset.width - cropLeft),
    height: Math.round(asset.height - image.crop.bottom * asset.height - cropTop)
  }, hotSpotVerticalRadius = image.hotspot.height * asset.height / 2, hotSpotHorizontalRadius = image.hotspot.width * asset.width / 2, hotSpotCenterX = image.hotspot.x * asset.width, hotSpotCenterY = image.hotspot.y * asset.height, hotspot = {
    left: hotSpotCenterX - hotSpotHorizontalRadius,
    top: hotSpotCenterY - hotSpotVerticalRadius,
    right: hotSpotCenterX + hotSpotHorizontalRadius,
    bottom: hotSpotCenterY + hotSpotVerticalRadius
  };
  return spec.rect || spec.focalPoint || spec.ignoreImageParams || spec.crop || (spec = { ...spec, ...fit({ crop, hotspot }, spec) }), specToImageUrl({ ...spec, asset });
}
function specToImageUrl(spec) {
  const cdnUrl = (spec.baseUrl || "https://cdn.sanity.io").replace(/\/+$/, ""), vanityStub = spec.vanityName ? `/${spec.vanityName}` : "", filename = `${spec.asset.id}-${spec.asset.width}x${spec.asset.height}.${spec.asset.format}${vanityStub}`;
  let baseUrl;
  spec.mediaLibraryId ? baseUrl = `${cdnUrl}/media-libraries/${spec.mediaLibraryId}/images/${filename}` : spec.canvasId ? baseUrl = `${cdnUrl}/images/canvases/${spec.canvasId}/${filename}` : baseUrl = `${cdnUrl}/images/${spec.projectId}/${spec.dataset}/${filename}`;
  const params = [];
  if (spec.rect) {
    const { left, top, width, height } = spec.rect;
    (left !== 0 || top !== 0 || height !== spec.asset.height || width !== spec.asset.width) && params.push(`rect=${left},${top},${width},${height}`);
  }
  spec.bg && params.push(`bg=${spec.bg}`), spec.focalPoint && (params.push(`fp-x=${spec.focalPoint.x}`), params.push(`fp-y=${spec.focalPoint.y}`));
  const flip = [spec.flipHorizontal && "h", spec.flipVertical && "v"].filter(Boolean).join("");
  return flip && params.push(`flip=${flip}`), SPEC_NAME_TO_URL_NAME_MAPPINGS.forEach((mapping) => {
    const [specName, param] = mapping;
    typeof spec[specName] < "u" ? params.push(`${param}=${encodeURIComponent(spec[specName])}`) : typeof spec[param] < "u" && params.push(`${param}=${encodeURIComponent(spec[param])}`);
  }), params.length === 0 ? baseUrl : `${baseUrl}?${params.join("&")}`;
}
function fit(source, spec) {
  let cropRect;
  const imgWidth = spec.width, imgHeight = spec.height;
  if (!(imgWidth && imgHeight))
    return { width: imgWidth, height: imgHeight, rect: source.crop };
  const crop = source.crop, hotspot = source.hotspot, desiredAspectRatio = imgWidth / imgHeight;
  if (crop.width / crop.height > desiredAspectRatio) {
    const height = Math.round(crop.height), width = Math.round(height * desiredAspectRatio), top = Math.max(0, Math.round(crop.top)), hotspotXCenter = Math.round((hotspot.right - hotspot.left) / 2 + hotspot.left);
    let left = Math.max(0, Math.round(hotspotXCenter - width / 2));
    left < crop.left ? left = crop.left : left + width > crop.left + crop.width && (left = crop.left + crop.width - width), cropRect = { left, top, width, height };
  } else {
    const width = crop.width, height = Math.round(width / desiredAspectRatio), left = Math.max(0, Math.round(crop.left)), hotspotYCenter = Math.round((hotspot.bottom - hotspot.top) / 2 + hotspot.top);
    let top = Math.max(0, Math.round(hotspotYCenter - height / 2));
    top < crop.top ? top = crop.top : top + height > crop.top + crop.height && (top = crop.top + crop.height - height), cropRect = { left, top, width, height };
  }
  return {
    width: imgWidth,
    height: imgHeight,
    rect: cropRect
  };
}
const validFits = ["clip", "crop", "fill", "fillmax", "max", "scale", "min"], validCrops = ["top", "bottom", "left", "right", "center", "focalpoint", "entropy"], validAutoModes = ["format"];
function isSanityModernClientLike(client) {
  return client && "config" in client ? typeof client.config == "function" : !1;
}
function isSanityClientLike(client) {
  return client && "clientConfig" in client ? typeof client.clientConfig == "object" : !1;
}
function clientConfigToOptions(config) {
  const { apiHost: apiUrl, projectId, dataset } = config, baseOptions = {
    baseUrl: (apiUrl || "https://api.sanity.io").replace(/^https:\/\/api\./, "https://cdn.")
  }, resource = config.resource ?? config["~experimental_resource"];
  if (resource?.type === "media-library") {
    if (typeof resource.id != "string" || resource.id.length === 0)
      throw new Error('Media library clients must include an id in "resource"');
    return { ...baseOptions, mediaLibraryId: resource.id };
  }
  if (resource?.type === "canvas") {
    if (typeof resource.id != "string" || resource.id.length === 0)
      throw new Error('Canvas clients must include an id in "resource"');
    return { ...baseOptions, canvasId: resource.id };
  }
  if (resource?.type === "dataset") {
    if (typeof resource.id != "string" || resource.id.length === 0)
      throw new Error('Dataset clients must include an id in "resource"');
    const [resourceProjectId, resourceDataset] = resource.id.split(".");
    if (!resourceProjectId || !resourceDataset)
      throw new Error(
        'Dataset resource id must be in the format "projectId.dataset", got: ' + resource.id
      );
    return { ...baseOptions, projectId: resourceProjectId, dataset: resourceDataset };
  }
  return { ...baseOptions, projectId, dataset };
}
function rewriteSpecName(key) {
  const specs = SPEC_NAME_TO_URL_NAME_MAPPINGS;
  for (const entry of specs) {
    const [specName, param] = entry;
    if (key === specName || key === param)
      return specName;
  }
  return key;
}
function getOptions(_options) {
  let options = {};
  return isSanityModernClientLike(_options) ? options = clientConfigToOptions(_options.config()) : isSanityClientLike(_options) ? options = clientConfigToOptions(_options.clientConfig) : options = _options || {}, options;
}
function createBuilder(Builder, _options) {
  const options = getOptions(_options);
  return new Builder(null, options);
}
function createImageUrlBuilder(options) {
  return createBuilder(ImageUrlBuilderImpl, options);
}
function constructNewOptions(currentOptions, options) {
  const baseUrl = options.baseUrl || currentOptions.baseUrl, newOptions = { baseUrl };
  for (const key in options)
    if (options.hasOwnProperty(key)) {
      const specKey = rewriteSpecName(key);
      newOptions[specKey] = options[key];
    }
  return { baseUrl, ...newOptions };
}
class ImageUrlBuilderImpl {
  options;
  constructor(parent, options) {
    this.options = parent ? { ...parent.options || {}, ...options || {} } : { ...options || {} };
  }
  withOptions(options) {
    const newOptions = constructNewOptions(this.options, options);
    return new ImageUrlBuilderImpl(this, newOptions);
  }
  // The image to be represented. Accepts a Sanity 'image'-document, 'asset'-document or
  // _id of asset. To get the benefit of automatic hot-spot/crop integration with the content
  // studio, the 'image'-document must be provided.
  image(source) {
    return this.withOptions({ source });
  }
  // Specify the dataset
  dataset(dataset) {
    return this.withOptions({ dataset });
  }
  // Specify the projectId
  projectId(projectId) {
    return this.withOptions({ projectId });
  }
  withClient(client) {
    const newOptions = getOptions(client), preservedOptions = { ...this.options };
    return delete preservedOptions.baseUrl, delete preservedOptions.projectId, delete preservedOptions.dataset, delete preservedOptions.mediaLibraryId, delete preservedOptions.canvasId, new ImageUrlBuilderImpl(null, { ...newOptions, ...preservedOptions });
  }
  // Specify background color
  bg(bg) {
    return this.withOptions({ bg });
  }
  // Set DPR scaling factor
  dpr(dpr) {
    return this.withOptions(dpr && dpr !== 1 ? { dpr } : {});
  }
  // Specify the width of the image in pixels
  width(width) {
    return this.withOptions({ width });
  }
  // Specify the height of the image in pixels
  height(height) {
    return this.withOptions({ height });
  }
  // Specify focal point in fraction of image dimensions. Each component 0.0-1.0
  focalPoint(x, y) {
    return this.withOptions({ focalPoint: { x, y } });
  }
  maxWidth(maxWidth) {
    return this.withOptions({ maxWidth });
  }
  minWidth(minWidth) {
    return this.withOptions({ minWidth });
  }
  maxHeight(maxHeight) {
    return this.withOptions({ maxHeight });
  }
  minHeight(minHeight) {
    return this.withOptions({ minHeight });
  }
  // Specify width and height in pixels
  size(width, height) {
    return this.withOptions({ width, height });
  }
  // Specify blur between 0 and 100
  blur(blur) {
    return this.withOptions({ blur });
  }
  sharpen(sharpen) {
    return this.withOptions({ sharpen });
  }
  // Specify the desired rectangle of the image
  rect(left, top, width, height) {
    return this.withOptions({ rect: { left, top, width, height } });
  }
  // Specify the image format of the image. 'jpg', 'pjpg', 'png', 'webp'
  format(format) {
    return this.withOptions({ format });
  }
  invert(invert) {
    return this.withOptions({ invert });
  }
  // Rotation in degrees 0, 90, 180, 270
  orientation(orientation) {
    return this.withOptions({ orientation });
  }
  // Compression quality 0-100
  quality(quality) {
    return this.withOptions({ quality });
  }
  // Make it a download link. Parameter is default filename.
  forceDownload(download) {
    return this.withOptions({ download });
  }
  // Flip image horizontally
  flipHorizontal() {
    return this.withOptions({ flipHorizontal: !0 });
  }
  // Flip image vertically
  flipVertical() {
    return this.withOptions({ flipVertical: !0 });
  }
  // Ignore crop/hotspot from image record, even when present
  ignoreImageParams() {
    return this.withOptions({ ignoreImageParams: !0 });
  }
  fit(value) {
    if (validFits.indexOf(value) === -1)
      throw new Error(`Invalid fit mode "${value}"`);
    return this.withOptions({ fit: value });
  }
  crop(value) {
    if (validCrops.indexOf(value) === -1)
      throw new Error(`Invalid crop mode "${value}"`);
    return this.withOptions({ crop: value });
  }
  // Saturation
  saturation(saturation) {
    return this.withOptions({ saturation });
  }
  auto(value) {
    if (validAutoModes.indexOf(value) === -1)
      throw new Error(`Invalid auto mode "${value}"`);
    return this.withOptions({ auto: value });
  }
  // Specify the number of pixels to pad the image
  pad(pad) {
    return this.withOptions({ pad });
  }
  // Vanity URL for more SEO friendly URLs
  vanityName(value) {
    return this.withOptions({ vanityName: value });
  }
  frame(frame) {
    if (frame !== 1)
      throw new Error(`Invalid frame value "${frame}"`);
    return this.withOptions({ frame });
  }
  // Gets the url based on the submitted parameters
  url() {
    return urlForImage(this.options);
  }
  // Alias for url()
  toString() {
    return this.url();
  }
}
function once(fn) {
  let didCall = !1, returnValue;
  return (...args) => (didCall || (returnValue = fn(...args), didCall = !0), returnValue);
}
const createWarningPrinter = (message) => once((...args) => {
  console.warn(message.join(" "), ...args);
}), printNoDefaultExport = createWarningPrinter([
  "The default export of @sanity/image-url has been deprecated. Use the named export `createImageUrlBuilder` instead."
]);
function defineDeprecated(createImageUrlBuilder2) {
  return function(options) {
    return printNoDefaultExport(), createImageUrlBuilder2(options);
  };
}
export {
  ImageUrlBuilderImpl,
  constructNewOptions,
  createBuilder,
  createImageUrlBuilder,
  defineDeprecated,
  urlForImage
};
//# sourceMappingURL=compat.js.map
