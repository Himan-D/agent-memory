"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.MouseButton = exports.isElementBasedEvent = exports.filterOutNonTrackableEvents = exports.generateUniqueId = exports.asyncLoadScript = exports.getClosestElement = exports.querySelectUniqueElements = exports.getCurrentPageViewId = exports.removeEmptyProperties = exports.isEmpty = exports.extractPrefixedAttributes = exports.parseAttributesToMask = exports.isNonSensitiveElement = exports.isTextNode = exports.createShouldTrackEvent = exports.isUrlAllowed = exports.isElementPointerCursor = void 0;
var tslib_1 = require("tslib");
/* eslint-disable no-restricted-globals */
var analytics_core_1 = require("@amplitude/analytics-core");
var constants = tslib_1.__importStar(require("./constants"));
var SENSITIVE_TAGS = ['input', 'select', 'textarea'];
var isElementPointerCursor = function (element, actionType) {
    var _a;
    /* istanbul ignore next */
    var computedStyle = (_a = window === null || window === void 0 ? void 0 : window.getComputedStyle) === null || _a === void 0 ? void 0 : _a.call(window, element);
    /* istanbul ignore next */
    return (computedStyle === null || computedStyle === void 0 ? void 0 : computedStyle.getPropertyValue('cursor')) === 'pointer' && actionType === 'click';
};
exports.isElementPointerCursor = isElementPointerCursor;
var isUrlAllowed = function (autocaptureOptions) {
    var pageUrlAllowlist = autocaptureOptions.pageUrlAllowlist, pageUrlExcludelist = autocaptureOptions.pageUrlExcludelist;
    // check if the URL is in the excludelist
    if (pageUrlExcludelist &&
        pageUrlExcludelist.length > 0 &&
        (0, analytics_core_1.isUrlMatchAllowlist)(window.location.href, pageUrlExcludelist)) {
        return false;
    }
    // check if the URL is in the allow list
    if (!(0, analytics_core_1.isUrlMatchAllowlist)(window.location.href, pageUrlAllowlist)) {
        return false;
    }
    return true;
};
exports.isUrlAllowed = isUrlAllowed;
var createShouldTrackEvent = function (autocaptureOptions, allowlist, // this can be any type of css selector allow list
isAlwaysCaptureCursorPointer) {
    if (isAlwaysCaptureCursorPointer === void 0) { isAlwaysCaptureCursorPointer = false; }
    return function (actionType, element) {
        var _a, _b;
        var shouldTrackEventResolver = autocaptureOptions.shouldTrackEventResolver;
        /* istanbul ignore next */
        var tag = (_b = (_a = element === null || element === void 0 ? void 0 : element.tagName) === null || _a === void 0 ? void 0 : _a.toLowerCase) === null || _b === void 0 ? void 0 : _b.call(_a);
        // window, document, and Text nodes have no tag
        if (!tag) {
            return false;
        }
        if (shouldTrackEventResolver) {
            return shouldTrackEventResolver(actionType, element);
        }
        if (!(0, exports.isUrlAllowed)(autocaptureOptions)) {
            return false;
        }
        /* istanbul ignore next */
        var elementType = String(element === null || element === void 0 ? void 0 : element.getAttribute('type')) || '';
        if (typeof elementType === 'string') {
            switch (elementType.toLowerCase()) {
                case 'hidden':
                    return false;
                case 'password':
                    return false;
            }
        }
        var isCursorPointer = (0, exports.isElementPointerCursor)(element, actionType);
        if (isAlwaysCaptureCursorPointer && isCursorPointer) {
            return true;
        }
        /* istanbul ignore if */
        if (allowlist) {
            var hasMatchAnyAllowedSelector = allowlist.some(function (selector) { var _a; return !!((_a = element === null || element === void 0 ? void 0 : element.matches) === null || _a === void 0 ? void 0 : _a.call(element, selector)); });
            if (!hasMatchAnyAllowedSelector) {
                return false;
            }
        }
        switch (tag) {
            case 'input':
            case 'select':
            case 'textarea':
                return actionType === 'change' || actionType === 'click';
            default: {
                /* istanbul ignore next */
                /* istanbul ignore next */
                if (isCursorPointer) {
                    return true;
                }
                return actionType === 'click';
            }
        }
    };
};
exports.createShouldTrackEvent = createShouldTrackEvent;
var isTextNode = function (node) {
    return !!node && node.nodeType === 3;
};
exports.isTextNode = isTextNode;
var isNonSensitiveElement = function (element) {
    var _a, _b, _c;
    /* istanbul ignore next */
    var tag = (_b = (_a = element === null || element === void 0 ? void 0 : element.tagName) === null || _a === void 0 ? void 0 : _a.toLowerCase) === null || _b === void 0 ? void 0 : _b.call(_a);
    var isContentEditable = element instanceof HTMLElement ? ((_c = element.getAttribute('contenteditable')) === null || _c === void 0 ? void 0 : _c.toLowerCase()) === 'true' : false;
    return !SENSITIVE_TAGS.includes(tag) && !isContentEditable;
};
exports.isNonSensitiveElement = isNonSensitiveElement;
/**
 * Collects redacted attribute names from element and ancestor elements with data-amp-mask-attributes
 * The 'id' and 'class' attributes cannot be redacted as they're critical for element identification
 * @param element - The target element to check for redaction attributes
 * @returns Set of attribute names that should be redacted
 */
