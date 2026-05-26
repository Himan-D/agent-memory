import { __values } from "tslib";
export var TEXT_MASK_ATTRIBUTE = 'data-amp-mask';
export var MASKED_TEXT_VALUE = '*****';
// Regex patterns for sensitive data
export var CC_REGEX = /\b(?:\d[ -]*?){13,16}\b/;
export var SSN_REGEX = /(\d{3}-?\d{2}-?\d{4})/g;
export var EMAIL_REGEX = /[^\s@]+@[^\s@.]+\.[^\s@]+/g;
/**
 * Replaces sensitive strings (credit cards, SSNs, emails) and custom patterns with masked text
 * @param text - The text to search for sensitive data
 * @param additionalMaskTextPatterns - Optional array of additional regex patterns to mask
 * @returns The text with sensitive data replaced by masked text
 */
export var replaceSensitiveString = function (text, additionalMaskTextPatterns) {
    var e_1, _a;
    if (additionalMaskTextPatterns === void 0) { additionalMaskTextPatterns = []; }
    if (typeof text !== 'string') {
        return '';
    }
    var result = text;
    // Check for credit card number (with or without spaces/dashes)
    result = result.replace(CC_REGEX, MASKED_TEXT_VALUE);
    // Check for social security number
    result = result.replace(SSN_REGEX, MASKED_TEXT_VALUE);
    // Check for email
    result = result.replace(EMAIL_REGEX, MASKED_TEXT_VALUE);
    try {
        // Check for additional mask text patterns
        for (var additionalMaskTextPatterns_1 = __values(additionalMaskTextPatterns), additionalMaskTextPatterns_1_1 = additionalMaskTextPatterns_1.next(); !additionalMaskTextPatterns_1_1.done; additionalMaskTextPatterns_1_1 = additionalMaskTextPatterns_1.next()) {
            var pattern = additionalMaskTextPatterns_1_1.value;
            try {
                result = result.replace(pattern, MASKED_TEXT_VALUE);
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
/**
 * Gets the page title, checking if the title element has data-amp-mask attribute
 * @returns The page title, masked if the title element has data-amp-mask attribute
 */
export var getPageTitle = function (parseTitleFunction) {
    if (typeof document === 'undefined' || !document.title) {
        return '';
    }
    var titleElement = document.querySelector('title');
    if (titleElement && titleElement.hasAttribute(TEXT_MASK_ATTRIBUTE)) {
        return MASKED_TEXT_VALUE;
    }
    return parseTitleFunction ? parseTitleFunction(document.title) : document.title; // document.title is always synced to the first title element
};
//# sourceMappingURL=helpers.js.map