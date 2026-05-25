"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.DataExtractor = void 0;
var tslib_1 = require("tslib");
/* eslint-disable no-restricted-globals */
var analytics_core_1 = require("@amplitude/analytics-core");
var constants = tslib_1.__importStar(require("./constants"));
var helpers_1 = require("./helpers");
var hierarchy_1 = require("./hierarchy");
var actions_1 = require("./pageActions/actions");
var element_path_1 = require("./libs/element-path");
var DataExtractor = /** @class */ (function () {
    function DataExtractor(options, context) {
        var e_1, _a;
        var _this = this;
        var _b;
        /**
         * Wrapper method to replace sensitive strings using the helper function
         * @param text - The text to search for sensitive data
         * @returns The text with sensitive data replaced by masked text
         */
        this.replaceSensitiveString = function (text) {
            return (0, analytics_core_1.replaceSensitiveString)(text, _this.additionalMaskTextPatterns);
        };
        // Get the DOM hierarchy of the element, starting from the target element to the root element.
        this.getHierarchy = function (element) {
            var e_2, _a;
            var _b, _c;
            var startTime = performance.now();
            var hierarchy = [];
            if (!element) {
                return [];
            }
            // Get list of ancestors including itself and get properties at each level in the hierarchy
            var ancestors = (0, hierarchy_1.getAncestors)(element);
            // Build attributes to mask map
            var elementToAttributesToMaskMap = new Map();
            for (var i = ancestors.length - 1; i >= 0; i--) {
                var node = ancestors[i];
                if (node) {
                    var attributesToMask = (0, helpers_1.parseAttributesToMask)(node.getAttribute(constants.DATA_AMP_MASK_ATTRIBUTES));
                    var ancestorAttributesToMask = i === ancestors.length - 1 ? [] : (_b = elementToAttributesToMaskMap.get(ancestors[i + 1])) !== null && _b !== void 0 ? _b : new Set();
                    var combinedAttributesToMask = new Set(tslib_1.__spreadArray(tslib_1.__spreadArray([], tslib_1.__read(ancestorAttributesToMask), false), tslib_1.__read(attributesToMask), false));
                    elementToAttributesToMaskMap.set(node, combinedAttributesToMask);
                }
            }
            hierarchy = ancestors.map(function (el) { var _a; return (0, hierarchy_1.getElementProperties)(el, (_a = elementToAttributesToMaskMap.get(el)) !== null && _a !== void 0 ? _a : new Set()); });
            var _loop_1 = function (hierarchyNode) {
                if (hierarchyNode === null || hierarchyNode === void 0 ? void 0 : hierarchyNode.attrs) {
                    Object.entries(hierarchyNode.attrs).forEach(function (_a) {
                        var _b = tslib_1.__read(_a, 2), key = _b[0], value = _b[1];
                        if (hierarchyNode.attrs) {
                            hierarchyNode.attrs[key] = _this.replaceSensitiveString(value);
                        }
                    });
                }
            };
            try {
                // Search for and mask any sensitive attribute values
                for (var hierarchy_2 = tslib_1.__values(hierarchy), hierarchy_2_1 = hierarchy_2.next(); !hierarchy_2_1.done; hierarchy_2_1 = hierarchy_2.next()) {
                    var hierarchyNode = hierarchy_2_1.value;
                    _loop_1(hierarchyNode);
                }
            }
            catch (e_2_1) { e_2 = { error: e_2_1 }; }
            finally {
                try {
                    if (hierarchy_2_1 && !hierarchy_2_1.done && (_a = hierarchy_2.return)) _a.call(hierarchy_2);
                }
                finally { if (e_2) throw e_2.error; }
            }
            var endTime = performance.now();
            (_c = _this.diagnosticsClient) === null || _c === void 0 ? void 0 : _c.recordHistogram('autocapturePlugin.getHierarchy', endTime - startTime);
            return hierarchy;
        };
        this.getNearestLabel = function (element) {
            var parent = element.parentElement;
            if (!parent) {
                return '';
            }
            var labelElement;
            try {
                labelElement = parent.querySelector(':scope>span,h1,h2,h3,h4,h5,h6');
            }
            catch (_a) {
                /* istanbul ignore next */
                labelElement = null;
            }
            if (labelElement) {
                /* istanbul ignore next */
                return _this.getText(labelElement);
            }
            return _this.getNearestLabel(parent);
        };
        this.getElementPath = function (element) {
            var _a;
            if (!element) {
                return '';
            }
            var startTime = performance.now();
            var elementPath = (0, element_path_1.cssPath)(element);
            var endTime = performance.now();
            (_a = _this.diagnosticsClient) === null || _a === void 0 ? void 0 : _a.recordHistogram('autocapturePlugin.getElementPath', endTime - startTime);
            return elementPath;
        };
        // Returns the Amplitude event properties for the given element.
        this.getEventProperties = function (actionType, element, dataAttributePrefix) {
            var _a;
            var _b, _c, _d;
            /* istanbul ignore next */
            var tag = (_c = (_b = element === null || element === void 0 ? void 0 : element.tagName) === null || _b === void 0 ? void 0 : _b.toLowerCase) === null || _c === void 0 ? void 0 : _c.call(_b);
            /* istanbul ignore next */
            var rect = typeof element.getBoundingClientRect === 'function' ? element.getBoundingClientRect() : { left: null, top: null };
            var hierarchy = _this.getHierarchy(element);
            var currentElementAttributes = (_d = hierarchy[0]) === null || _d === void 0 ? void 0 : _d.attrs;
            var nearestLabel = _this.getNearestLabel(element);
            var attributes = (0, helpers_1.extractPrefixedAttributes)(currentElementAttributes !== null && currentElementAttributes !== void 0 ? currentElementAttributes : {}, dataAttributePrefix);
            /* istanbul ignore next */
            var properties = (_a = {},
                _a[constants.AMPLITUDE_EVENT_PROP_ELEMENT_HIERARCHY] = hierarchy,
                _a[constants.AMPLITUDE_EVENT_PROP_ELEMENT_TAG] = tag,
                _a[constants.AMPLITUDE_EVENT_PROP_ELEMENT_TEXT] = _this.getText(element),
                _a[constants.AMPLITUDE_EVENT_PROP_ELEMENT_POSITION_LEFT] = rect.left == null ? null : Math.round(rect.left),
                _a[constants.AMPLITUDE_EVENT_PROP_ELEMENT_POSITION_TOP] = rect.top == null ? null : Math.round(rect.top),
                _a[constants.AMPLITUDE_EVENT_PROP_ELEMENT_ATTRIBUTES] = attributes,
                _a[constants.AMPLITUDE_EVENT_PROP_ELEMENT_PATH] = _this.getElementPath(element),
                _a[constants.AMPLITUDE_EVENT_PROP_ELEMENT_PARENT_LABEL] = nearestLabel,
                _a[constants.AMPLITUDE_EVENT_PROP_PAGE_URL] = (0, analytics_core_1.getDecodeURI)(window.location.href.split('?')[0]),
                _a[constants.AMPLITUDE_EVENT_PROP_PAGE_TITLE] = analytics_core_1.getPageTitle(_this.replaceSensitiveString),
                _a[constants.AMPLITUDE_EVENT_PROP_VIEWPORT_HEIGHT] = window.innerHeight,
                _a[constants.AMPLITUDE_EVENT_PROP_VIEWPORT_WIDTH] = window.innerWidth,
                _a);
            var pageViewId = (0, helpers_1.getCurrentPageViewId)();
            /* istanbul ignore next */
            if (pageViewId) {
                /* istanbul ignore next */
                properties[constants.AMPLITUDE_EVENT_PROP_PAGE_VIEW_ID] = pageViewId;
            }
            // id is never masked, so always include it
            properties[constants.AMPLITUDE_EVENT_PROP_ELEMENT_ID] = element.getAttribute('id') || '';
            // class is never masked, so always include it
            properties[constants.AMPLITUDE_EVENT_PROP_ELEMENT_CLASS] = element.getAttribute('class');
            properties[constants.AMPLITUDE_EVENT_PROP_ELEMENT_ARIA_LABEL] = currentElementAttributes === null || currentElementAttributes === void 0 ? void 0 : currentElementAttributes['aria-label'];
            if (tag === 'a' && actionType === 'click' && element instanceof HTMLAnchorElement) {
                var href = element.href.substring(0, constants.MAX_ATTRIBUTE_LENGTH);
                properties[constants.AMPLITUDE_EVENT_PROP_ELEMENT_HREF] = _this.replaceSensitiveString(href); // we don't use hierarchy here because we don't want href value to be changed
            }
            return (0, helpers_1.removeEmptyProperties)(properties);
        };
        this.addTypeAndTimestamp = function (event, type) {
            return {
                event: event,
                timestamp: Date.now(),
                type: type,
            };
        };
        this.addAdditionalEventProperties = function (event, type, selectorAllowlist, dataAttributePrefix, 
        // capture the event if the cursor is a "pointer" when this element is clicked on
        // reason: a "pointer" cursor indicates that an element should be interactable
        //         regardless of the element's tag name
        isCapturingCursorPointer) {
            if (isCapturingCursorPointer === void 0) { isCapturingCursorPointer = false; }
            var baseEvent = _this.addTypeAndTimestamp(event, type);
            if ((0, helpers_1.isElementBasedEvent)(baseEvent) && baseEvent.event.target !== null) {
                if (isCapturingCursorPointer) {
                    var isCursorPointer = (0, helpers_1.isElementPointerCursor)(baseEvent.event.target, baseEvent.type);
                    if (isCursorPointer) {
                        baseEvent.closestTrackedAncestor = baseEvent.event.target;
                        baseEvent.targetElementProperties = _this.getEventProperties(baseEvent.type, baseEvent.closestTrackedAncestor, dataAttributePrefix);
                        return baseEvent;
                    }
                }
                // Retrieve additional event properties from the target element
                var closestTrackedAncestor = (0, helpers_1.getClosestElement)(baseEvent.event.target, selectorAllowlist);
                if (closestTrackedAncestor) {
                    baseEvent.closestTrackedAncestor = closestTrackedAncestor;
                    baseEvent.targetElementProperties = _this.getEventProperties(baseEvent.type, closestTrackedAncestor, dataAttributePrefix);
                }
                return baseEvent;
            }
            return baseEvent;
        };
        this.extractDataFromDataSource = function (dataSource, contextElement) {
            // Extract from DOM Element
            if (dataSource.sourceType === 'DOM_ELEMENT') {
                var sourceElement = (0, actions_1.getDataSource)(dataSource, contextElement);
                if (!sourceElement) {
                    return undefined;
                }
                if (dataSource.elementExtractType === 'TEXT') {
                    return _this.getText(sourceElement);
                }
                else if (dataSource.elementExtractType === 'ATTRIBUTE' && dataSource.attribute) {
                    return sourceElement.getAttribute(dataSource.attribute);
                }
                return undefined;
            }
            // TODO: Extract from other source types
            return undefined;
        };
        // Traverse text content without cloning DOM nodes, which avoids media/network side effects
        // from recreating nested elements such as <video>, <audio>, or <img>.
        this.getTextWithMaskedDescendants = function (element) {
            var e_3, _a;
            var maskedSelector = "[".concat(analytics_core_1.TEXT_MASK_ATTRIBUTE, "], [contenteditable]");
            // Fast path: if no masked descendants exist, rely on native text extraction.
            if (!element.querySelector(maskedSelector)) {
                return element.innerText;
            }
            var output = '';
            var childNodes = Array.from(element.childNodes);
            try {
                for (var childNodes_1 = tslib_1.__values(childNodes), childNodes_1_1 = childNodes_1.next(); !childNodes_1_1.done; childNodes_1_1 = childNodes_1.next()) {
                    var childNode = childNodes_1_1.value;
                    if (childNode.nodeType === Node.TEXT_NODE) {
                        output += childNode.textContent || '';
                        continue;
                    }
                    if (!(childNode instanceof Element)) {
                        continue;
                    }
                    // Replace entire masked/contenteditable subtrees with the mask token.
                    if (childNode.hasAttribute(analytics_core_1.TEXT_MASK_ATTRIBUTE) || childNode.hasAttribute('contenteditable')) {
                        output += analytics_core_1.MASKED_TEXT_VALUE;
                        continue;
                    }
                    output += _this.getTextWithMaskedDescendants(childNode);
                }
            }
            catch (e_3_1) { e_3 = { error: e_3_1 }; }
            finally {
                try {
                    if (childNodes_1_1 && !childNodes_1_1.done && (_a = childNodes_1.return)) _a.call(childNodes_1);
                }
                finally { if (e_3) throw e_3.error; }
            }
            return output;
        };
        this.getText = function (element) {
            // Check if element or any parent has data-amp-mask attribute
            var hasMaskAttribute = element.closest("[".concat(analytics_core_1.TEXT_MASK_ATTRIBUTE, "]")) !== null;
            if (hasMaskAttribute) {
                return analytics_core_1.MASKED_TEXT_VALUE;
            }
            var output = '';
            if (!element.querySelector("[".concat(analytics_core_1.TEXT_MASK_ATTRIBUTE, "], [contenteditable]"))) {
                output = element.innerText || '';
            }
            else {
                output = _this.getTextWithMaskedDescendants(element);
            }
            return _this.replaceSensitiveString(output.substring(0, 255)).replace(/\s+/g, ' ').trim();
        };
        // Returns the element properties for the given element in Visual Labeling.
        this.getEventTagProps = function (element) {
            var _a;
            var _b, _c;
            if (!element) {
                return {};
            }
            /* istanbul ignore next */
            var tag = (_c = (_b = element === null || element === void 0 ? void 0 : element.tagName) === null || _b === void 0 ? void 0 : _b.toLowerCase) === null || _c === void 0 ? void 0 : _c.call(_b);
            var properties = (_a = {},
                _a[constants.AMPLITUDE_EVENT_PROP_ELEMENT_TAG] = tag,
                _a[constants.AMPLITUDE_EVENT_PROP_ELEMENT_TEXT] = _this.getText(element),
                _a[constants.AMPLITUDE_EVENT_PROP_PAGE_URL] = window.location.href.split('?')[0],
                _a);
            return (0, helpers_1.removeEmptyProperties)(properties);
        };
        this.diagnosticsClient = context === null || context === void 0 ? void 0 : context.diagnosticsClient;
        var rawPatterns = (_b = options.maskTextRegex) !== null && _b !== void 0 ? _b : [];
        var compiled = [];
        try {
            for (var rawPatterns_1 = tslib_1.__values(rawPatterns), rawPatterns_1_1 = rawPatterns_1.next(); !rawPatterns_1_1.done; rawPatterns_1_1 = rawPatterns_1.next()) {
                var entry = rawPatterns_1_1.value;
                if (compiled.length >= constants.MAX_MASK_TEXT_PATTERNS) {
                    break;
                }
                if (entry instanceof RegExp) {
                    compiled.push(entry);
                }
                else if ('pattern' in entry && typeof entry.pattern === 'string') {
                    try {
                        compiled.push(new RegExp(entry.pattern, 'i'));
                    }
                    catch (_c) {
                        // ignore invalid pattern strings
                    }
                }
            }
        }
        catch (e_1_1) { e_1 = { error: e_1_1 }; }
        finally {
            try {
                if (rawPatterns_1_1 && !rawPatterns_1_1.done && (_a = rawPatterns_1.return)) _a.call(rawPatterns_1);
            }
            finally { if (e_1) throw e_1.error; }
        }
        this.additionalMaskTextPatterns = compiled;
    }
    return DataExtractor;
}());
exports.DataExtractor = DataExtractor;
//# sourceMappingURL=data-extractor.js.map