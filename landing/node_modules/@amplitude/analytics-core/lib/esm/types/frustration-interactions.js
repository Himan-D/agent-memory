import { __read, __spreadArray } from "tslib";
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
var DEFAULT_ERROR_AND_DEAD_CLICK_ALLOWLIST = __spreadArray([
    'input[type="button"]',
    'input[type="submit"]',
    'input[type="reset"]',
    'input[type="image"]',
    'input[type="file"]'
], __read(CLICKABLE_ELEMENT_SELECTORS), false);
/**
 * Default CSS selectors for dead clicks tracking
 */
export var DEFAULT_DEAD_CLICK_ALLOWLIST = DEFAULT_ERROR_AND_DEAD_CLICK_ALLOWLIST;
/**
 * Default CSS selectors for error tracking
 */
export var DEFAULT_ERROR_CLICK_ALLOWLIST = DEFAULT_ERROR_AND_DEAD_CLICK_ALLOWLIST;
/**
 * Default CSS selectors for rage clicks tracking
 */
export var DEFAULT_RAGE_CLICK_ALLOWLIST = ['*'];
/**
 * Default time window for dead clicks (3 seconds)
 */
export var DEFAULT_DEAD_CLICK_WINDOW_MS = 3000;
/**
 * Default time window for rage clicks (1 second)
 */
export var DEFAULT_RAGE_CLICK_WINDOW_MS = 1000;
/**
 * Default threshold for rage clicks (4 clicks)
 */
export var DEFAULT_RAGE_CLICK_THRESHOLD = 4;
/**
 * Default threshold for rage clicks to be considered out of bounds (50 pixels)
 */
export var DEFAULT_RAGE_CLICK_OUT_OF_BOUNDS_THRESHOLD = 50; // pixels
//# sourceMappingURL=frustration-interactions.js.map