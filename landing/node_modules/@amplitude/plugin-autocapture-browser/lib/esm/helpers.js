import { __read } from "tslib";
/* eslint-disable no-restricted-globals */
import { isUrlMatchAllowlist, getGlobalScope } from '@amplitude/analytics-core';
import * as constants from './constants';
var SENSITIVE_TAGS = ['input', 'select', 'textarea'];
export var isElementPointerCursor = function (element, actionType) {
    var _a;
    /* istanbul ignore next */
    var computedStyle = (_a = window === null || window === void 0 ? void 0 : window.getComputedStyle) === null || _a === void 0 ? void 0 : _a.call(window, element);
    /* istanbul ignore next */
    return (computedStyle === null || computedStyle === void 0 ? void 0 : computedStyle.getPropertyValue('cursor')) === 'pointer' && actionType === 'click';
};
export var isUrlAllowed = function (autocaptureOptions) {
    var pageUrlAllowlist = autocaptureOptions.pageUrlAllowlist, pageUrlExcludelist = autocaptureOptions.pageUrlExcludelist;
    // check if the URL is in the excludelist
    if (pageUrlExcludelist &&
        pageUrlExcludelist.length > 0 &&
        isUrlMatchAllowlist(window.location.href, pageUrlExcludelist)) {
        return false;
    }
    // check if the URL is in the allow list
    if (!isUrlMatchAllowlist(window.location.href, pageUrlAllowlist)) {
        return false;
    }
    return true;
};
export var createShouldTrackEvent = function (autocaptureOptions, allowlist, // this can be any type of css selector allow list
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
        if (!isUrlAllowed(autocaptureOptions)) {
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
        var isCursorPointer = isElementPointerCursor(element, actionType);
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
export var isTextNode = function (node) {
    return !!node && node.nodeType === 3;
};
export var isNonSensitiveElement = function (element) {
    var _a, _b, _c;
    /* istanbul ignore next */
    var tag = (_b = (_a = element === null || element === void 0 ? void 0 : element.tagName) === null || _a === void 0 ? void 0 : _a.toLowerCase) === null || _b === void 0 ? void 0 : _b.call(_a);
    var isContentEditable = element instanceof HTMLElement ? ((_c = element.getAttribute('contenteditable')) === null || _c === void 0 ? void 0 : _c.toLowerCase()) === 'true' : false;
    return !SENSITIVE_TAGS.includes(tag) && !isContentEditable;
};
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
export var parseAttributesToMask = function (attributeString) {
    return attributeString
        ? attributeString
            .split(',')
            .map(function (attr) { return attr.trim(); })
            .filter(function (attr) { return attr.length > 0 && attr !== 'id' && attr !== 'class'; }) // Prevent 'id' and 'class' from being redacted as they're critical for element identification
        : [];
};
export var extractPrefixedAttributes = function (attrs, prefix) {
    return Object.entries(attrs).reduce(function (attributes, _a) {
        var _b = __read(_a, 2), attributeName = _b[0], attributeValue = _b[1];
        if (attributeName.startsWith(prefix)) {
            var attributeKey = attributeName.replace(prefix, '');
            if (attributeKey) {
                attributes[attributeKey] = attributeValue || '';
            }
        }
        return attributes;
    }, {});
};
export var isEmpty = function (value) {
    return (value === undefined ||
        value === null ||
        (typeof value === 'object' && Object.keys(value).length === 0) ||
        (typeof value === 'string' && value.trim().length === 0));
};
export var removeEmptyProperties = function (properties) {
    return Object.keys(properties).reduce(function (filteredProperties, key) {
        var value = properties[key];
        if (!isEmpty(value)) {
            filteredProperties[key] = value;
        }
        return filteredProperties;
    }, {});
};
export var getCurrentPageViewId = function () {
    var _a;
    try {
        var globalScope = getGlobalScope();
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
export var querySelectUniqueElements = function (root, selectors) {
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
// Similar as element.closest, but works with multiple selectors
export var getClosestElement = function (element, selectors) {
    if (!element) {
        return null;
    }
    /* istanbul ignore next */
    if (selectors.some(function (selector) { var _a; return (_a = element === null || element === void 0 ? void 0 : element.matches) === null || _a === void 0 ? void 0 : _a.call(element, selector); })) {
        return element;
    }
    /* istanbul ignore next */
    return getClosestElement(element === null || element === void 0 ? void 0 : element.parentElement, selectors);
};
export var asyncLoadScript = function (url) {
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
export function generateUniqueId() {
    return "".concat(Date.now(), "-").concat(Math.random().toString(36).substr(2, 9));
}
export var filterOutNonTrackableEvents = function (event) {
    // Filter out changeEvent events with no target
    // This could happen when change events are triggered programmatically
    if (event.event.target === null || !event.closestTrackedAncestor) {
        return false;
    }
    return true;
};
// Type predicate
export function isElementBasedEvent(event) {
    return event.type === 'click' || event.type === 'change';
}
export var MouseButton;
(function (MouseButton) {
    MouseButton[MouseButton["LEFT_OR_TOUCH_CONTACT"] = 0] = "LEFT_OR_TOUCH_CONTACT";
    MouseButton[MouseButton["MIDDLE"] = 1] = "MIDDLE";
    MouseButton[MouseButton["RIGHT"] = 2] = "RIGHT";
})(MouseButton || (MouseButton = {}));
//# sourceMappingURL=helpers.js.map