"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.fileDownloadTracking = void 0;
var tslib_1 = require("tslib");
var constants_1 = require("../constants");
var analytics_core_1 = require("@amplitude/analytics-core");
var fileDownloadTracking = function () {
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
    var name = '@amplitude/plugin-file-download-tracking-browser';
    var type = 'enrichment';
    var setup = function (config, amplitude) { return tslib_1.__awaiter(void 0, void 0, void 0, function () {
        var initializeFileDownloadTracking, window_1;
        return tslib_1.__generator(this, function (_a) {
            initializeFileDownloadTracking = function () {
                /* istanbul ignore if */
                if (!amplitude) {
                    // TODO: Add required minimum version of @amplitude/analytics-browser
                    config.loggerProvider.warn('File download tracking requires a later version of @amplitude/analytics-browser. File download events are not tracked.');
                    return;
                }
                /* istanbul ignore if */
                if (typeof document === 'undefined') {
                    return;
                }
                var addFileDownloadListener = function (a) {
                    var url;
                    try {
                        // eslint-disable-next-line no-restricted-globals
                        url = new URL(a.href, window.location.href);
                    }
                    catch (_a) {
                        /* istanbul ignore next */
                        return;
                    }
                    var result = ext.exec(url.href);
                    var fileExtension = result === null || result === void 0 ? void 0 : result[1];
                    if (fileExtension) {
                        addEventListener(a, 'click', function () {
                            var _a;
                            if (fileExtension) {
                                amplitude.track(constants_1.DEFAULT_FILE_DOWNLOAD_EVENT, (_a = {},
                                    _a[constants_1.FILE_EXTENSION] = fileExtension,
                                    _a[constants_1.FILE_NAME] = url.pathname,
                                    _a[constants_1.LINK_ID] = a.id,
                                    _a[constants_1.LINK_TEXT] = a.text,
                                    _a[constants_1.LINK_URL] = a.href,
                                    _a));
                            }
                        });
                    }
                };
                var ext = /\.(pdf|xlsx?|docx?|txt|rtf|csv|exe|key|pp(s|t|tx)|7z|pkg|rar|gz|zip|avi|mov|mp4|mpe?g|wmv|midi?|mp3|wav|wma)(\?.+)?$/;
                // Adds listener to existing anchor tags
                var links = Array.from(document.getElementsByTagName('a'));
                links.forEach(addFileDownloadListener);
                // Adds listener to anchor tags added after initial load
                /* istanbul ignore else */
                if (typeof MutationObserver !== 'undefined') {
                    observer = new MutationObserver(function (mutations) {
                        mutations.forEach(function (mutation) {
                            mutation.addedNodes.forEach(function (node) {
                                if (node.nodeName === 'A') {
                                    addFileDownloadListener(node);
                                }
                                if ('querySelectorAll' in node && typeof node.querySelectorAll === 'function') {
                                    Array.from(node.querySelectorAll('a')).map(addFileDownloadListener);
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
            /* istanbul ignore else*/
            if (document.readyState === 'complete') {
                initializeFileDownloadTracking();
            }
            else {
                window_1 = (0, analytics_core_1.getGlobalScope)();
                /* istanbul ignore else*/
                if (window_1) {
                    window_1.addEventListener('load', initializeFileDownloadTracking);
                }
                else {
                    config.loggerProvider.debug('File download tracking is not installed because global is undefined.');
                }
            }
            return [2 /*return*/];
        });
    }); };
    var execute = function (event) { return tslib_1.__awaiter(void 0, void 0, void 0, function () { return tslib_1.__generator(this, function (_a) {
        return [2 /*return*/, event];
    }); }); };
    var teardown = function () { return tslib_1.__awaiter(void 0, void 0, void 0, function () {
        return tslib_1.__generator(this, function (_a) {
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
exports.fileDownloadTracking = fileDownloadTracking;
//# sourceMappingURL=file-download-tracking.js.map