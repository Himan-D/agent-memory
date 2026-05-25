"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getStorageKey = void 0;
var constants_1 = require("../types/constants");
var getStorageKey = function (apiKey, postKey, limit) {
    if (postKey === void 0) { postKey = ''; }
    if (limit === void 0) { limit = 10; }
    return [constants_1.AMPLITUDE_PREFIX, postKey, apiKey.substring(0, limit)].filter(Boolean).join('_');
};
exports.getStorageKey = getStorageKey;
//# sourceMappingURL=helpers.js.map