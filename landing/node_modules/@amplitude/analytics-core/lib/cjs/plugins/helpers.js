"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getPageTitle = exports.replaceSensitiveString = exports.EMAIL_REGEX = exports.SSN_REGEX = exports.CC_REGEX = exports.MASKED_TEXT_VALUE = exports.TEXT_MASK_ATTRIBUTE = void 0;
var tslib_1 = require("tslib");
exports.TEXT_MASK_ATTRIBUTE = 'data-amp-mask';
exports.MASKED_TEXT_VALUE = '*****';
// Regex patterns for sensitive data
exports.CC_REGEX = /\b(?:\d[ -]*?){13,16}\b/;
exports.SSN_REGEX = /(\d{3}-?\d{2}-?\d{4})/g;
exports.EMAIL_REGEX = /[^\s@]+@[^\s@.]+\.[^\s@]+/g;
/**
 * Replaces sensitive strings (credit cards, SSNs, emails) and custom patterns with masked text
 * @param text - The text to search for sensitive data
 * @param additionalMaskTextPatterns - Optional array of additional regex patterns to mask
 * @returns The text with sensitive data replaced by masked text
 */
var replaceSensitiveString = function (text, additionalMaskTextPatterns) {
    var e_1, _a;
    if (additionalMaskTextPatterns === void 0) { additionalMaskTextPatterns = []; }
    if (typeof text !== 'string') {
        return '';
    }
    var result = text;
    // Check for credit card number (with or without spaces/dashes)
    result = result.replace(exports.CC_REGEX, exports.MASKED_TEXT_VALUE);
    // Check for social security number
    result = result.replace(exports.SSN_REGEX, exports.MASKED_TEXT_VALUE);
    // Check for email
    result = result.replace(exports.EMAIL_REGEX, exports.MASKED_TEXT_VALUE);
    try {
        // Check for additional mask text patterns
        for (var additionalMaskTextPatterns_1 = tslib_1.__values(additionalMaskTextPatterns), additionalMaskTextPatterns_1_1 = additionalMaskTextPatterns_1.next(); !additionalMaskTextPatterns_1_1.done; additionalMaskTextPatterns_1_1 = additionalMaskTextPatterns_1.next()) {
            var pattern = additionalMaskTextPatterns_1_1.value;
            try {
                result = result.replace(pattern, exports.MASKED_TEXT_VALUE);
            }
            catch (_b) {
                // ignore invalid pattern
            }
        }
    }
    catch (e_1_1) { e_1 = { error: e_1_1 }; }
    finally {
        try {
            if (additionalMaskTextPatterns_1_1 && !additionalMaskTextPatterns_1_1.done && (_a = additionalMaskTextPatterns_1.return)) _a.call(additionalMaskTextPatterns_1);
        }
        finally { if (e_1) throw e_1.error; }
    }
    return result;
};
exports.replaceSensitiveString = replaceSensitiveString;
/**
 * Gets the page title, checking if the title element has data-amp-mask attribute
 * @returns The page title, masked if the title element has data-amp-mask attribute
 */
var getPageTitle = function (parseTitleFunction) {
    if (typeof document === 'undefined' || !document.title) {
        return '';
    }
    var titleElement = document.querySelector('title');
    if (titleElement && titleElement.hasAttribute(exports.TEXT_MASK_ATTRIBUTE)) {
        return exports.MASKED_TEXT_VALUE;
    }
    return parseTitleFunction ? parseTitleFunction(document.title) : document.title; // document.title is always synced to the first title element
};
exports.getPageTitle = getPageTitle;
//# sourceMappingURL=helpers.js.map