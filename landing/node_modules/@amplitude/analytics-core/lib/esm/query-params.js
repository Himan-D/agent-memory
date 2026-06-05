import { getGlobalScope } from './global-scope';
export var getQueryParams = function () {
    var _a;
    var globalScope = getGlobalScope();
    /* istanbul ignore if */
    if (!((_a = globalScope === null || globalScope === void 0 ? void 0 : globalScope.location) === null || _a === void 0 ? void 0 : _a.search)) {
        return {};
    }
    var pairs = globalScope.location.search.substring(1).split('&').filter(Boolean);
    var params = pairs.reduce(function (acc, curr) {
        var query = curr.split('=', 2);
        var key = tryDecodeURIComponent(query[0]);
        var value = tryDecodeURIComponent(query[1]);
        if (!value) {
            return acc;
        }
        acc[key] = value;
        return acc;
    }, {});
    return params;
};
export var tryDecodeURIComponent = function (value) {
    if (value === void 0) { value = ''; }
    try {
        return decodeURIComponent(value);
    }
    catch (_a) {
        return '';
    }
};
//# sourceMappingURL=query-params.js.map