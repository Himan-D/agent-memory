"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.DEFAULT_RAGE_CLICK_OUT_OF_BOUNDS_THRESHOLD = exports.DEFAULT_RAGE_CLICK_THRESHOLD = exports.DEFAULT_RAGE_CLICK_WINDOW_MS = exports.DEFAULT_DEAD_CLICK_WINDOW_MS = exports.DEFAULT_RAGE_CLICK_ALLOWLIST = exports.DEFAULT_ERROR_CLICK_ALLOWLIST = exports.DEFAULT_DEAD_CLICK_ALLOWLIST = void 0;
var tslib_1 = require("tslib");
var CLICKABLE_ELEMENT_SELECTORS = [
    'a',
    'button',
    '[role="button"]',
    '[role="link"]',
    '[role="menuitem"]',
    '[role="menuitemcheckbox"]',
    '[role="menuitemradio"]',
    '[role="option"]',
    '[role="tab"]',
    '[role="treeitem"]',
    '[contenteditable="true" i]',
];
var DEFAULT_ERROR_AND_DEAD_CLICK_ALLOWLIST = tslib_1.__spreadArray([
    'input[type="button"]',
    'input[type="submit"]',
    'input[type="reset"]',
    'input[type="image"]',
    'input[type="file"]'
], tslib_1.__read(CLICKABLE_ELEMENT_SELECTORS), false);
/**
 * Default CSS selectors for dead clicks tracking
 */
exports.DEFAULT_DEAD_CLICK_ALLOWLIST = DEFAULT_ERROR_AND_DEAD_CLICK_ALLOWLIST;
/**
 * Default CSS selectors for error tracking
 */
exports.DEFAULT_ERROR_CLICK_ALLOWLIST = DEFAULT_ERROR_AND_DEAD_CLICK_ALLOWLIST;
/**
 * Default CSS selectors for rage clicks tracking
 */
exports.DEFAULT_RAGE_CLICK_ALLOWLIST = ['*'];
/**
 * Default time window for dead clicks (3 seconds)
 */
exports.DEFAULT_DEAD_CLICK_WINDOW_MS = 3000;
/**
 * Default time window for rage clicks (1 second)
 */
exports.DEFAULT_RAGE_CLICK_WINDOW_MS = 1000;
/**
 * Default threshold for rage clicks (4 clicks)
 */
exports.DEFAULT_RAGE_CLICK_THRESHOLD = 4;
/**
 * Default threshold for rage clicks to be considered out of bounds (50 pixels)
 */
exports.DEFAULT_RAGE_CLICK_OUT_OF_BOUNDS_THRESHOLD = 50; // pixels
//# sourceMappingURL=frustration-interactions.js.map