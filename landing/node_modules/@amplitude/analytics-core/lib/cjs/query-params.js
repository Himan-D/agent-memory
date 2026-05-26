"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.tryDecodeURIComponent = exports.getQueryParams = void 0;
var global_scope_1 = require("./global-scope");
var getQueryParams = function () {
    var _a;
    var globalScope = (0, global_scope_1.getGlobalScope)();
    /* istanbul ignore if */
    if (!((_a = globalScope === null || globalScope === void 0 ? void 0 : globalScope.location) === null || _a === void 0 ? void 0 : _a.search)) {
        return {};
    }
    var pairs = globalScope.location.search.substring(1).split('&').filter(Boolean);
    var params = pairs.reduce(function (acc, curr) {
        var query = curr.split('=', 2);
        var key = (0, exports.tryDecodeURIComponent)(query[0]);
        var value = (0, exports.tryDecodeURIComponent)(query[1]);
        if (!value) {
            return acc;
        }
        acc[key] = value;
        return acc;
    }, {});
    return params;
};
exports.getQueryParams = getQueryParams;
var tryDecodeURIComponent = function (value) {
    if (value === void 0) { value = ''; }
    try {
        return decodeURIComponent(value);
    }
    catch (_a) {
        return '';
    }
};
exports.tryDecodeURIComponent = tryDecodeURIComponent;
//# sourceMappingURL=query-params.js.map