"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.createMouseMoveObservable = exports.createErrorObservable = exports.createExposureObservable = exports.createScrollObservable = exports.createClickObservable = exports.createMutationObservable = void 0;
var tslib_1 = require("tslib");
var analytics_core_1 = require("@amplitude/analytics-core");
/* eslint-disable-next-line no-restricted-globals */
var globalScope = (0, analytics_core_1.getGlobalScope)();
var createMutationObservable = function () {
    return new analytics_core_1.Observable(function (observer) {
        var mutationObserver = new MutationObserver(function (mutations) {
            observer.next(mutations);
        });
        if (document.body) {
            mutationObserver.observe(document.body, {
                childList: true,
                attributes: true,
                characterData: true,
                subtree: true,
            });
        }
        return function () { return mutationObserver.disconnect(); };
    });
};
exports.createMutationObservable = createMutationObservable;
/**
 * Creates an observable that tracks click events on the document.
 * @param clickType - The type of click event to track (click or pointerdown)
 */
var createClickObservable = function (clickType) {
    if (clickType === void 0) { clickType = 'click'; }
    return new analytics_core_1.Observable(function (observer) {
        var _a;
        var handler = function (event) {
            observer.next(event);
        };
        (_a = (0, analytics_core_1.getGlobalScope)()) === null || _a === void 0 ? void 0 : _a.document.addEventListener(clickType, handler, { capture: true });
        return function () {
            var _a;
            (_a = (0, analytics_core_1.getGlobalScope)()) === null || _a === void 0 ? void 0 : _a.document.removeEventListener(clickType, handler, { capture: true });
        };
    });
};
exports.createClickObservable = createClickObservable;
var createScrollObservable = function () {
    return new analytics_core_1.Observable(function (observer) {
        var _a;
        var handler = function (event) {
            observer.next(event);
        };
        (_a = (0, analytics_core_1.getGlobalScope)()) === null || _a === void 0 ? void 0 : _a.addEventListener('scroll', handler);
        return function () {
            var _a;
            (_a = (0, analytics_core_1.getGlobalScope)()) === null || _a === void 0 ? void 0 : _a.removeEventListener('scroll', handler);
        };
    });
};
exports.createScrollObservable = createScrollObservable;
var createConsoleErrorObservable = function () {
    return new analytics_core_1.Observable(function (observer) {
        var handler = function (_) {
            var args = [];
            for (var _i = 1; _i < arguments.length; _i++) {
                args[_i - 1] = arguments[_i];
            }
            /* istanbul ignore next */
            var message = undefined;
            if (Array.isArray(args[0]) && typeof args[0][0] === 'string') {
                message = args[0][0];
            }
            observer.next({ kind: 'console', message: message });
        };
        analytics_core_1.consoleObserver.addListener('error', handler);
        return function () {
            analytics_core_1.consoleObserver.removeListener(handler);
        };
    });
};
// Tracks when a trackedelement is exposed to the viewport
var createExposureObservable = function (mutationObservable, selectorAllowlist) {
    return new analytics_core_1.Observable(function (observer) {
        var _a;
        var globalScope = (0, analytics_core_1.getGlobalScope)();
        if (!(globalScope === null || globalScope === void 0 ? void 0 : globalScope.IntersectionObserver)) {
            return function () {
                return;
            };
        }
        var intersectionObserver = new IntersectionObserver(function (entries) {
            entries.forEach(function (entry) {
                observer.next(entry);
            });
        }, {
            root: null,
            rootMargin: '0px',
            threshold: 1.0, // trigger when 100% of the element is visible
        });
        // Observe initial elements
        var selectorString = selectorAllowlist.join(',');
        /* istanbul ignore next */
        var initialElements = (_a = globalScope === null || globalScope === void 0 ? void 0 : globalScope.document.querySelectorAll(selectorString)) !== null && _a !== void 0 ? _a : [];
        initialElements.forEach(function (element) {
            intersectionObserver.observe(element);
        });
        // Use mutation observable to observe new elements that match the allowlist
        var mutationSubscription = mutationObservable.subscribe(function (_a) {
            var event = _a.event;
            return event.forEach(function (_a) {
                var addedNodes = _a.addedNodes;
                return addedNodes.forEach(function (node) {
                    if (!(node instanceof Element)) {
                        return;
                    }
                    if (node.matches(selectorString)) {
                        intersectionObserver.observe(node);
                    }
                    node.querySelectorAll(selectorString).forEach(function (child) {
                        intersectionObserver.observe(child);
                    });
                });
            });
        });
        return function () {
            mutationSubscription.unsubscribe();
            intersectionObserver.disconnect();
        };
    });
};
exports.createExposureObservable = createExposureObservable;
var createUnhandledErrorObservable = function () {
    return new analytics_core_1.Observable(function (observer) {
        var handler = function (event) {
            if (!(event instanceof ErrorEvent)) {
                return;
            }
            var output = {
                kind: 'error',
            };
            if (event.error instanceof Error || event.error instanceof DOMException) {
                output = tslib_1.__assign(tslib_1.__assign({}, output), { message: event.error.message, stack: event.error.stack, filename: event.filename, lineNumber: event.lineno, columnNumber: event.colno });
            }
            else if (typeof event.error === 'string') {
                output.message = event.error;
            }
            observer.next(output);
        };
        globalScope.addEventListener('error', handler);
        return function () {
            globalScope.removeEventListener('error', handler);
        };
    });
};
var createUnhandledRejectionObservable = function () {
    return new analytics_core_1.Observable(function (observer) {
        var handler = function (event) {
            var output = {
                kind: 'unhandledrejection',
            };
            if (event.reason instanceof Error || event.reason instanceof DOMException) {
                output.message = event.reason.message;
                output.stack = event.reason.stack;
            }
            else if (typeof event.reason === 'string') {
                output.message = event.reason;
            }
            observer.next(output);
        };
        globalScope.addEventListener('unhandledrejection', handler);
        return function () {
            globalScope.removeEventListener('unhandledrejection', handler);
        };
    });
};
var createErrorObservable = function () {
    var unhandledErrorObservable = (0, analytics_core_1.merge)(createUnhandledErrorObservable(), createUnhandledRejectionObservable());
    return (0, analytics_core_1.merge)(unhandledErrorObservable, createConsoleErrorObservable());
};
exports.createErrorObservable = createErrorObservable;
var createMouseMoveObservable = function () {
    return new analytics_core_1.Observable(function (observer) {
        var handler = function (event) {
            observer.next(event);
        };
        var args = { capture: true };
        globalScope.document.addEventListener('mousemove', handler, args);
        return function () {
            globalScope.document.removeEventListener('mousemove', handler, args);
        };
    });
};
exports.createMouseMoveObservable = createMouseMoveObservable;
//# sourceMappingURL=observables.js.map