/**
 * Parses a comma-separated string of attribute names and filters out protected attributes
 * @param attributeString - Comma-separated string of attribute names
 * @returns Array of valid attribute names, excluding 'id' and 'class'
 */
var parseAttributesToMask = function (attributeString) {
    return attributeString
        ? attributeString
            .split(',')
            .map(function (attr) { return attr.trim(); })
            .filter(function (attr) { return attr.length > 0 && attr !== 'id' && attr !== 'class'; }) // Prevent 'id' and 'class' from being redacted as they're critical for element identification
        : [];
};
exports.parseAttributesToMask = parseAttributesToMask;
var extractPrefixedAttributes = function (attrs, prefix) {
    return Object.entries(attrs).reduce(function (attributes, _a) {
        var _b = tslib_1.__read(_a, 2), attributeName = _b[0], attributeValue = _b[1];
        if (attributeName.startsWith(prefix)) {
            var attributeKey = attributeName.replace(prefix, '');
            if (attributeKey) {
                attributes[attributeKey] = attributeValue || '';
            }
        }
        return attributes;
    }, {});
};
exports.extractPrefixedAttributes = extractPrefixedAttributes;
var isEmpty = function (value) {
    return (value === undefined ||
        value === null ||
        (typeof value === 'object' && Object.keys(value).length === 0) ||
        (typeof value === 'string' && value.trim().length === 0));
};
exports.isEmpty = isEmpty;
var removeEmptyProperties = function (properties) {
    return Object.keys(properties).reduce(function (filteredProperties, key) {
        var value = properties[key];
        if (!(0, exports.isEmpty)(value)) {
            filteredProperties[key] = value;
        }
        return filteredProperties;
    }, {});
};
exports.removeEmptyProperties = removeEmptyProperties;
var getCurrentPageViewId = function () {
    var _a;
    try {
        var globalScope = (0, analytics_core_1.getGlobalScope)();
        /* istanbul ignore next */
        var raw = (_a = globalScope === null || globalScope === void 0 ? void 0 : globalScope.sessionStorage) === null || _a === void 0 ? void 0 : _a.getItem(constants.PAGE_VIEW_SESSION_STORAGE_KEY);
        if (!raw) {
            return undefined;
        }
        var parsed = JSON.parse(raw);
        if (typeof parsed.pageViewId === 'string') {
            return parsed.pageViewId;
        }
    }
    catch (_b) {
        // ignore storage access or JSON errors
    }
    return undefined;
};
exports.getCurrentPageViewId = getCurrentPageViewId;
var querySelectUniqueElements = function (root, selectors) {
    if (root && 'querySelectorAll' in root && typeof root.querySelectorAll === 'function') {
        var elementSet = selectors.reduce(function (elements, selector) {
            if (selector) {
                var selectedElements = Array.from(root.querySelectorAll(selector));
                selectedElements.forEach(function (element) {
                    elements.add(element);
                });
            }
            return elements;
        }, new Set());
        return Array.from(elementSet);
    }
    return [];
};
exports.querySelectUniqueElements = querySelectUniqueElements;
// Similar as element.closest, but works with multiple selectors
var getClosestElement = function (element, selectors) {
    if (!element) {
        return null;
    }
    /* istanbul ignore next */
    if (selectors.some(function (selector) { var _a; return (_a = element === null || element === void 0 ? void 0 : element.matches) === null || _a === void 0 ? void 0 : _a.call(element, selector); })) {
        return element;
    }
    /* istanbul ignore next */
    return (0, exports.getClosestElement)(element === null || element === void 0 ? void 0 : element.parentElement, selectors);
};
exports.getClosestElement = getClosestElement;
var asyncLoadScript = function (url) {
    return new Promise(function (resolve, reject) {
        var _a;
        try {
            var scriptElement = document.createElement('script');
            scriptElement.type = 'text/javascript';
            scriptElement.async = true;
            scriptElement.src = url;
            scriptElement.addEventListener('load', function () {
                resolve({ status: true });
            }, { once: true });
            scriptElement.addEventListener('error', function () {
                reject({
                    status: false,
                    message: "Failed to load the script ".concat(url),
                });
            });
            /* istanbul ignore next */
            (_a = document.head) === null || _a === void 0 ? void 0 : _a.appendChild(scriptElement);
        }
        catch (error) {
            /* istanbul ignore next */
            reject(error);
        }
    });
};
exports.asyncLoadScript = asyncLoadScript;
function generateUniqueId() {
    return "".concat(Date.now(), "-").concat(Math.random().toString(36).substr(2, 9));
}
exports.generateUniqueId = generateUniqueId;
var filterOutNonTrackableEvents = function (event) {
    // Filter out changeEvent events with no target
    // This could happen when change events are triggered programmatically
    if (event.event.target === null || !event.closestTrackedAncestor) {
        return false;
    }
    return true;
};
exports.filterOutNonTrackableEvents = filterOutNonTrackableEvents;
// Type predicate
function isElementBasedEvent(event) {
    return event.type === 'click' || event.type === 'change';
}
exports.isElementBasedEvent = isElementBasedEvent;
var MouseButton;
(function (MouseButton) {
    MouseButton[MouseButton["LEFT_OR_TOUCH_CONTACT"] = 0] = "LEFT_OR_TOUCH_CONTACT";
    MouseButton[MouseButton["MIDDLE"] = 1] = "MIDDLE";
    MouseButton[MouseButton["RIGHT"] = 2] = "RIGHT";
})(MouseButton = exports.MouseButton || (exports.MouseButton = {}));
//# sourceMappingURL=helpers.js.map