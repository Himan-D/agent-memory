"use strict";
/* eslint-disable no-restricted-globals */
/* Only file allowed to access to globalThis, window, self */
Object.defineProperty(exports, "__esModule", { value: true });
exports.getGlobalScope = void 0;
var getGlobalScope = function () {
    // This should only be used for integrations with Amplitude that are not running in a browser environment
    //   We need to specify the name of the global variable as a string to prevent it from being minified
    var ampIntegrationContextName = 'ampIntegrationContext';
    if (typeof globalThis !== 'undefined' && typeof globalThis[ampIntegrationContextName] !== 'undefined') {
        return globalThis[ampIntegrationContextName];
    }
    if (typeof globalThis !== 'undefined') {
        return globalThis;
    }
    if (typeof window !== 'undefined') {
        return window;
    }
    if (typeof self !== 'undefined') {
        return self;
    }
    if (typeof global !== 'undefined') {
        return global;
    }
    return undefined;
};
exports.getGlobalScope = getGlobalScope;
//# sourceMappingURL=global-scope.js.map