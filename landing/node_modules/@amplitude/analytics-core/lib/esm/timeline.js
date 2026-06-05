import { __assign, __awaiter, __generator, __read, __values } from "tslib";
import { buildResult } from './utils/result-builder';
import { UUID } from './utils/uuid';
var Timeline = /** @class */ (function () {
    function Timeline(client) {
        this.client = client;
        this.queue = [];
        // Flag to guarantee one schedule apply is running
        this.applying = false;
        // Flag indicates whether timeline is ready to process event
        // Events collected before timeline is ready will stay in the queue to be processed later
        this.plugins = [];
        this._optOutListeners = [];
    }
    Timeline.prototype.register = function (plugin, config) {
        var _a, _b;
        return __awaiter(this, void 0, void 0, function () {
            return __generator(this, function (_c) {
                switch (_c.label) {
                    case 0:
                        if (this.plugins.some(function (existingPlugin) { return existingPlugin.name === plugin.name; })) {
                            // eslint-disable-next-line @typescript-eslint/restrict-template-expressions
                            this.loggerProvider.warn("Plugin with name ".concat(plugin.name, " already exists, skipping registration"));
                            return [2 /*return*/];
                        }
                        if (plugin.name === undefined) {
                            plugin.name = UUID();
                            this.loggerProvider.warn("Plugin name is undefined. \n      Generating a random UUID for plugin name: ".concat(plugin.name, ". \n      Set a name for the plugin to prevent it from being added multiple times."));
                        }
                        plugin.type = (_a = plugin.type) !== null && _a !== void 0 ? _a : 'enrichment';
                        return [4 /*yield*/, ((_b = plugin.setup) === null || _b === void 0 ? void 0 : _b.call(plugin, config, this.client))];
                    case 1:
                        _c.sent();
                        this.plugins.push(plugin);
                        return [2 /*return*/];
                }
            });
        });
    };
    Timeline.prototype.deregister = function (pluginName, config) {
        var _a;
        return __awaiter(this, void 0, void 0, function () {
            var index, plugin;
            return __generator(this, function (_b) {
                switch (_b.label) {
                    case 0:
                        index = this.plugins.findIndex(function (plugin) { return plugin.name === pluginName; });
                        if (index === -1) {
                            config.loggerProvider.warn("Plugin with name ".concat(pluginName, " does not exist, skipping deregistration"));
                            return [2 /*return*/];
                        }
                        plugin = this.plugins[index];
                        this.plugins.splice(index, 1);
                        return [4 /*yield*/, ((_a = plugin.teardown) === null || _a === void 0 ? void 0 : _a.call(plugin))];
                    case 1:
                        _b.sent();
                        return [2 /*return*/];
                }
            });
        });
    };
    Timeline.prototype.reset = function (client) {
        this._clearOptOutListeners();
        this.applying = false;
        var plugins = this.plugins;
        plugins.map(function (plugin) { var _a; return (_a = plugin.teardown) === null || _a === void 0 ? void 0 : _a.call(plugin); });
        this.plugins = [];
        this.client = client;
    };
    Timeline.prototype.push = function (event) {
        var _this = this;
        return new Promise(function (resolve) {
            _this.queue.push([event, resolve]);
            _this.scheduleApply(0);
        });
    };
    Timeline.prototype.scheduleApply = function (timeout) {
        var _this = this;
        if (this.applying)
            return;
        this.applying = true;
        setTimeout(function () {
            void _this.apply(_this.queue.shift()).then(function () {
                _this.applying = false;
                if (_this.queue.length > 0) {
                    _this.scheduleApply(0);
                }
            });
        }, timeout);
    };
    Timeline.prototype.apply = function (item) {
        return __awaiter(this, void 0, void 0, function () {
            var _a, event, _b, resolve, before, before_1, before_1_1, plugin, e, e_1_1, enrichment, enrichment_1, enrichment_1_1, plugin, e, e_2_1, destination, executeDestinations;
            var e_1, _c, e_2, _d;
            return __generator(this, function (_e) {
                switch (_e.label) {
                    case 0:
                        if (!item) {
                            return [2 /*return*/];
                        }
                        _a = __read(item, 1), event = _a[0];
                        _b = __read(item, 2), resolve = _b[1];
                        // Log initial event
                        this.loggerProvider.log('Timeline.apply: Initial event', event);
                        before = this.plugins.filter(function (plugin) { return plugin.type === 'before'; });
                        _e.label = 1;
                    case 1:
                        _e.trys.push([1, 6, 7, 8]);
                        before_1 = __values(before), before_1_1 = before_1.next();
                        _e.label = 2;
                    case 2:
                        if (!!before_1_1.done) return [3 /*break*/, 5];
                        plugin = before_1_1.value;
                        /* istanbul ignore if */
                        if (!plugin.execute) {
                            // do nothing
                            return [3 /*break*/, 4];
                        }
                        return [4 /*yield*/, plugin.execute(__assign({}, event))];
                    case 3:
                        e = _e.sent();
                        if (e === null) {
                            this.loggerProvider.log("Timeline.apply: Event filtered out by before plugin '".concat(String(plugin.name), "', event: ").concat(JSON.stringify(event)));
                            resolve({ event: event, code: 0, message: '' });
                            return [2 /*return*/];
                        }
                        else {
                            event = e;
                            this.loggerProvider.log("Timeline.apply: Event after before plugin '".concat(String(plugin.name), "', event: ").concat(JSON.stringify(event)));
                        }
                        _e.label = 4;
                    case 4:
                        before_1_1 = before_1.next();
                        return [3 /*break*/, 2];
                    case 5: return [3 /*break*/, 8];
                    case 6:
                        e_1_1 = _e.sent();
                        e_1 = { error: e_1_1 };
                        return [3 /*break*/, 8];
                    case 7:
                        try {
                            if (before_1_1 && !before_1_1.done && (_c = before_1.return)) _c.call(before_1);
                        }
                        finally { if (e_1) throw e_1.error; }
                        return [7 /*endfinally*/];
                    case 8:
                        enrichment = this.plugins.filter(function (plugin) { return plugin.type === 'enrichment' || plugin.type === undefined; });
                        _e.label = 9;
                    case 9:
                        _e.trys.push([9, 14, 15, 16]);
                        enrichment_1 = __values(enrichment), enrichment_1_1 = enrichment_1.next();
                        _e.label = 10;
                    case 10:
                        if (!!enrichment_1_1.done) return [3 /*break*/, 13];
                        plugin = enrichment_1_1.value;
                        /* istanbul ignore if */
                        if (!plugin.execute) {
                            // do nothing
                            return [3 /*break*/, 12];
                        }
                        return [4 /*yield*/, plugin.execute(__assign({}, event))];
                    case 11:
                        e = _e.sent();
                        if (e === null) {
                            this.loggerProvider.log("Timeline.apply: Event filtered out by enrichment plugin '".concat(String(plugin.name), "', event: ").concat(JSON.stringify(event)));
                            resolve({ event: event, code: 0, message: '' });
                            return [2 /*return*/];
                        }
                        else {
                            event = e;
                            this.loggerProvider.log("Timeline.apply: Event after enrichment plugin '".concat(String(plugin.name), "', event: ").concat(JSON.stringify(event)));
                        }
                        _e.label = 12;
                    case 12:
                        enrichment_1_1 = enrichment_1.next();
                        return [3 /*break*/, 10];
                    case 13: return [3 /*break*/, 16];
                    case 14:
                        e_2_1 = _e.sent();
                        e_2 = { error: e_2_1 };
                        return [3 /*break*/, 16];
                    case 15:
                        try {
                            if (enrichment_1_1 && !enrichment_1_1.done && (_d = enrichment_1.return)) _d.call(enrichment_1);
                        }
                        finally { if (e_2) throw e_2.error; }
                        return [7 /*endfinally*/];
                    case 16:
                        destination = this.plugins.filter(function (plugin) { return plugin.type === 'destination'; });
                        // Log final event before sending to destinations
                        this.loggerProvider.log("Timeline.apply: Final event before destinations, event: ".concat(JSON.stringify(event)));
                        executeDestinations = destination.map(function (plugin) {
                            var eventClone = __assign({}, event);
                            return plugin.execute(eventClone).catch(function (e) { return buildResult(eventClone, 0, String(e)); });
                        });
                        void Promise.all(executeDestinations).then(function (_a) {
                            var _b = __read(_a, 1), result = _b[0];
                            var resolveResult = result || buildResult(event, 100, 'Event not tracked, no destination plugins on the instance');
                            resolve(resolveResult);
                        });
                        return [2 /*return*/];
                }
            });
        });
    };
    Timeline.prototype.flush = function () {
        return __awaiter(this, void 0, void 0, function () {
            var queue, destination, executeDestinations;
            var _this = this;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        queue = this.queue;
                        this.queue = [];
                        return [4 /*yield*/, Promise.all(queue.map(function (item) { return _this.apply(item); }))];
                    case 1:
                        _a.sent();
                        destination = this.plugins.filter(function (plugin) { return plugin.type === 'destination'; });
                        executeDestinations = destination.map(function (plugin) {
                            return plugin.flush && plugin.flush();
                        });
                        return [4 /*yield*/, Promise.all(executeDestinations)];
                    case 2:
                        _a.sent();
                        return [2 /*return*/];
                }
            });
        });
    };
    Timeline.prototype.addOptOutListener = function (cb) {
        this._optOutListeners.push(cb);
    };
    Timeline.prototype._clearOptOutListeners = function () {
        this._optOutListeners = [];
    };
    Timeline.prototype.onIdentityChanged = function (identity) {
        this.plugins.forEach(function (plugin) {
            var _a;
            // Intentionally to not await plugin.onIdentityChanged() for non-blocking.
            // Ignore optional channing next line for test coverage.
            // If the plugin doesn't implement it, it won't be called.
            /* istanbul ignore next */
            void ((_a = plugin.onIdentityChanged) === null || _a === void 0 ? void 0 : _a.call(plugin, identity));
        });
    };
    Timeline.prototype.onSessionIdChanged = function (sessionId) {
        this.plugins.forEach(function (plugin) {
            var _a;
            // Intentionally to not await plugin.onSessionIdChanged() for non-blocking.
            // Ignore optional channing next line for test coverage.
            // If the plugin doesn't implement it, it won't be called.
            /* istanbul ignore next */
            void ((_a = plugin.onSessionIdChanged) === null || _a === void 0 ? void 0 : _a.call(plugin, sessionId));
        });
    };
    Timeline.prototype.onOptOutChanged = function (optOut) {
        this.plugins.forEach(function (plugin) {
            var _a;
            // Intentionally to not await plugin.onOptOutChanged() for non-blocking.
            // Ignore optional channing next line for test coverage.
            // If the plugin doesn't implement it, it won't be called.
            /* istanbul ignore next */
            void ((_a = plugin.onOptOutChanged) === null || _a === void 0 ? void 0 : _a.call(plugin, optOut));
        });
        void this._callOptOutListeners(optOut);
    };
    Timeline.prototype._callOptOutListeners = function (optOut) {
        return __awaiter(this, void 0, void 0, function () {
            var _a, _b, listener, e_3, e_4_1;
            var e_4, _c;
            return __generator(this, function (_d) {
                switch (_d.label) {
                    case 0:
                        _d.trys.push([0, 7, 8, 9]);
                        _a = __values(this._optOutListeners), _b = _a.next();
                        _d.label = 1;
                    case 1:
                        if (!!_b.done) return [3 /*break*/, 6];
                        listener = _b.value;
                        _d.label = 2;
                    case 2:
                        _d.trys.push([2, 4, , 5]);
                        return [4 /*yield*/, listener(optOut)];
                    case 3:
                        _d.sent();
                        return [3 /*break*/, 5];
                    case 4:
                        e_3 = _d.sent();
                        /* istanbul ignore next */
                        this.loggerProvider.error('Error calling optOut listener', e_3);
                        return [3 /*break*/, 5];
                    case 5:
                        _b = _a.next();
                        return [3 /*break*/, 1];
                    case 6: return [3 /*break*/, 9];
                    case 7:
                        e_4_1 = _d.sent();
                        e_4 = { error: e_4_1 };
                        return [3 /*break*/, 9];
                    case 8:
                        try {
                            if (_b && !_b.done && (_c = _a.return)) _c.call(_a);
                        }
                        finally { if (e_4) throw e_4.error; }
                        return [7 /*endfinally*/];
                    case 9: return [2 /*return*/];
                }
            });
        });
    };
    Timeline.prototype.onReset = function () {
        this.plugins.forEach(function (plugin) {
            var _a;
            // Intentionally to not await plugin.onReset() for non-blocking.
            // Ignore optional channing next line for test coverage.
            // If the plugin doesn't implement it, it won't be called.
            /* istanbul ignore next */
            void ((_a = plugin.onReset) === null || _a === void 0 ? void 0 : _a.call(plugin));
        });
    };
    return Timeline;
}());
export { Timeline };
//# sourceMappingURL=timeline.js.map