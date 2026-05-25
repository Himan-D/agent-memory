/* eslint-disable no-restricted-globals */
/**
 * Dynamically loads an external script by appending a <script> tag to the document head.
 * Deduplicates by checking if a script with the same src already exists.
 */
export var asyncLoadScript = function (url) {
    // Dedup: if a script with this src already exists, resolve immediately
    var existing = document.querySelector("script[src=\"".concat(CSS.escape(url), "\"]"));
    if (existing) {
        return Promise.resolve({ status: true });
    }
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
/**
 * Generates a simple unique ID for message request/response correlation.
 */
export function generateUniqueId() {
    return "".concat(Date.now(), "-").concat(Math.random().toString(36).substr(2, 9));
}
//# sourceMappingURL=utils.js.map