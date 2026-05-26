import { __awaiter, __generator } from "tslib";
import { DEFAULT_FORM_START_EVENT, DEFAULT_FORM_SUBMIT_EVENT, FORM_ID, FORM_NAME, FORM_DESTINATION, } from '../constants';
import { getGlobalScope, } from '@amplitude/analytics-core';
import { getFormInteractionsConfig } from '../default-tracking';
export var formInteractionTracking = function () {
    var observer;
    var eventListeners = [];
    var addEventListener = function (element, type, handler) {
        element.addEventListener(type, handler);
        eventListeners.push({
            element: element,
            type: type,
            handler: handler,
        });
    };
    var removeClickListeners = function () {
        eventListeners.forEach(function (_a) {
            var element = _a.element, type = _a.type, handler = _a.handler;
            /* istanbul ignore next */
            element === null || element === void 0 ? void 0 : element.removeEventListener(type, handler);
        });
        eventListeners = [];
    };
    var formInteractionsConfig;
    var name = '@amplitude/plugin-form-interaction-tracking-browser';
    var type = 'enrichment';
    var setup = function (config, amplitude) { return __awaiter(void 0, void 0, void 0, function () {
        var initializeFormTracking, window_1;
        return __generator(this, function (_a) {
            formInteractionsConfig = getFormInteractionsConfig(config);
            initializeFormTracking = function () {
                /* istanbul ignore if */
                if (!amplitude) {
                    // TODO: Add required minimum version of @amplitude/analytics-browser
                    config.loggerProvider.warn('Form interaction tracking requires a later version of @amplitude/analytics-browser. Form interaction events are not tracked.');
                    return;
                }
                /* istanbul ignore if */
                if (typeof document === 'undefined') {
                    return;
                }
                var addedFormNodes = new WeakSet();
                var addFormInteractionListener = function (form) {
                    if (addedFormNodes.has(form)) {
                        return;
                    }
                    addedFormNodes.add(form);
                    var hasFormChanged = false;
                    addEventListener(form, 'change', function () {
                        var _a;
                        var formDestination = extractFormAction(form);
                        if (!hasFormChanged) {
                            amplitude.track(DEFAULT_FORM_START_EVENT, (_a = {},
                                _a[FORM_ID] = stringOrUndefined(form.id),
                                _a[FORM_NAME] = stringOrUndefined(form.name),
                                _a[FORM_DESTINATION] = formDestination,
                                _a));
                        }
                        hasFormChanged = true;
                    });
                    addEventListener(form, 'submit', function (event) {
                        var _a, _b;
                        var formDestination = extractFormAction(form);
                        if (!hasFormChanged) {
                            amplitude.track(DEFAULT_FORM_START_EVENT, (_a = {},
                                _a[FORM_ID] = stringOrUndefined(form.id),
                                _a[FORM_NAME] = stringOrUndefined(form.name),
                                _a[FORM_DESTINATION] = formDestination,
                                _a));
                        }
                        hasFormChanged = true;
                        // Check if shouldTrackSubmit callback is provided and use it to determine whether to track form_submit
                        if ((formInteractionsConfig === null || formInteractionsConfig === void 0 ? void 0 : formInteractionsConfig.shouldTrackSubmit) !== undefined) {
                            if (typeof formInteractionsConfig.shouldTrackSubmit === 'function' &&
                                typeof SubmitEvent !== 'undefined' &&
                                event instanceof SubmitEvent) {
                                try {
                                    var shouldTrack = formInteractionsConfig.shouldTrackSubmit(event);
                                    if (!shouldTrack) {
                                        return;
                                    }
                                }
                                catch (e) {
                                    config.loggerProvider.warn('shouldTrackSubmit callback threw an error, proceeding with tracking.');
                                }
                            }
                            else {
                                config.loggerProvider.warn('shouldTrackSubmit is ignored because it is not a function or event is not a SubmitEvent.');
                            }
                        }
                        amplitude.track(DEFAULT_FORM_SUBMIT_EVENT, (_b = {},
                            _b[FORM_ID] = stringOrUndefined(form.id),
                            _b[FORM_NAME] = stringOrUndefined(form.name),
                            _b[FORM_DESTINATION] = formDestination,
                            _b));
                        hasFormChanged = false;
                    });
                };
                // Adds listener to existing anchor tags
                var forms = Array.from(document.getElementsByTagName('form'));
                forms.forEach(addFormInteractionListener);
                // Adds listener to anchor tags added after initial load
                /* istanbul ignore else */
                if (typeof MutationObserver !== 'undefined') {
                    observer = new MutationObserver(function (mutations) {
                        mutations.forEach(function (mutation) {
                            mutation.addedNodes.forEach(function (node) {
                                if (node.nodeName === 'FORM') {
                                    addFormInteractionListener(node);
                                }
                                if ('querySelectorAll' in node && typeof node.querySelectorAll === 'function') {
                                    Array.from(node.querySelectorAll('form')).map(addFormInteractionListener);
                                }
                            });
                        });
                    });
                    observer.observe(document.body, {
                        subtree: true,
                        childList: true,
                    });
                }
            };
            // If the document is already loaded, initialize immediately.
            if (document.readyState === 'complete') {
                initializeFormTracking();
            }
            else {
                window_1 = getGlobalScope();
                /* istanbul ignore else*/
                if (window_1) {
                    window_1.addEventListener('load', initializeFormTracking);
                }
                else {
                    config.loggerProvider.debug('Form interaction tracking is not installed because global is undefined.');
                }
            }
            return [2 /*return*/];
        });
    }); };
    var execute = function (event) { return __awaiter(void 0, void 0, void 0, function () { return __generator(this, function (_a) {
        return [2 /*return*/, event];
    }); }); };
    var teardown = function () { return __awaiter(void 0, void 0, void 0, function () {
        return __generator(this, function (_a) {
            observer === null || observer === void 0 ? void 0 : observer.disconnect();
            removeClickListeners();
            return [2 /*return*/];
        });
    }); };
    return {
        name: name,
        type: type,
        setup: setup,
        execute: execute,
        teardown: teardown,
    };
};
export var stringOrUndefined = function (name) {
    /* istanbul ignore if */
    if (typeof name !== 'string') {
        // We found instances where the value of `name` is an Element and not a string.
        // Elements may have circular references and would throw an error when passed to `JSON.stringify(...)`.
        // If a non-string value is seen, assume there is no value.
        return undefined;
    }
    return name;
};
// Extracts the form action attribute, and normalizes it to a valid URL to preserve the previous behavior of accessing the action property directly.
export var extractFormAction = function (form) {
    var formDestination = form.getAttribute('action');
    try {
        // eslint-disable-next-line no-restricted-globals
        formDestination = new URL(encodeURI(formDestination !== null && formDestination !== void 0 ? formDestination : ''), window.location.href).href;
    }
    catch (_a) {
        //
    }
    return formDestination;
};
//# sourceMappingURL=form-interaction-tracking.js.map