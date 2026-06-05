import { __assign, __values } from "tslib";
import { getGlobalScope } from '../global-scope';
export var GLOBAL_KEY = '__AMPLITUDE_SCRIPT_URL__';
export var EVENT_NAME_ERROR_UNCAUGHT = 'sdk.error.uncaught';
var getNormalizedScriptUrls = function () {
    var scope = getGlobalScope();
    /* istanbul ignore next */
    if (!scope) {
        return [];
    }
    var value = scope[GLOBAL_KEY];
    if (Array.isArray(value)) {
        return value;
    }
    /* istanbul ignore next - legacy single URL stored as string */
    if (typeof value === 'string') {
        return [value];
    }
    return [];
};
var addNormalizedScriptUrl = function (url) {
    var scope = getGlobalScope();
    /* istanbul ignore next */
    if (!scope) {
        return;
    }
    var urls = getNormalizedScriptUrls();
    if (!urls.includes(url)) {
        urls.push(url);
        scope[GLOBAL_KEY] = urls;
    }
};
export var registerSdkLoaderMetadata = function (metadata) {
    if (metadata.scriptUrl) {
        var normalized = normalizeUrl(metadata.scriptUrl);
        if (normalized) {
            addNormalizedScriptUrl(normalized);
        }
    }
};
export var enableSdkErrorListeners = function (client) {
    var scope = getGlobalScope();
    if (!scope || typeof scope.addEventListener !== 'function') {
        return;
    }
    var handleError = function (event) {
        var error = event.error instanceof Error ? event.error : undefined;
        var stack = error === null || error === void 0 ? void 0 : error.stack;
        var match = detectSdkOrigin({ filename: event.filename, stack: stack });
        if (!match) {
            return;
        }
        capture({
            type: 'error',
            message: event.message,
            stack: stack,
            filename: event.filename,
            errorName: error === null || error === void 0 ? void 0 : error.name,
            metadata: {
                colno: event.colno,
                lineno: event.lineno,
                isTrusted: event.isTrusted,
                matchReason: match,
            },
        });
    };
    var handleRejection = function (event) {
        var _a;
        var error = event.reason instanceof Error ? event.reason : undefined;
        var stack = error === null || error === void 0 ? void 0 : error.stack;
        var filename = extractFilenameFromStack(stack);
        var match = detectSdkOrigin({ filename: filename, stack: stack });
        if (!match) {
            return;
        }
        /* istanbul ignore next */
        capture({
            type: 'unhandledrejection',
            message: (_a = error === null || error === void 0 ? void 0 : error.message) !== null && _a !== void 0 ? _a : stringifyReason(event.reason),
            stack: stack,
            filename: filename,
            errorName: error === null || error === void 0 ? void 0 : error.name,
            metadata: {
                isTrusted: event.isTrusted,
                matchReason: match,
            },
        });
    };
    var capture = function (context) {
        client.recordEvent(EVENT_NAME_ERROR_UNCAUGHT, __assign({ type: context.type, message: context.message, filename: context.filename, error_name: context.errorName, stack: context.stack }, context.metadata));
    };
    scope.addEventListener('error', handleError, true);
    scope.addEventListener('unhandledrejection', handleRejection, true);
};
var detectSdkOrigin = function (payload) {
    var e_1, _a;
    var normalizedScriptUrls = getNormalizedScriptUrls();
    if (normalizedScriptUrls.length === 0) {
        return undefined;
    }
    try {
        for (var normalizedScriptUrls_1 = __values(normalizedScriptUrls), normalizedScriptUrls_1_1 = normalizedScriptUrls_1.next(); !normalizedScriptUrls_1_1.done; normalizedScriptUrls_1_1 = normalizedScriptUrls_1.next()) {
            var normalizedScriptUrl = normalizedScriptUrls_1_1.value;
            if (payload.filename && payload.filename.includes(normalizedScriptUrl)) {
                return 'filename';
            }
            if (payload.stack && payload.stack.includes(normalizedScriptUrl)) {
                return 'stack';
            }
        }
    }
    catch (e_1_1) { e_1 = { error: e_1_1 }; }
    finally {
        try {
            if (normalizedScriptUrls_1_1 && !normalizedScriptUrls_1_1.done && (_a = normalizedScriptUrls_1.return)) _a.call(normalizedScriptUrls_1);
        }
        finally { if (e_1) throw e_1.error; }
    }
    return undefined;
};
var normalizeUrl = function (value) {
    var _a, _b;
    try {
        /* istanbul ignore next */
        var url = new URL(value, (_b = (_a = getGlobalScope()) === null || _a === void 0 ? void 0 : _a.location) === null || _b === void 0 ? void 0 : _b.origin);
        return url.origin + url.pathname;
    }
    catch (_c) {
        return undefined;
    }
};
var extractFilenameFromStack = function (stack) {
    if (!stack) {
        return undefined;
    }
    var match = stack.match(/(https?:\/\/\S+?)(?=[)\s]|$)/);
    /* istanbul ignore next */
    return match ? match[1] : undefined;
};
/* istanbul ignore next */
var stringifyReason = function (reason) {
    if (typeof reason === 'string') {
        return reason;
    }
    try {
        return JSON.stringify(reason);
    }
    catch (_a) {
        return '[object Object]';
    }
};
//# sourceMappingURL=uncaught-sdk-errors.js.map