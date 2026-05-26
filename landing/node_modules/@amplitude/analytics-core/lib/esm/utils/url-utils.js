/**
 * Checks if a given URL matches any pattern in an allowlist of URLs or regex patterns.
 * @param url - The URL to check
 * @param allowlist - Array of allowed URLs (strings) or regex patterns
 * @returns true if the URL matches any pattern in the allowlist, false otherwise
 */
export var isUrlMatchAllowlist = function (url, allowlist) {
    if (!allowlist || !allowlist.length) {
        return true;
    }
    return allowlist.some(function (allowedUrl) {
        if (typeof allowedUrl === 'string') {
            return url === allowedUrl;
        }
        return url.match(allowedUrl);
    });
};
export var getDecodeURI = function (locationStr, loggerProvider) {
    var decodedLocationStr = locationStr;
    try {
        decodedLocationStr = decodeURI(locationStr);
    }
    catch (e) {
        /* istanbul ignore next */
        loggerProvider === null || loggerProvider === void 0 ? void 0 : loggerProvider.error('Malformed URI sequence: ', e);
    }
    return decodedLocationStr;
};
//# sourceMappingURL=url-utils.js.map