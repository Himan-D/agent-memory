"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.isSuccessStatusCode = void 0;
/**
 * Checks if an HTTP status code indicates success (2xx range)
 * @param code - The HTTP status code to check
 * @returns true if the status code is in the 2xx range, false otherwise
 */
function isSuccessStatusCode(code) {
    return code >= 200 && code < 300;
}
exports.isSuccessStatusCode = isSuccessStatusCode;
//# sourceMappingURL=status-code.js.map