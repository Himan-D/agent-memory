"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getOldCookieName = exports.getCookieName = void 0;
var constants_1 = require("./types/constants");
var getCookieName = function (apiKey, postKey, limit) {
    if (postKey === void 0) { postKey = ''; }
    if (limit === void 0) { limit = 10; }
    return [constants_1.AMPLITUDE_PREFIX, postKey, apiKey.substring(0, limit)].filter(Boolean).join('_');
};
exports.getCookieName = getCookieName;
var getOldCookieName = function (apiKey) {
    return "".concat(constants_1.AMPLITUDE_PREFIX.toLowerCase(), "_").concat(apiKey.substring(0, 6));
};
exports.getOldCookieName = getOldCookieName;
//# sourceMappingURL=cookie-name.js.map