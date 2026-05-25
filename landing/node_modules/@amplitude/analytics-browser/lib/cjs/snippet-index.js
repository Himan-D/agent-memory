"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var tslib_1 = require("tslib");
var analytics_core_1 = require("@amplitude/analytics-core");
var amplitude = tslib_1.__importStar(require("./index"));
var browser_client_factory_1 = require("./browser-client-factory");
var snippet_helper_1 = require("./utils/snippet-helper");
(0, analytics_core_1.registerSdkLoaderMetadata)({
    scriptUrl: resolveCurrentScriptUrl(),
});
function resolveCurrentScriptUrl() {
    if (typeof document === 'undefined') {
        return undefined;
    }
    var currentScript = document.currentScript;
    if (currentScript === null || currentScript === void 0 ? void 0 : currentScript.src) {
        return currentScript.src;
    }
    return undefined;
}
// https://developer.mozilla.org/en-US/docs/Glossary/IIFE
(function () {
    var GlobalScope = (0, analytics_core_1.getGlobalScope)();
    if (!GlobalScope) {
        console.error('[Amplitude] Error: GlobalScope is not defined');
        return;
    }
    var createNamedInstance = function (instanceName) {
        var instance = (0, browser_client_factory_1.createInstance)();
        var GlobalScope = (0, analytics_core_1.getGlobalScope)();
        if (GlobalScope && GlobalScope.amplitude && GlobalScope.amplitude._iq && instanceName) {
            // eslint-disable-next-line @typescript-eslint/ban-ts-comment
            // @ts-ignore
            GlobalScope.amplitude._iq[instanceName] = instance;
        }
        return instance;
    };
    GlobalScope.amplitude = Object.assign(GlobalScope.amplitude || {}, amplitude, {
        createInstance: createNamedInstance,
    });
    if (GlobalScope.amplitude.invoked) {
        var queue = GlobalScope.amplitude._q;
        GlobalScope.amplitude._q = [];
        (0, snippet_helper_1.runQueuedFunctions)(amplitude, queue);
        var instanceNames = Object.keys(GlobalScope.amplitude._iq) || [];
        for (var i = 0; i < instanceNames.length; i++) {
            var instanceName = instanceNames[i];
            var instance = Object.assign(GlobalScope.amplitude._iq[instanceName], createNamedInstance(instanceName));
            var queue_1 = instance._q;
            instance._q = [];
            (0, snippet_helper_1.runQueuedFunctions)(instance, queue_1);
        }
    }
})();
//# sourceMappingURL=snippet-index.js.map