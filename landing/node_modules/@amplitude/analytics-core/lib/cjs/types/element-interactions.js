"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.DEFAULT_EXPOSURE_DURATION = exports.DEFAULT_ACTION_CLICK_ALLOWLIST = exports.DEFAULT_DATA_ATTRIBUTE_PREFIX = exports.DEFAULT_CSS_SELECTOR_ALLOWLIST = void 0;
/**
 * Default CSS selectors to define which elements on the page to track.
 * Extend this list to include additional elements to track. For example:
 * ```
 * autocapturePlugin({
 *    cssSelectorAllowlist: [...DEFAULT_CSS_SELECTOR_ALLOWLIST, ".my-class"],
 * })
 * ```
 */
exports.DEFAULT_CSS_SELECTOR_ALLOWLIST = [
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
exports.DEFAULT_DATA_ATTRIBUTE_PREFIX = 'data-amp-track-';
/**
 * Default list of elements on the page should be tracked when the page changes.
 */
exports.DEFAULT_ACTION_CLICK_ALLOWLIST = ['div', 'span', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6'];
exports.DEFAULT_EXPOSURE_DURATION = 150;
//# sourceMappingURL=element-interactions.js.map