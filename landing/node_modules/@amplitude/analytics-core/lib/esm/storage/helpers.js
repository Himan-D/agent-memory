import { AMPLITUDE_PREFIX } from '../types/constants';
export var getStorageKey = function (apiKey, postKey, limit) {
    if (postKey === void 0) { postKey = ''; }
    if (limit === void 0) { limit = 10; }
    return [AMPLITUDE_PREFIX, postKey, apiKey.substring(0, limit)].filter(Boolean).join('_');
};
//# sourceMappingURL=helpers.js.map