import { getGlobalScope } from '@amplitude/analytics-core';
import * as amplitudeGTM from './index';
import { createInstance } from './browser-client-factory';
import { runQueuedFunctions } from './utils/snippet-helper';
// https://developer.mozilla.org/en-US/docs/Glossary/IIFE
(function () {
    var GlobalScope = getGlobalScope();
    if (!GlobalScope) {
        console.error('[Amplitude] Error: GlobalScope is not defined');
        return;
    }
    var createNamedInstance = function (instanceName) {
        var instance = createInstance();
        var GlobalScope = getGlobalScope();
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
        runQueuedFunctions(amplitudeGTM, queue);
        var instanceNames = Object.keys(GlobalScope.amplitudeGTM._iq) || [];
        for (var i = 0; i < instanceNames.length; i++) {
            var instanceName = instanceNames[i];
            var instance = Object.assign(GlobalScope.amplitudeGTM._iq[instanceName], createNamedInstance(instanceName));
            var queue_1 = instance._q;
            instance._q = [];
            runQueuedFunctions(instance, queue_1);
        }
    }
})();
//# sourceMappingURL=gtm-snippet-index.js.map