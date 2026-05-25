/**
 * Source: [jed's gist's comment]{@link https://gist.github.com/jed/982883?permalink_comment_id=3223002#gistcomment-3223002}.
 * Returns a random v4 UUID of the form xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx,
 * where each x is replaced with a random hexadecimal digit from 0 to f, and
 * y is replaced with a random hexadecimal digit from 8 to b.
 * Used to generate UUIDs for deviceIds.
 * @private
 */
import { __read, __spreadArray } from "tslib";
import { getGlobalScope } from '../global-scope';
var legacyUUID = function (a) {
    return a // if the placeholder was passed, return
        ? // a random number from 0 to 15
            (a ^ // unless b is 8,
                ((Math.random() * // in which case
                    16) >> // a random number from
                    (a / 4))) // 8 to 11
                .toString(16) // in hexadecimal
        : // or otherwise a concatenated string:
            (String(1e7) + // 10000000 +
                String(-1e3) + // -1000 +
                String(-4e3) + // -4000 +
                String(-8e3) + // -80000000 +
                String(-1e11)) // -100000000000,
                .replace(
            // replacing
            /[018]/g, // zeroes, ones, and eights with
            UUID);
};
var hex = __spreadArray([], __read(Array(256).keys()), false).map(function (index) { return index.toString(16).padStart(2, '0'); });
export var UUID = function (a) {
    var _a;
    var globalScope = getGlobalScope();
    /* istanbul ignore next */
    if (!((_a = globalScope === null || globalScope === void 0 ? void 0 : globalScope.crypto) === null || _a === void 0 ? void 0 : _a.getRandomValues)) {
        // Fallback to legacy UUID generation if crypto is not available
        return legacyUUID(a);
    }
    var r = globalScope.crypto.getRandomValues(new Uint8Array(16));
    r[6] = (r[6] & 0x0f) | 0x40;
    r[8] = (r[8] & 0x3f) | 0x80;
    return __spreadArray([], __read(r.entries()), false).map(function (_a) {
        var _b = __read(_a, 2), index = _b[0], int = _b[1];
        return ([4, 6, 8, 10].includes(index) ? "-".concat(hex[int]) : hex[int]);
    }).join('');
};
//# sourceMappingURL=uuid.js.map