"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.isInstanceProxy = exports.convertProxyObjectToRealObject = exports.runQueuedFunctions = void 0;
/**
 * Applies the proxied functions on the proxied amplitude snippet to an instance of the real object.
 * @ignore
 */
var runQueuedFunctions = function (instance, queue) {
    (0, exports.convertProxyObjectToRealObject)(instance, queue);
};
exports.runQueuedFunctions = runQueuedFunctions;
/**
 * Applies the proxied functions on the proxied object to an instance of the real object.
 * Used to convert proxied Identify and Revenue objects.
 */
var convertProxyObjectToRealObject = function (instance, queue) {
    for (var i = 0; i < queue.length; i++) {
        var _a = queue[i], name_1 = _a.name, args = _a.args, resolve = _a.resolve;
        var fn = instance && instance[name_1];
        if (typeof fn === 'function') {
            var result = fn.apply(instance, args);
            if (typeof resolve === 'function') {
                resolve(result === null || result === void 0 ? void 0 : result.promise);
            }
        }
    }
    return instance;
};
exports.convertProxyObjectToRealObject = convertProxyObjectToRealObject;
/**
 * Check if the param is snippet proxy
 */
var isInstanceProxy = function (instance) {
    var instanceProxy = instance;
    return instanceProxy && instanceProxy._q !== undefined;
};
exports.isInstanceProxy = isInstanceProxy;
//# sourceMappingURL=snippet-helper.js.map