import { __assign } from "tslib";
import { Observable, consoleObserver, getGlobalScope, merge } from '@amplitude/analytics-core';
/* eslint-disable-next-line no-restricted-globals */
var globalScope = getGlobalScope();
export var createMutationObservable = function () {
    return new Observable(function (observer) {
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
/**
 * Creates an observable that tracks click events on the document.
 * @param clickType - The type of click event to track (click or pointerdown)
 */
export var createClickObservable = function (clickType) {
    if (clickType === void 0) { clickType = 'click'; }
    return new Observable(function (observer) {
        var _a;
        var handler = function (event) {
            observer.next(event);
        };
        (_a = getGlobalScope()) === null || _a === void 0 ? void 0 : _a.document.addEventListener(clickType, handler, { capture: true });
        return function () {
            var _a;
            (_a = getGlobalScope()) === null || _a === void 0 ? void 0 : _a.document.removeEventListener(clickType, handler, { capture: true });
        };
    });
};
export var createScrollObservable = function () {
    return new Observable(function (observer) {
        var _a;
        var handler = function (event) {
            observer.next(event);
        };
        (_a = getGlobalScope()) === null || _a === void 0 ? void 0 : _a.addEventListener('scroll', handler);
        return function () {
            var _a;
            (_a = getGlobalScope()) === null || _a === void 0 ? void 0 : _a.removeEventListener('scroll', handler);
        };
    });
};
var createConsoleErrorObservable = function () {
    return new Observable(function (observer) {
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
        consoleObserver.addListener('error', handler);
        return function () {
            consoleObserver.removeListener(handler);
        };
    });
};
// Tracks when a trackedelement is exposed to the viewport
export var createExposureObservable = function (mutationObservable, selectorAllowlist) {
    return new Observable(function (observer) {
        var _a;
        var globalScope = getGlobalScope();
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
var createUnhandledErrorObservable = function () {
    return new Observable(function (observer) {
        var handler = function (event) {
            if (!(event instanceof ErrorEvent)) {
                return;
            }
            var output = {
                kind: 'error',
            };
            if (event.error instanceof Error || event.error instanceof DOMException) {
                output = __assign(__assign({}, output), { message: event.error.message, stack: event.error.stack, filename: event.filename, lineNumber: event.lineno, columnNumber: event.colno });
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
    return new Observable(function (observer) {
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
export var createErrorObservable = function () {
    var unhandledErrorObservable = merge(createUnhandledErrorObservable(), createUnhandledRejectionObservable());
    return merge(unhandledErrorObservable, createConsoleErrorObservable());
};
export var createMouseMoveObservable = function () {
    return new Observable(function (observer) {
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
//# sourceMappingURL=observables.js.map