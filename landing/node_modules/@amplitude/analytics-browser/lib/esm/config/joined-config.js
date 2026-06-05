import { __assign, __read, __spreadArray, __values } from "tslib";
import { SAFE_HEADERS, } from '@amplitude/analytics-core';
/**
 * Performs a deep transformation of a remote config object so that
 * it matches the expected schema of the local config.
 *
 * Specifically, it normalizes nested `enabled` flags into concise union types.
 *
 * ### Transformation Rules:
 * - If an object has `enabled: true`, it is replaced by the same object without the `enabled` field.
 * - If it has only `enabled: true`, it is replaced with `true`.
 * - If it has `enabled: false`, it is replaced with `false` regardless of other fields.
 *
 * ### Examples:
 * Input:  { prop: { enabled: true, hello: 'world' }}
 * Output: { prop: { hello: 'world' } }
 *
 * Input:  { prop: { enabled: true }}
 * Output: { prop: true }
 *
 * Input:  { prop: { enabled: false, hello: 'world' }}
 * Output: { prop: false }
 *
 * Input:  { prop: { hello: 'world' }}
 * Output: { prop: { hello: 'world' } } // No change
 *
 * @param config Remote config object to be transformed
 * @returns Transformed config object compatible with local schema
 */
export function translateRemoteConfigToLocal(config) {
    var e_1, _a, e_2, _b, e_3, _c;
    var _d, _e, _f, _g, _h, _j;
    // Disabling type checking rules because remote config comes from a remote source
    // and this function needs to handle any unexpected values
    /* eslint-disable @typescript-eslint/no-unsafe-member-access,
       @typescript-eslint/no-unsafe-assignment,
       @typescript-eslint/no-unsafe-argument
   */
    if (typeof config !== 'object' || config === null) {
        return;
    }
    // translations are not applied on array properties
    if (Array.isArray(config)) {
        return;
    }
    var propertyNames = Object.keys(config);
    try {
        for (var propertyNames_1 = __values(propertyNames), propertyNames_1_1 = propertyNames_1.next(); !propertyNames_1_1.done; propertyNames_1_1 = propertyNames_1.next()) {
            var propertyName = propertyNames_1_1.value;
            try {
                var value = config[propertyName];
                // transform objects with { enabled } property to boolean | object
                if (typeof (value === null || value === void 0 ? void 0 : value.enabled) === 'boolean') {
                    if (value.enabled) {
                        // if enabled is true, set the value to the rest of the object
                        // or true if the object has no other properties
                        delete value.enabled;
                        if (Object.keys(value).length === 0) {
                            config[propertyName] = true;
                        }
                    }
                    else {
                        // If enabled is false, set the value to false
                        config[propertyName] = false;
                    }
                }
                // recursively translate properties of the value
                translateRemoteConfigToLocal(value);
            }
            catch (e) {
                // a failure here means that an accessor threw an error
                // so don't translate it
                // TODO(diagnostics): add a diagnostic event for this
            }
        }
    }
    catch (e_1_1) { e_1 = { error: e_1_1 }; }
    finally {
        try {
            if (propertyNames_1_1 && !propertyNames_1_1.done && (_a = propertyNames_1.return)) _a.call(propertyNames_1);
        }
        finally { if (e_1) throw e_1.error; }
    }
    // translate remote responseHeaders and requestHeaders to local responseHeaders and requestHeaders
    try {
        if ((_f = (_e = (_d = config.autocapture) === null || _d === void 0 ? void 0 : _d.networkTracking) === null || _e === void 0 ? void 0 : _e.captureRules) === null || _f === void 0 ? void 0 : _f.length) {
            try {
                for (var _k = __values(config.autocapture.networkTracking.captureRules), _l = _k.next(); !_l.done; _l = _k.next()) {
                    var rule = _l.value;
                    try {
                        for (var _m = (e_3 = void 0, __values(['responseHeaders', 'requestHeaders'])), _o = _m.next(); !_o.done; _o = _m.next()) {
                            var header = _o.value;
                            var _p = (_g = rule[header]) !== null && _g !== void 0 ? _g : {}, captureSafeHeaders = _p.captureSafeHeaders, allowlist = _p.allowlist;
                            if (!captureSafeHeaders && !allowlist) {
                                continue;
                            }
                            // if allowlist is not an array, remote config contract is violated, remove it
                            if (allowlist !== undefined && !Array.isArray(allowlist)) {
                                delete rule[header];
                                continue;
                            }
                            rule[header] = __spreadArray(__spreadArray([], __read((captureSafeHeaders ? SAFE_HEADERS : [])), false), __read((allowlist !== null && allowlist !== void 0 ? allowlist : [])), false);
                        }
                    }
                    catch (e_3_1) { e_3 = { error: e_3_1 }; }
                    finally {
                        try {
                            if (_o && !_o.done && (_c = _m.return)) _c.call(_m);
                        }
                        finally { if (e_3) throw e_3.error; }
                    }
                }
            }
            catch (e_2_1) { e_2 = { error: e_2_1 }; }
            finally {
                try {
                    if (_l && !_l.done && (_b = _k.return)) _b.call(_k);
                }
                finally { if (e_2) throw e_2.error; }
            }
        }
    }
    catch (e) {
        /* istanbul ignore next */
        // surprise exception, so don't translate it
    }
    // translate frustrationInteractions pluralization
    var frustrationInteractions = (_h = config.autocapture) === null || _h === void 0 ? void 0 : _h.frustrationInteractions;
    if (frustrationInteractions) {
        if (frustrationInteractions.rageClick) {
            frustrationInteractions.rageClicks = frustrationInteractions.rageClick;
            delete frustrationInteractions.rageClick;
        }
        if (frustrationInteractions.deadClick) {
            frustrationInteractions.deadClicks = frustrationInteractions.deadClick;
            delete frustrationInteractions.deadClick;
        }
    }
    // normalize viewportContentUpdated inside elementInteractions
    try {
        var elementInteractions = (_j = config.autocapture) === null || _j === void 0 ? void 0 : _j.elementInteractions;
        if (elementInteractions && typeof elementInteractions === 'object') {
            // { enabled: true } (no other fields) collapses to `true`; convert back to {} for the SDK.
            if (elementInteractions.viewportContentUpdated === true) {
                elementInteractions.viewportContentUpdated = {};
            }
            // { enabled: false, ... } collapses to `false`; convert back to { enabled: false } for the SDK.
            if (elementInteractions.viewportContentUpdated === false) {
                elementInteractions.viewportContentUpdated = { enabled: false };
            }
            // Migrate deprecated top-level exposureDuration to viewportContentUpdated.exposureDuration
            if (elementInteractions.exposureDuration !== undefined) {
                var viewportContentUpdated = elementInteractions.viewportContentUpdated;
                if (viewportContentUpdated === undefined) {
                    elementInteractions.viewportContentUpdated = { exposureDuration: elementInteractions.exposureDuration };
                }
                else if (typeof viewportContentUpdated === 'object' &&
                    viewportContentUpdated.exposureDuration === undefined &&
                    viewportContentUpdated.enabled !== false) {
                    viewportContentUpdated.exposureDuration = elementInteractions.exposureDuration;
                }
                delete elementInteractions.exposureDuration;
            }
        }
    }
    catch (e) {
        /* istanbul ignore next */
        // surprise exception, so don't translate it
    }
}
function mergeUrls(urlsExact, urlsRegex, browserConfig) {
    var e_4, _a;
    // Convert string patterns to RegExp objects, warn on invalid patterns and skip them
    var regexList = [];
    try {
        for (var _b = __values(urlsRegex !== null && urlsRegex !== void 0 ? urlsRegex : []), _c = _b.next(); !_c.done; _c = _b.next()) {
            var pattern = _c.value;
            try {
                regexList.push(new RegExp(pattern));
            }
            catch (regexError) {
                browserConfig.loggerProvider.warn("Invalid regex pattern: ".concat(pattern), regexError);
            }
        }
    }
    catch (e_4_1) { e_4 = { error: e_4_1 }; }
    finally {
        try {
            if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
        }
        finally { if (e_4) throw e_4.error; }
    }
    return urlsExact.concat(regexList);
}
/**
 * Updates the browser config in place by applying remote configuration settings.
 * Primarily merges autocapture settings from the remote config into the browser config.
 *
 * @param remoteConfig - The remote configuration to apply, or null if none available
 * @param browserConfig - The browser config object to update (modified in place)
 */
