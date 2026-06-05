import { __values } from "tslib";
import { isNonSensitiveElement } from './helpers';
import { DATA_AMP_MASK_ATTRIBUTES } from './constants';
import * as constants from './constants';
import { MASKED_TEXT_VALUE, TEXT_MASK_ATTRIBUTE } from '@amplitude/analytics-core';
var BLOCKED_ATTRIBUTES = new Set([
    // Already captured elsewhere in the hierarchy object
    'id',
    'class',
    // non-useful and potentially large attribute
    'style',
    // sensitive as prefilled form data may populate this attribute
    'value',
    // DOM events
    'onclick',
    'onchange',
    'oninput',
    'onblur',
    'onsubmit',
    'onfocus',
    'onkeydown',
    'onkeyup',
    'onkeypress',
    // React specific
    'data-reactid',
    'data-react-checksum',
    'data-reactroot',
    // Amplitude specific - used for redaction but should not be included in getElementProperties
    DATA_AMP_MASK_ATTRIBUTES,
    TEXT_MASK_ATTRIBUTE,
]);
var SENSITIVE_ELEMENT_ATTRIBUTE_ALLOWLIST = ['type'];
var SVG_TAGS = ['svg', 'path', 'g'];
var HIGHLY_SENSITIVE_INPUT_TYPES = ['password', 'hidden'];
export var MAX_HIERARCHY_LENGTH = 1024;
export function getElementProperties(element, userMaskedAttributeNames) {
    var e_1, _a;
    var _b, _c, _d, _e;
    if (element === null) {
        return null;
    }
    var tagName = String(element.tagName).toLowerCase();
    var properties = {
        tag: tagName,
    };
    var siblings = Array.from((_c = (_b = element.parentElement) === null || _b === void 0 ? void 0 : _b.children) !== null && _c !== void 0 ? _c : []);
    if (siblings.length) {
        properties.index = siblings.indexOf(element);
        properties.indexOfType = siblings.filter(function (el) { return el.tagName === element.tagName; }).indexOf(element);
    }
    var prevSiblingTag = (_e = (_d = element.previousElementSibling) === null || _d === void 0 ? void 0 : _d.tagName) === null || _e === void 0 ? void 0 : _e.toLowerCase();
    if (prevSiblingTag) {
        properties.prevSib = String(prevSiblingTag);
    }
    var id = element.getAttribute('id');
    if (id) {
        properties.id = String(id);
    }
    var classes = Array.from(element.classList);
    if (classes.length) {
        properties.classes = classes;
    }
    var attributes = {};
    var attributesArray = Array.from(element.attributes);
    var filteredAttributes = attributesArray.filter(function (attr) { return !BLOCKED_ATTRIBUTES.has(attr.name); });
    var isSensitiveElement = !isNonSensitiveElement(element);
    // if input is hidden or password or for SVGs, skip attribute collection entirely
    if (!HIGHLY_SENSITIVE_INPUT_TYPES.includes(String(element.getAttribute('type'))) && !SVG_TAGS.includes(tagName)) {
        try {
            for (var filteredAttributes_1 = __values(filteredAttributes), filteredAttributes_1_1 = filteredAttributes_1.next(); !filteredAttributes_1_1.done; filteredAttributes_1_1 = filteredAttributes_1.next()) {
                var attr = filteredAttributes_1_1.value;
                // If sensitive element, only allow certain attributes
                if (isSensitiveElement && !SENSITIVE_ELEMENT_ATTRIBUTE_ALLOWLIST.includes(attr.name)) {
                    continue;
                }
                if (userMaskedAttributeNames.has(attr.name)) {
                    attributes[attr.name] = MASKED_TEXT_VALUE;
                    continue;
                }
                // Finally cast attribute value to string and limit attribute value length
                attributes[attr.name] = String(attr.value).substring(0, constants.MAX_ATTRIBUTE_LENGTH);
            }
        }
        catch (e_1_1) { e_1 = { error: e_1_1 }; }
        finally {
            try {
                if (filteredAttributes_1_1 && !filteredAttributes_1_1.done && (_a = filteredAttributes_1.return)) _a.call(filteredAttributes_1);
            }
            finally { if (e_1) throw e_1.error; }
        }
    }
    if (Object.keys(attributes).length) {
        properties.attrs = attributes;
    }
    return properties;
}
export function getAncestors(targetEl) {
    var ancestors = [];
    if (!targetEl) {
        return ancestors;
    }
    // Add self to the list of ancestors
    ancestors.push(targetEl);
    var current = targetEl.parentElement;
    while (current && current.tagName !== 'HTML') {
        ancestors.push(current);
        current = current.parentElement;
    }
    return ancestors;
}
//# sourceMappingURL=hierarchy.js.map