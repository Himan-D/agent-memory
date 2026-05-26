var _a;
import { __awaiter, __generator, __values } from "tslib";
import { getGlobalScope } from '../global-scope';
import { AMPLITUDE_ORIGIN } from './constants';
import { asyncLoadScript, generateUniqueId } from './utils';
/**
 * Brand key used to identify BaseWindowMessenger instances across bundle boundaries.
 */
var MESSENGER_BRAND = '__AMPLITUDE_MESSENGER_INSTANCE__';
/** Global scope key where the singleton messenger is stored. */
var MESSENGER_GLOBAL_KEY = '__AMPLITUDE_MESSENGER__';
/**
 * BaseWindowMessenger provides generic cross-window communication via postMessage.
 * Singleton access via getOrCreateWindowMessenger() to prevent duplicate instances
 */
var BaseWindowMessenger = /** @class */ (function () {
    function BaseWindowMessenger(_b) {
        var _c = _b === void 0 ? {} : _b, _d = _c.origin, origin = _d === void 0 ? AMPLITUDE_ORIGIN : _d;
        /** Brand property for cross-bundle instanceof checks. */
        this[_a] = true;
        this.isSetup = false;
        this.messageHandler = null;
        this.requestCallbacks = {};
        this.actionHandlers = new Map();
        /**
         * Messages received for actions that had no registered handler yet.
         * Drained automatically when the corresponding handler is registered via
         * registerActionHandler(), solving startup race conditions between
         * independently-initialized plugins.
         */
        this.pendingMessages = new Map();
        /**
         * Tracks in-flight and completed script loads by URL.
         * Using a map, this prevents duplicate loads before the first resolves.
         */
        this.scriptLoadPromises = new Map();
        this.endpoint = origin;
    }
    /**
     * Send a message to the parent window (window.opener).
     */
    BaseWindowMessenger.prototype.notify = function (message) {
        var _b, _c, _d, _e;
        (_c = (_b = this.logger) === null || _b === void 0 ? void 0 : _b.debug) === null || _c === void 0 ? void 0 : _c.call(_b, 'Message sent: ', JSON.stringify(message));
        (_e = (_d = window.opener) === null || _d === void 0 ? void 0 : _d.postMessage) === null || _e === void 0 ? void 0 : _e.call(_d, message, this.endpoint);
    };
    /**
     * Send an async request to the parent window with a unique ID.
     * Returns a Promise that resolves when the parent responds.
     */
    BaseWindowMessenger.prototype.sendRequest = function (action, args, options) {
        var _this = this;
        if (options === void 0) { options = { timeout: 15000 }; }
        var id = generateUniqueId();
        var request = { id: id, action: action, args: args };
        var promise = new Promise(function (resolve, reject) {
            _this.requestCallbacks[id] = { resolve: resolve, reject: reject };
            _this.notify(request);
            if (options.timeout > 0) {
                setTimeout(function () {
                    reject(new Error("".concat(action, " timed out (id: ").concat(id, ")")));
                    delete _this.requestCallbacks[id];
                }, options.timeout);
            }
        });
        return promise;
    };
    /**
     * Handle a response to a previous request by resolving its Promise.
     */
    BaseWindowMessenger.prototype.handleResponse = function (response) {
        var _b;
        if (!this.requestCallbacks[response.id]) {
            (_b = this.logger) === null || _b === void 0 ? void 0 : _b.warn("No callback found for request id: ".concat(response.id));
            return;
        }
        this.requestCallbacks[response.id].resolve(response.responseData);
        delete this.requestCallbacks[response.id];
    };
    /**
     * Register a handler for a specific action type.
     * Logs a warning if overwriting an existing handler.
     */
    BaseWindowMessenger.prototype.registerActionHandler = function (action, handler) {
        var e_1, _b;
        var _c, _d;
        if (this.actionHandlers.has(action)) {
            (_d = (_c = this.logger) === null || _c === void 0 ? void 0 : _c.warn) === null || _d === void 0 ? void 0 : _d.call(_c, "Overwriting existing action handler for: ".concat(action));
        }
        this.actionHandlers.set(action, handler);
        // Replay any messages that arrived before this handler was registered
        var queued = this.pendingMessages.get(action);
        if (queued) {
            this.pendingMessages.delete(action);
            try {
                for (var queued_1 = __values(queued), queued_1_1 = queued_1.next(); !queued_1_1.done; queued_1_1 = queued_1.next()) {
                    var data = queued_1_1.value;
                    handler(data);
                }
            }
            catch (e_1_1) { e_1 = { error: e_1_1 }; }
            finally {
                try {
                    if (queued_1_1 && !queued_1_1.done && (_b = queued_1.return)) _b.call(queued_1);
                }
                finally { if (e_1) throw e_1.error; }
            }
        }
    };
    /**
     * Load a script once, deduplicating by URL.
     * Safe against concurrent calls — the second call awaits the first's in-flight Promise
     * rather than triggering a duplicate load.
     */
    BaseWindowMessenger.prototype.loadScriptOnce = function (url) {
        return __awaiter(this, void 0, void 0, function () {
            var existing, loadPromise, error_1;
            return __generator(this, function (_b) {
                switch (_b.label) {
                    case 0:
                        existing = this.scriptLoadPromises.get(url);
                        if (existing) {
                            return [2 /*return*/, existing];
                        }
                        loadPromise = asyncLoadScript(url).then(function () {
                            // Resolve to void
                        });
                        this.scriptLoadPromises.set(url, loadPromise);
                        _b.label = 1;
                    case 1:
                        _b.trys.push([1, 3, , 4]);
                        return [4 /*yield*/, loadPromise];
                    case 2:
                        _b.sent();
                        return [3 /*break*/, 4];
                    case 3:
                        error_1 = _b.sent();
                        // Remove failed loads so they can be retried
                        this.scriptLoadPromises.delete(url);
                        throw error_1;
                    case 4: return [2 /*return*/];
                }
            });
        });
    };
    /**
     * Set up the message listener. Idempotent — safe to call multiple times.
     * Subclasses should call super.setup() and then register their own action handlers.
     */
    BaseWindowMessenger.prototype.setup = function (_b) {
        var _this = this;
        var _c, _d;
        var _e = _b === void 0 ? {} : _b, logger = _e.logger, endpoint = _e.endpoint;
        if (logger) {
            this.logger = logger;
        }
        // If endpoint is customized, don't override a previously customized endpoint.
        if (endpoint && this.endpoint === AMPLITUDE_ORIGIN) {
            this.endpoint = endpoint;
        }
        // Only attach the message listener once
        if (this.isSetup) {
            return;
        }
        this.isSetup = true;
        (_d = (_c = this.logger) === null || _c === void 0 ? void 0 : _c.debug) === null || _d === void 0 ? void 0 : _d.call(_c, 'Setting up messenger');
        // Attach Event Listener to listen for messages from the parent window
        this.messageHandler = function (event) {
            var _b, _c, _d, _e, _f;
            (_c = (_b = _this.logger) === null || _b === void 0 ? void 0 : _b.debug) === null || _c === void 0 ? void 0 : _c.call(_b, 'Message received: ', JSON.stringify(event));
            // Only accept messages from the specified origin
            if (_this.endpoint !== event.origin) {
                return;
            }
            var eventData = event.data;
            var action = eventData === null || eventData === void 0 ? void 0 : eventData.action;
            // Ignore messages without action
            if (!action) {
                return;
            }
            // If id exists, handle responses to previous requests
            if ('id' in eventData && eventData.id) {
                (_e = (_d = _this.logger) === null || _d === void 0 ? void 0 : _d.debug) === null || _e === void 0 ? void 0 : _e.call(_d, 'Received Response to previous request: ', JSON.stringify(event));
                _this.handleResponse(eventData);
            }
            else {
                if (action === 'ping') {
                    _this.notify({ action: 'pong' });
                    return;
                }
                // Dispatch to registered action handlers, or buffer for late registration
                var handler = _this.actionHandlers.get(action);
                if (handler) {
                    handler(eventData.data);
                }
                else {
                    var queue = (_f = _this.pendingMessages.get(action)) !== null && _f !== void 0 ? _f : [];
                    queue.push(eventData.data);
                    _this.pendingMessages.set(action, queue);
                }
            }
        };
        window.addEventListener('message', this.messageHandler);
        this.notify({ action: 'page-loaded' });
    };
    /**
     * Tear down the messenger: remove the message listener, clear all state.
     */
    BaseWindowMessenger.prototype.destroy = function () {
        if (this.messageHandler) {
            window.removeEventListener('message', this.messageHandler);
            this.messageHandler = null;
        }
        this.isSetup = false;
        this.actionHandlers.clear();
        this.pendingMessages.clear();
        this.requestCallbacks = {};
        this.scriptLoadPromises.clear();
        // Remove from global scope if this is the singleton
        var globalScope = getGlobalScope();
        if ((globalScope === null || globalScope === void 0 ? void 0 : globalScope[MESSENGER_GLOBAL_KEY]) === this) {
            delete globalScope[MESSENGER_GLOBAL_KEY];
        }
    };
    return BaseWindowMessenger;
}());
_a = MESSENGER_BRAND;
/**
 * Type guard: checks whether a value is a BaseWindowMessenger instance.
 */
function isWindowMessenger(value) {
    return (typeof value === 'object' &&
        value !== null &&
        MESSENGER_BRAND in value &&
        value[MESSENGER_BRAND] === true);
}
/**
 * Get or create a singleton BaseWindowMessenger instance.
 * Ensures only one messenger (and one message listener) exists per page,
 * preventing duplicate script loads and double notifications.
 *
 * The singleton is stored on globalScope under the same MESSENGER_KEY.
 * The branded property check verifies the stored value is actually a messenger.
 */
export function getOrCreateWindowMessenger(options) {
    var globalScope = getGlobalScope();
    var existing = globalScope === null || globalScope === void 0 ? void 0 : globalScope[MESSENGER_GLOBAL_KEY];
    if (isWindowMessenger(existing)) {
        return existing;
    }
    var messenger = new BaseWindowMessenger(options);
    if (globalScope) {
        globalScope[MESSENGER_GLOBAL_KEY] = messenger;
    }
    return messenger;
}
//# sourceMappingURL=base-window-messenger.js.map