"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.AmplitudeCore = void 0;
var tslib_1 = require("tslib");
var event_1 = require("./types/event/event");
var identify_1 = require("./identify");
var messages_1 = require("./types/messages");
var timeline_1 = require("./timeline");
var event_builder_1 = require("./utils/event-builder");
var result_builder_1 = require("./utils/result-builder");
var return_wrapper_1 = require("./utils/return-wrapper");
var AmplitudeCore = /** @class */ (function () {
    function AmplitudeCore(name) {
        if (name === void 0) { name = '$default'; }
        this.initializing = false;
        this.isReady = false;
        this.q = [];
        this.dispatchQ = [];
        this.logEvent = this.track.bind(this);
        this.timeline = new timeline_1.Timeline(this);
        this.name = name;
    }
    AmplitudeCore.prototype._init = function (config) {
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            return tslib_1.__generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        this.config = config;
                        this.timeline.reset(this);
                        this.timeline.loggerProvider = this.config.loggerProvider;
                        return [4 /*yield*/, this.runQueuedFunctions('q')];
                    case 1:
                        _a.sent();
                        this.isReady = true;
                        return [2 /*return*/];
                }
            });
        });
    };
    AmplitudeCore.prototype.runQueuedFunctions = function (queueName) {
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            var queuedFunctions, queuedFunctions_1, queuedFunctions_1_1, queuedFunction, val, e_1_1;
            var e_1, _a;
            return tslib_1.__generator(this, function (_b) {
                switch (_b.label) {
                    case 0:
                        queuedFunctions = this[queueName];
                        this[queueName] = [];
                        _b.label = 1;
                    case 1:
                        _b.trys.push([1, 8, 9, 10]);
                        queuedFunctions_1 = tslib_1.__values(queuedFunctions), queuedFunctions_1_1 = queuedFunctions_1.next();
                        _b.label = 2;
                    case 2:
                        if (!!queuedFunctions_1_1.done) return [3 /*break*/, 7];
                        queuedFunction = queuedFunctions_1_1.value;
                        val = queuedFunction();
                        if (!(val && 'promise' in val)) return [3 /*break*/, 4];
                        return [4 /*yield*/, val.promise];
                    case 3:
                        _b.sent();
                        return [3 /*break*/, 6];
                    case 4: return [4 /*yield*/, val];
                    case 5:
                        _b.sent();
                        _b.label = 6;
                    case 6:
                        queuedFunctions_1_1 = queuedFunctions_1.next();
                        return [3 /*break*/, 2];
                    case 7: return [3 /*break*/, 10];
                    case 8:
                        e_1_1 = _b.sent();
                        e_1 = { error: e_1_1 };
                        return [3 /*break*/, 10];
                    case 9:
                        try {
                            if (queuedFunctions_1_1 && !queuedFunctions_1_1.done && (_a = queuedFunctions_1.return)) _a.call(queuedFunctions_1);
                        }
                        finally { if (e_1) throw e_1.error; }
                        return [7 /*endfinally*/];
                    case 10:
                        if (!this[queueName].length) return [3 /*break*/, 12];
                        return [4 /*yield*/, this.runQueuedFunctions(queueName)];
                    case 11:
                        _b.sent();
                        _b.label = 12;
                    case 12: return [2 /*return*/];
                }
            });
        });
    };
    AmplitudeCore.prototype.track = function (eventInput, eventProperties, eventOptions) {
        var event = (0, event_builder_1.createTrackEvent)(eventInput, eventProperties, eventOptions);
        // Update client user properties immediately and synchronously
        this.userProperties = this.getOperationAppliedUserProperties(event.user_properties);
        return (0, return_wrapper_1.returnWrapper)(this.dispatch(event));
    };
    AmplitudeCore.prototype.identify = function (identify, eventOptions) {
        var event = (0, event_builder_1.createIdentifyEvent)(identify, eventOptions);
        // Update client user properties immediately and synchronously
        this.userProperties = this.getOperationAppliedUserProperties(event.user_properties);
        return (0, return_wrapper_1.returnWrapper)(this.dispatch(event));
    };
    AmplitudeCore.prototype.groupIdentify = function (groupType, groupName, identify, eventOptions) {
        var event = (0, event_builder_1.createGroupIdentifyEvent)(groupType, groupName, identify, eventOptions);
        return (0, return_wrapper_1.returnWrapper)(this.dispatch(event));
    };
    AmplitudeCore.prototype.setGroup = function (groupType, groupName, eventOptions) {
        var event = (0, event_builder_1.createGroupEvent)(groupType, groupName, eventOptions);
        // Update client user properties immediately and synchronously
        this.userProperties = this.getOperationAppliedUserProperties(event.user_properties);
        return (0, return_wrapper_1.returnWrapper)(this.dispatch(event));
    };
    AmplitudeCore.prototype.revenue = function (revenue, eventOptions) {
        var event = (0, event_builder_1.createRevenueEvent)(revenue, eventOptions);
        return (0, return_wrapper_1.returnWrapper)(this.dispatch(event));
    };
    AmplitudeCore.prototype.add = function (plugin) {
        if (!this.isReady) {
            this.q.push(this._addPlugin.bind(this, plugin));
            return (0, return_wrapper_1.returnWrapper)();
        }
        return this._addPlugin(plugin);
    };
    AmplitudeCore.prototype._addPlugin = function (plugin) {
        return (0, return_wrapper_1.returnWrapper)(this.timeline.register(plugin, this.config));
    };
    AmplitudeCore.prototype.remove = function (pluginName) {
        if (!this.isReady) {
            this.q.push(this._removePlugin.bind(this, pluginName));
            return (0, return_wrapper_1.returnWrapper)();
        }
        return this._removePlugin(pluginName);
    };
    AmplitudeCore.prototype._removePlugin = function (pluginName) {
        return (0, return_wrapper_1.returnWrapper)(this.timeline.deregister(pluginName, this.config));
    };
    AmplitudeCore.prototype.dispatchWithCallback = function (event, callback) {
        if (!this.isReady) {
            return callback((0, result_builder_1.buildResult)(event, 0, messages_1.CLIENT_NOT_INITIALIZED));
        }
        void this.process(event).then(callback);
    };
    AmplitudeCore.prototype.dispatch = function (event) {
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            var _this = this;
            return tslib_1.__generator(this, function (_a) {
                if (!this.isReady) {
                    return [2 /*return*/, new Promise(function (resolve) {
                            _this.dispatchQ.push(_this.dispatchWithCallback.bind(_this, event, resolve));
                        })];
                }
                return [2 /*return*/, this.process(event)];
            });
        });
    };
    /**
     *
     * This method applies identify operations to user properties and
     * returns a single object representing the final user property state.
     *
     * This is a best-effort api that only supports $set, $clearAll, and $unset.
     * Other operations are not supported and are ignored.
     *
     * Operations are applied on top of current client state (this.userProperties).
     *
     * @param userProperties The new user properties object from identify() or setIdentity().
     * @returns A key-value object user properties without operations.
     *
     * @example
     * Input:
     * {
     *   $set: { plan: 'premium' },
     *   custom_flag: true
     * }
     *
     * Output:
     * {
     *   plan: 'premium',
     *   custom_flag: true
     * }
     */
    AmplitudeCore.prototype.getOperationAppliedUserProperties = function (userProperties) {
        var _a;
        var base = (_a = this.userProperties) !== null && _a !== void 0 ? _a : {};
        var updatedProperties = tslib_1.__assign({}, base);
        if (userProperties === undefined) {
            return updatedProperties;
        }
        // Keep non-operation keys for later merge
        var nonOpProperties = {};
        Object.keys(userProperties).forEach(function (key) {
            if (!Object.values(event_1.IdentifyOperation).includes(key)) {
                // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                // @ts-ignore
                // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
                nonOpProperties[key] = userProperties[key];
            }
        });
        identify_1.OrderedIdentifyOperations.forEach(function (operation) {
            // Skip when key is an operation.
            if (!Object.keys(userProperties).includes(operation))
                return;
            // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
            var opProperties = userProperties[operation];
            switch (operation) {
                case event_1.IdentifyOperation.CLEAR_ALL:
                    // Due to operation order, the following line will never execute.
                    /* istanbul ignore next */
                    Object.keys(updatedProperties).forEach(function (prop) {
                        delete updatedProperties[prop];
                    });
                    break;
                case event_1.IdentifyOperation.UNSET:
                    Object.keys(opProperties).forEach(function (prop) {
                        delete updatedProperties[prop];
                    });
                    break;
                case event_1.IdentifyOperation.SET:
                    Object.assign(updatedProperties, opProperties);
                    break;
            }
        });
        // Merge non-operation properties.
        // Custom properties should not be affected by operations.
        // https://github.com/amplitude/nova/blob/343f678ded83c032e83b189796b3c2be161b48f5/src/main/java/com/amplitude/userproperty/model/ModifyUserPropertiesIdent.java#L79-L83
        Object.assign(updatedProperties, nonOpProperties);
        return updatedProperties;
    };
    AmplitudeCore.prototype.process = function (event) {
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            var result, e_2, message, result;
            return tslib_1.__generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        _a.trys.push([0, 2, , 3]);
                        // skip event processing if opt out
                        if (this.config.optOut) {
                            return [2 /*return*/, (0, result_builder_1.buildResult)(event, 0, messages_1.OPT_OUT_MESSAGE)];
                        }
                        if (event.event_type === event_1.SpecialEventType.IDENTIFY) {
                            // Do not update this.userProperties here.
                            // It is only set synchronously in identify() or setIdentity()
                            this.timeline.onIdentityChanged({ userProperties: this.userProperties });
                        }
                        return [4 /*yield*/, this.timeline.push(event)];
                    case 1:
                        result = _a.sent();
                        result.code === 200
                            ? this.config.loggerProvider.log(result.message)
                            : result.code === 100
                                ? this.config.loggerProvider.warn(result.message)
                                : this.config.loggerProvider.error(result.message);
                        return [2 /*return*/, result];
                    case 2:
                        e_2 = _a.sent();
                        message = String(e_2);
                        this.config.loggerProvider.error(message);
                        result = (0, result_builder_1.buildResult)(event, 0, message);
                        return [2 /*return*/, result];
                    case 3: return [2 /*return*/];
                }
            });
        });
    };
    AmplitudeCore.prototype.setOptOut = function (optOut) {
        if (!this.isReady) {
            this.q.push(this._setOptOut.bind(this, Boolean(optOut)));
            return;
        }
        this._setOptOut(optOut);
    };
    AmplitudeCore.prototype._setOptOut = function (optOut) {
        if (this.config.optOut !== optOut) {
            this.config.optOut = Boolean(optOut);
            this.timeline.onOptOutChanged(optOut);
        }
    };
    AmplitudeCore.prototype.flush = function () {
        return (0, return_wrapper_1.returnWrapper)(this.timeline.flush());
    };
    AmplitudeCore.prototype.plugin = function (name) {
        var plugin = this.timeline.plugins.find(function (plugin) { return plugin.name === name; });
        if (plugin === undefined) {
            this.config.loggerProvider.debug("Cannot find plugin with name ".concat(name));
            return undefined;
        }
        return plugin;
    };
    AmplitudeCore.prototype.plugins = function (pluginClass) {
        return this.timeline.plugins.filter(function (plugin) { return plugin instanceof pluginClass; });
    };
    return AmplitudeCore;
}());
exports.AmplitudeCore = AmplitudeCore;
//# sourceMappingURL=core-client.js.map