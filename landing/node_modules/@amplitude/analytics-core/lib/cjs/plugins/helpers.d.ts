export declare const TEXT_MASK_ATTRIBUTE = "data-amp-mask";
export declare const MASKED_TEXT_VALUE = "*****";
export declare const CC_REGEX: RegExp;
export declare const SSN_REGEX: RegExp;
export declare const EMAIL_REGEX: RegExp;
/**
 * Replaces sensitive strings (credit cards, SSNs, emails) and custom patterns with masked text
 * @param text - The text to search for sensitive data
 * @param additionalMaskTextPatterns - Optional array of additional regex patterns to mask
 * @returns The text with sensitive data replaced by masked text
 */
export declare const replaceSensitiveString: (text: string | null, additionalMaskTextPatterns?: RegExp[]) => string;
/**
 * Gets the page title, checking if the title element has data-amp-mask attribute
 * @returns The page title, masked if the title element has data-amp-mask attribute
 */
export declare const getPageTitle: (parseTitleFunction?: ((title: string) => string) | undefined) => string;
//# sourceMappingURL=helpers.d.ts.map