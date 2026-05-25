"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.consoleObserver = void 0;
var tslib_1 = require("tslib");
var global_scope_1 = require("../global-scope");
var globalScope = (0, global_scope_1.getGlobalScope)();
/* istanbul ignore next */
var originalConsole = globalScope === null || globalScope === void 0 ? void 0 : globalScope.console;
var handlers = {};
// keeps reference to original console methods
var originalFn = {};
var inConsoleOverride = false;
function overrideConsole(logLevel) {
    /* istanbul ignore if */
    if (!originalConsole) {
        return false;
    }
    // should not override if original console property is not a function
    /* eslint-disable-next-line @typescript-eslint/no-unsafe-member-access */
    if (typeof originalConsole[logLevel] !== 'function') {
        return false;
    }
    // if console is already overridden, return true
    if (originalFn[logLevel]) {
        return true;
    }
    // override console method
    var handler = function () {
        var args = [];
        for (var _i = 0; _i < arguments.length; _i++) {
            args[_i] = arguments[_i];
        }
        try {
            if (handlers[logLevel] && !inConsoleOverride) {
                // add a re-entrancy guard to prevent infinite recursion
                inConsoleOverride = true;
                var callbacks = handlers[logLevel];
                if (callbacks) {
                    callbacks.forEach(function (callback) {
                        try {
                            callback(logLevel, args);
                        }
                        catch (_a) {
                            // do nothing
                        }
                    });
                }
            }
        }
        catch (_a) {
            // do nothing
        }
        inConsoleOverride = false;
        return originalFn[logLevel].apply(originalConsole, args);
    };
    /* eslint-disable-next-line @typescript-eslint/no-unsafe-member-access */
    originalFn[logLevel] = originalConsole[logLevel];
    /* eslint-disable-next-line @typescript-eslint/no-unsafe-member-access */
    originalConsole[logLevel] = handler;
    return true;
}
/**
 * Observe a console log method (log, warn, error, etc.)
 * @param level - The console log level to observe
 * @param callback - The callback function to call when the console log level is observed
 */
function addListener(level, callback) {
    var res = overrideConsole(level);
    /* istanbul ignore if */
    if (!res) {
        return new Error('Console override failed');
    }
    if (handlers[level]) {
        // using ! is safe because we know the key exists based on condition above
        handlers[level].push(callback);
    }
    else {
        handlers[level] = [callback];
    }
}
/**
 * Disconnect a callback function from a console log method
 * @param callback - The callback function to disconnect
 */
function removeListener(callback) {
    var e_1, _a;
    try {
        for (var _b = tslib_1.__values(Object.values(handlers)), _c = _b.next(); !_c.done; _c = _b.next()) {
            var callbacks = _c.value;
            // iterate backwards to avoid index shifting
            for (var i = callbacks.length - 1; i >= 0; i--) {
                if (callbacks[i] === callback) {
                    callbacks.splice(i, 1);
                    break;
                }
            }
        }
    }
    catch (e_1_1) { e_1 = { error: e_1_1 }; }
    finally {
        try {
            if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
        }
        finally { if (e_1) throw e_1.error; }
    }
}
// this should only be used for testing
// restoring console can break console overrides
function _restoreConsole() {
    var e_2, _a;
    try {
        for (var _b = tslib_1.__values(Object.entries(originalFn)), _c = _b.next(); !_c.done; _c = _b.next()) {
            var _d = tslib_1.__read(_c.value, 2), key = _d[0], originalHandler = _d[1];
            if (originalHandler) {
                /* eslint-disable-next-line @typescript-eslint/no-unsafe-member-access */
                originalConsole[key] = originalHandler;
            }
        }
    }
    catch (e_2_1) { e_2 = { error: e_2_1 }; }
    finally {
        try {
            if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
        }
        finally { if (e_2) throw e_2.error; }
    }
    originalFn = {};
    handlers = {};
}
var consoleObserver = {
    addListener: addListener,
    removeListener: removeListener,
    _restoreConsole: _restoreConsole,
};
exports.consoleObserver = consoleObserver;
//# sourceMappingURL=console.js.map