export function updateBrowserConfigWithRemoteConfig(remoteConfig, browserConfig) {
    var e_5, _a;
    var _b, _c, _d, _e, _f;
    if (!remoteConfig) {
        return;
    }
    // translate remote config to local compatible format
    translateRemoteConfigToLocal(remoteConfig);
    try {
        browserConfig.loggerProvider.debug('Update browser config with remote configuration:', JSON.stringify(remoteConfig));
        // type cast error will be thrown if remoteConfig is not a valid RemoteConfigBrowserSDK
        // and it will be caught by the try-catch block
        var typedRemoteConfig = remoteConfig;
        // merge remoteConfig.autocapture and browserConfig.autocapture
        // if a field is in remoteConfig.autocapture, use that value
        // if a field is not in remoteConfig.autocapture, use the value from browserConfig.autocapture
        if (typedRemoteConfig && 'autocapture' in typedRemoteConfig) {
            if (typeof typedRemoteConfig.autocapture === 'boolean') {
                browserConfig.autocapture = typedRemoteConfig.autocapture;
            }
            if (typeof typedRemoteConfig.autocapture === 'object' && typedRemoteConfig.autocapture !== null) {
                var transformedAutocaptureRemoteConfig = __assign({}, typedRemoteConfig.autocapture);
                if (browserConfig.autocapture === undefined) {
                    browserConfig.autocapture = typedRemoteConfig.autocapture;
                }
                // Handle Element Interactions config initialization
                if (typeof typedRemoteConfig.autocapture.elementInteractions === 'object' &&
                    typedRemoteConfig.autocapture.elementInteractions !== null &&
                    ((_b = typedRemoteConfig.autocapture.elementInteractions.pageUrlAllowlistRegex) === null || _b === void 0 ? void 0 : _b.length)) {
                    transformedAutocaptureRemoteConfig.elementInteractions = __assign({}, typedRemoteConfig.autocapture.elementInteractions);
                    var transformedRcElementInteractions = transformedAutocaptureRemoteConfig.elementInteractions;
                    // combine exact allow list and regex allow list into just 'pageUrlAllowlist'
                    var exactAllowList = (_c = transformedRcElementInteractions.pageUrlAllowlist) !== null && _c !== void 0 ? _c : [];
                    var urlsRegex = typedRemoteConfig.autocapture.elementInteractions.pageUrlAllowlistRegex;
                    transformedRcElementInteractions.pageUrlAllowlist = mergeUrls(exactAllowList, urlsRegex, browserConfig);
                    // clean up the regex allow list
                    delete transformedRcElementInteractions.pageUrlAllowlistRegex;
                }
                // Handle Network Tracking config initialization
                if (typeof typedRemoteConfig.autocapture.networkTracking === 'object' &&
                    typedRemoteConfig.autocapture.networkTracking !== null &&
                    ((_d = typedRemoteConfig.autocapture.networkTracking.captureRules) === null || _d === void 0 ? void 0 : _d.length)) {
                    transformedAutocaptureRemoteConfig.networkTracking = __assign({}, typedRemoteConfig.autocapture.networkTracking);
                    var transformedRcNetworkTracking = transformedAutocaptureRemoteConfig.networkTracking;
                    /* istanbul ignore next */
                    var captureRules = (_e = transformedRcNetworkTracking.captureRules) !== null && _e !== void 0 ? _e : [];
                    try {
                        for (var captureRules_1 = __values(captureRules), captureRules_1_1 = captureRules_1.next(); !captureRules_1_1.done; captureRules_1_1 = captureRules_1.next()) {
                            var rule = captureRules_1_1.value;
                            rule.urls = mergeUrls((_f = rule.urls) !== null && _f !== void 0 ? _f : [], rule.urlsRegex, browserConfig);
                            delete rule.urlsRegex;
                        }
                    }
                    catch (e_5_1) { e_5 = { error: e_5_1 }; }
                    finally {
                        try {
                            if (captureRules_1_1 && !captureRules_1_1.done && (_a = captureRules_1.return)) _a.call(captureRules_1);
                        }
                        finally { if (e_5) throw e_5.error; }
                    }
                }
                if (typeof browserConfig.autocapture === 'boolean') {
                    browserConfig.autocapture = __assign({ attribution: browserConfig.autocapture, fileDownloads: browserConfig.autocapture, formInteractions: browserConfig.autocapture, pageViews: browserConfig.autocapture, sessions: browserConfig.autocapture, elementInteractions: browserConfig.autocapture, webVitals: browserConfig.autocapture, frustrationInteractions: browserConfig.autocapture }, transformedAutocaptureRemoteConfig);
                }
                if (typeof browserConfig.autocapture === 'object') {
                    browserConfig.autocapture = __assign(__assign({}, browserConfig.autocapture), transformedAutocaptureRemoteConfig);
                }
            }
            // Override default tracking options if autocapture is updated by remote config
            browserConfig.defaultTracking = browserConfig.autocapture;
        }
        if ('customEnrichment' in typedRemoteConfig && typedRemoteConfig.customEnrichment !== null) {
            // Respect a locally-explicit false: if the user disabled custom enrichment at init time,
            // remote config must not re-enable it.
            if (browserConfig.customEnrichment !== false) {
                browserConfig.customEnrichment = typedRemoteConfig.customEnrichment;
            }
        }
        browserConfig.loggerProvider.debug('Browser config after remote config update:', JSON.stringify(browserConfig));
    }
    catch (e) {
        browserConfig.loggerProvider.error('Failed to apply remote configuration because of error: ', e);
    }
}
//# sourceMappingURL=joined-config.js.map