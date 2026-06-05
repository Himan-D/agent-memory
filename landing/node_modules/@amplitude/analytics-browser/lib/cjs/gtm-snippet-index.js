"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var tslib_1 = require("tslib");
var analytics_core_1 = require("@amplitude/analytics-core");
var amplitudeGTM = tslib_1.__importStar(require("./index"));
var browser_client_factory_1 = require("./browser-client-factory");
var snippet_helper_1 = require("./utils/snippet-helper");
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
        if (GlobalScope && GlobalScope.amplitudeGTM && GlobalScope.amplitudeGTM._iq && instanceName) {
            // eslint-disable-next-line @typescript-eslint/ban-ts-comment
            // @ts-ignore
            GlobalScope.amplitudeGTM._iq[instanceName] = instance;
        }
        return instance;
    };
    GlobalScope.amplitudeGTM = Object.assign(GlobalScope.amplitudeGTM || {}, amplitudeGTM, {
        createInstance: createNamedInstance,
    });
    if (GlobalScope.amplitudeGTM.invoked) {
        var queue = GlobalScope.amplitudeGTM._q;
        GlobalScope.amplitudeGTM._q = [];
        (0, snippet_helper_1.runQueuedFunctions)(amplitudeGTM, queue);
        var instanceNames = Object.keys(GlobalScope.amplitudeGTM._iq) || [];
        for (var i = 0; i < instanceNames.length; i++) {
            var instanceName = instanceNames[i];
            var instance = Object.assign(GlobalScope.amplitudeGTM._iq[instanceName], createNamedInstance(instanceName));
            var queue_1 = instance._q;
            instance._q = [];
            (0, snippet_helper_1.runQueuedFunctions)(instance, queue_1);
        }
    }
})();
//# sourceMappingURL=gtm-snippet-index.js.map