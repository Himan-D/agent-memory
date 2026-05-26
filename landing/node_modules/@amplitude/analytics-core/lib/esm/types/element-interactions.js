/**
 * Default CSS selectors to define which elements on the page to track.
 * Extend this list to include additional elements to track. For example:
 * ```
 * autocapturePlugin({
 *    cssSelectorAllowlist: [...DEFAULT_CSS_SELECTOR_ALLOWLIST, ".my-class"],
 * })
 * ```
 */
export var DEFAULT_CSS_SELECTOR_ALLOWLIST = [
    'a',
    'button',
    'input',
    'select',
    'textarea',
    'label',
    'video',
    'audio',
    '[contenteditable="true" i]',
    '[data-amp-default-track]',
    '.amp-default-track',
];
/**
 * Default prefix to allow the plugin to capture data attributes as an event property.
 */
export var DEFAULT_DATA_ATTRIBUTE_PREFIX = 'data-amp-track-';
/**
 * Default list of elements on the page should be tracked when the page changes.
 */
export var DEFAULT_ACTION_CLICK_ALLOWLIST = ['div', 'span', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6'];
export var DEFAULT_EXPOSURE_DURATION = 150;
//# sourceMappingURL=element-interactions.js.map