"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.DiagnosticsClient = exports.MAX_MEMORY_STORAGE_EVENTS_COUNT = exports.MAX_MEMORY_STORAGE_COUNT = exports.DIAGNOSTICS_EU_SERVER_URL = exports.DIAGNOSTICS_US_SERVER_URL = exports.FLUSH_INTERVAL_MS = exports.SAVE_INTERVAL_MS = void 0;
var tslib_1 = require("tslib");
var diagnostics_storage_1 = require("./diagnostics-storage");
var global_scope_1 = require("../global-scope");
var sampling_1 = require("../utils/sampling");
var uncaught_sdk_errors_1 = require("./uncaught-sdk-errors");
exports.SAVE_INTERVAL_MS = 1000; // 1 second
exports.FLUSH_INTERVAL_MS = 5 * 60 * 1000; // 5 minutes
exports.DIAGNOSTICS_US_SERVER_URL = 'https://diagnostics.prod.us-west-2.amplitude.com/v1/capture';
exports.DIAGNOSTICS_EU_SERVER_URL = 'https://diagnostics.prod.eu-central-1.amplitude.com/v1/capture';
// In-memory storage limits
exports.MAX_MEMORY_STORAGE_COUNT = 10000; // for tags, counters, histograms separately
exports.MAX_MEMORY_STORAGE_EVENTS_COUNT = 10;
var DiagnosticsClient = /** @class */ (function () {
    function DiagnosticsClient(apiKey, logger, serverZone, options) {
        if (serverZone === void 0) { serverZone = 'US'; }
        // In-memory storages
        this.inMemoryTags = {};
        this.inMemoryCounters = {};
        this.inMemoryHistograms = {};
        this.inMemoryEvents = [];
        // Timer for 1-second persistence
        this.saveTimer = null;
        // Timer for flush interval
        this.flushTimer = null;
        this.apiKey = apiKey;
        this.logger = logger;
        this.serverUrl = serverZone === 'US' ? exports.DIAGNOSTICS_US_SERVER_URL : exports.DIAGNOSTICS_EU_SERVER_URL;
        this.logger.debug('DiagnosticsClient: Initializing with options', JSON.stringify(options, null, 2));
        // Diagnostics is enabled by default with sample rate of 0 (no sampling)
        this.config = tslib_1.__assign({ enabled: true, sampleRate: 0 }, options);
        this.startTimestamp = Date.now();
        this.shouldTrack = (0, sampling_1.isTimestampInSampleTemp)(this.startTimestamp, this.config.sampleRate) && this.config.enabled;
        if (diagnostics_storage_1.DiagnosticsStorage.isSupported()) {
            this.storage = new diagnostics_storage_1.DiagnosticsStorage(apiKey, logger);
        }
        else {
            this.logger.debug('DiagnosticsClient: IndexedDB is not supported');
        }
        void this.initializeFlushInterval();
        // Track internal diagnostics metrics for sampling
        if (this.shouldTrack) {
            this.increment('sdk.diagnostics.sampled.in.and.enabled');
            (0, uncaught_sdk_errors_1.enableSdkErrorListeners)(this);
        }
    }
    /**
     * Check if storage is available and tracking is enabled
     */
    DiagnosticsClient.prototype.isStorageAndTrackEnabled = function () {
        return Boolean(this.storage) && Boolean(this.shouldTrack);
    };
    DiagnosticsClient.prototype.setTag = function (name, value) {
        if (!this.isStorageAndTrackEnabled()) {
            return;
        }
        if (Object.keys(this.inMemoryTags).length >= exports.MAX_MEMORY_STORAGE_COUNT) {
            this.logger.debug('DiagnosticsClient: Early return setTags as reaching memory limit');
            return;
        }
        this.inMemoryTags[name] = value;
        this.startTimersIfNeeded();
    };
    DiagnosticsClient.prototype.increment = function (name, size) {
        if (size === void 0) { size = 1; }
        if (!this.isStorageAndTrackEnabled()) {
            return;
        }
        if (Object.keys(this.inMemoryCounters).length >= exports.MAX_MEMORY_STORAGE_COUNT) {
            this.logger.debug('DiagnosticsClient: Early return increment as reaching memory limit');
            return;
        }
        this.inMemoryCounters[name] = (this.inMemoryCounters[name] || 0) + size;
        this.startTimersIfNeeded();
    };
    DiagnosticsClient.prototype.recordHistogram = function (name, value) {
        if (!this.isStorageAndTrackEnabled()) {
            return;
        }
        if (Object.keys(this.inMemoryHistograms).length >= exports.MAX_MEMORY_STORAGE_COUNT) {
            this.logger.debug('DiagnosticsClient: Early return recordHistogram as reaching memory limit');
            return;
        }
        var existing = this.inMemoryHistograms[name];
        if (existing) {
            // Update existing stats incrementally
            existing.count += 1;
            existing.min = Math.min(existing.min, value);
            existing.max = Math.max(existing.max, value);
            existing.sum += value;
        }
        else {
            // Create new stats
            this.inMemoryHistograms[name] = {
                count: 1,
                min: value,
                max: value,
                sum: value,
            };
        }
        this.startTimersIfNeeded();
    };
    DiagnosticsClient.prototype.recordEvent = function (name, properties) {
        if (!this.isStorageAndTrackEnabled()) {
            return;
        }
        if (this.inMemoryEvents.length >= exports.MAX_MEMORY_STORAGE_EVENTS_COUNT) {
            this.logger.debug('DiagnosticsClient: Early return recordEvent as reaching memory limit');
            return;
        }
        this.inMemoryEvents.push({
            event_name: name,
            time: Date.now(),
            event_properties: properties,
        });
        this.startTimersIfNeeded();
    };
    DiagnosticsClient.prototype.startTimersIfNeeded = function () {
        var _this = this;
        if (!this.saveTimer) {
            this.saveTimer = setTimeout(function () {
                _this.saveAllDataToStorage()
                    .catch(function (error) {
                    _this.logger.debug('DiagnosticsClient: Failed to save all data to storage', error);
                })
                    .finally(function () {
                    _this.saveTimer = null;
                });
            }, exports.SAVE_INTERVAL_MS);
        }
        if (!this.flushTimer) {
            this.flushTimer = setTimeout(function () {
                _this._flush()
                    .catch(function (error) {
                    _this.logger.debug('DiagnosticsClient: Failed to flush', error);
                })
                    .finally(function () {
                    _this.flushTimer = null;
                });
            }, exports.FLUSH_INTERVAL_MS);
        }
    };
    DiagnosticsClient.prototype.saveAllDataToStorage = function () {
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            var tagsToSave, countersToSave, histogramsToSave, eventsToSave;
            return tslib_1.__generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        if (!this.storage) {
                            return [2 /*return*/];
                        }
                        tagsToSave = tslib_1.__assign({}, this.inMemoryTags);
                        countersToSave = tslib_1.__assign({}, this.inMemoryCounters);
                        histogramsToSave = tslib_1.__assign({}, this.inMemoryHistograms);
                        eventsToSave = tslib_1.__spreadArray([], tslib_1.__read(this.inMemoryEvents), false);
                        this.inMemoryEvents = [];
                        this.inMemoryTags = {};
                        this.inMemoryCounters = {};
                        this.inMemoryHistograms = {};
                        return [4 /*yield*/, Promise.all([
                                this.storage.setTags(tagsToSave),
                                this.storage.incrementCounters(countersToSave),
                                this.storage.setHistogramStats(histogramsToSave),
                                this.storage.addEventRecords(eventsToSave),
                            ])];
                    case 1:
                        _a.sent();
                        return [2 /*return*/];
                }
            });
        });
    };
    DiagnosticsClient.prototype._flush = function () {
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            var _a, tagRecords, counterRecords, histogramStatsRecords, eventRecords, tags, counters, histogram, events, payload;
            return tslib_1.__generator(this, function (_b) {
                switch (_b.label) {
                    case 0:
                        if (!this.storage) {
                            return [2 /*return*/];
                        }
                        return [4 /*yield*/, this.saveAllDataToStorage()];
                    case 1:
                        _b.sent();
                        this.saveTimer = null;
                        this.flushTimer = null;
                        return [4 /*yield*/, this.storage.getAllAndClear()];
                    case 2:
                        _a = _b.sent(), tagRecords = _a.tags, counterRecords = _a.counters, histogramStatsRecords = _a.histogramStats, eventRecords = _a.events;
                        // Update the last flush timestamp
                        void this.storage.setLastFlushTimestamp(Date.now());
                        tags = {};
                        tagRecords.forEach(function (record) {
                            tags[record.key] = record.value;
                        });
                        counters = {};
                        counterRecords.forEach(function (record) {
                            counters[record.key] = record.value;
                        });
                        histogram = {};
                        histogramStatsRecords.forEach(function (stats) {
                            histogram[stats.key] = {
                                count: stats.count,
                                min: stats.min,
                                max: stats.max,
                                avg: Math.round((stats.sum / stats.count) * 100) / 100, // round the average to 2 decimal places.
                            };
                        });
                        events = eventRecords.map(function (record) { return ({
                            event_name: record.event_name,
                            time: record.time,
                            event_properties: record.event_properties,
                        }); });
                        // Early return if all data collections are empty
                        if (Object.keys(counters).length === 0 && Object.keys(histogram).length === 0 && events.length === 0) {
                            return [2 /*return*/];
                        }
                        payload = {
                            tags: tags,
                            histogram: histogram,
                            counters: counters,
                            events: events,
                        };
                        // Send payload to diagnostics server
                        void this.fetch(payload);
                        return [2 /*return*/];
                }
            });
        });
    };
    /**
     * Send diagnostics data to the server
     */
    DiagnosticsClient.prototype.fetch = function (payload) {
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            var response, error_1;
            return tslib_1.__generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        _a.trys.push([0, 2, , 3]);
                        if (!(0, global_scope_1.getGlobalScope)()) {
                            throw new Error('DiagnosticsClient: Fetch is not supported');
                        }
                        return [4 /*yield*/, fetch(this.serverUrl, {
                                method: 'POST',
                                headers: {
                                    'X-ApiKey': this.apiKey,
                                    'Content-Type': 'application/json',
                                },
                                body: JSON.stringify(payload),
                            })];
                    case 1:
                        response = _a.sent();
                        if (!response.ok) {
                            this.logger.debug('DiagnosticsClient: Failed to send diagnostics data.');
                            return [2 /*return*/];
                        }
                        this.logger.debug('DiagnosticsClient: Successfully sent diagnostics data');
                        return [3 /*break*/, 3];
                    case 2:
                        error_1 = _a.sent();
                        this.logger.debug('DiagnosticsClient: Failed to send diagnostics data. ', error_1);
                        return [3 /*break*/, 3];
                    case 3: return [2 /*return*/];
                }
            });
        });
    };
    /**
     * Initialize flush interval logic.
     * Check if 5 minutes has passed since last flush, if so flush immediately.
     * Otherwise set a timer to flush when the interval is reached.
     */
    DiagnosticsClient.prototype.initializeFlushInterval = function () {
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            var now, lastFlushTimestamp, timeSinceLastFlush;
            return tslib_1.__generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        if (!this.storage) {
                            return [2 /*return*/];
                        }
                        now = Date.now();
                        return [4 /*yield*/, this.storage.getLastFlushTimestamp()];
                    case 1:
                        lastFlushTimestamp = (_a.sent()) || -1;
                        // If last flush timestamp is -1, it means this is a new client
                        // Save current timestamp as the initial "last flush timestamp"
                        // and schedule the flush timer
                        if (lastFlushTimestamp === -1) {
                            void this.storage.setLastFlushTimestamp(now);
                            this._setFlushTimer(exports.FLUSH_INTERVAL_MS);
                            return [2 /*return*/];
                        }
                        timeSinceLastFlush = now - lastFlushTimestamp;
                        if (timeSinceLastFlush >= exports.FLUSH_INTERVAL_MS) {
                            // More than 5 minutes has passed, flush immediately
                            void this._flush();
                            return [2 /*return*/];
                        }
                        else {
                            // Set timer for remaining time
                            this._setFlushTimer(exports.FLUSH_INTERVAL_MS - timeSinceLastFlush);
                        }
                        return [2 /*return*/];
                }
            });
        });
    };
    /**
     * Helper method to set flush timer with consistent error handling
     */
    DiagnosticsClient.prototype._setFlushTimer = function (delay) {
        var _this = this;
        this.flushTimer = setTimeout(function () {
            _this._flush()
                .catch(function (error) {
                _this.logger.debug('DiagnosticsClient: Failed to flush', error);
            })
                .finally(function () {
                _this.flushTimer = null;
            });
        }, delay);
    };
    DiagnosticsClient.prototype._setSampleRate = function (sampleRate) {
        this.logger.debug('DiagnosticsClient: Setting sample rate to', sampleRate);
        this.config.sampleRate = sampleRate;
        this.shouldTrack = (0, sampling_1.isTimestampInSampleTemp)(this.startTimestamp, this.config.sampleRate) && this.config.enabled;
        this.logger.debug('DiagnosticsClient: Should track is', this.shouldTrack);
    };
    return DiagnosticsClient;
}());
exports.DiagnosticsClient = DiagnosticsClient;
//# sourceMappingURL=diagnostics-client.js.map