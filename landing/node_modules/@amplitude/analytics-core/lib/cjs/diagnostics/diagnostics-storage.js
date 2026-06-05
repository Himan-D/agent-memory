"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.DiagnosticsStorage = exports.INTERNAL_KEYS = exports.TABLE_NAMES = void 0;
var tslib_1 = require("tslib");
var global_scope_1 = require("../global-scope");
var MAX_PERSISTENT_STORAGE_EVENTS_COUNT = 10;
// Database configuration
var DB_VERSION = 1;
// Table names for different diagnostics types
exports.TABLE_NAMES = {
    TAGS: 'tags',
    COUNTERS: 'counters',
    HISTOGRAMS: 'histograms',
    EVENTS: 'events',
    INTERNAL: 'internal', // New table for internal storage like flush timestamps
};
// Keys for internal storage table
exports.INTERNAL_KEYS = {
    LAST_FLUSH_TIMESTAMP: 'last_flush_timestamp',
};
/**
 * Purpose-specific IndexedDB storage for diagnostics data
 * Provides optimized methods for each type of diagnostics data
 */
var DiagnosticsStorage = /** @class */ (function () {
    function DiagnosticsStorage(apiKey, logger) {
        this.dbPromise = null;
        this.logger = logger;
        this.dbName = "AMP_diagnostics_".concat(apiKey.substring(0, 10));
    }
    /**
     * Check if IndexedDB is supported in the current environment
     * @returns true if IndexedDB is available, false otherwise
     */
    DiagnosticsStorage.isSupported = function () {
        var _a;
        return ((_a = (0, global_scope_1.getGlobalScope)()) === null || _a === void 0 ? void 0 : _a.indexedDB) !== undefined;
    };
    DiagnosticsStorage.prototype.getDB = function () {
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            return tslib_1.__generator(this, function (_a) {
                if (!this.dbPromise) {
                    this.dbPromise = this.openDB();
                }
                return [2 /*return*/, this.dbPromise];
            });
        });
    };
    DiagnosticsStorage.prototype.openDB = function () {
        var _this = this;
        return new Promise(function (resolve, reject) {
            var request = indexedDB.open(_this.dbName, DB_VERSION);
            request.onerror = function () {
                // Clear dbPromise when it rejects for the first time
                _this.dbPromise = null;
                reject(new Error('Failed to open IndexedDB'));
            };
            request.onsuccess = function () {
                var db = request.result;
                // Clear dbPromise when connection was on but went off later
                db.onclose = function () {
                    _this.dbPromise = null;
                    _this.logger.debug('DiagnosticsStorage: DB connection closed.');
                };
                db.onerror = function (event) {
                    _this.logger.debug('DiagnosticsStorage: A global database error occurred.', event);
                    db.close();
                };
                resolve(db);
            };
            request.onupgradeneeded = function (event) {
                var db = event.target.result;
                _this.createTables(db);
            };
        });
    };
    DiagnosticsStorage.prototype.createTables = function (db) {
        // Create tags table
        if (!db.objectStoreNames.contains(exports.TABLE_NAMES.TAGS)) {
            db.createObjectStore(exports.TABLE_NAMES.TAGS, { keyPath: 'key' });
        }
        // Create counters table
        if (!db.objectStoreNames.contains(exports.TABLE_NAMES.COUNTERS)) {
            db.createObjectStore(exports.TABLE_NAMES.COUNTERS, { keyPath: 'key' });
        }
        // Create histograms table for storing histogram stats (count, min, max, sum)
        if (!db.objectStoreNames.contains(exports.TABLE_NAMES.HISTOGRAMS)) {
            db.createObjectStore(exports.TABLE_NAMES.HISTOGRAMS, {
                keyPath: 'key',
            });
        }
        // Create events table
        if (!db.objectStoreNames.contains(exports.TABLE_NAMES.EVENTS)) {
            var eventsStore = db.createObjectStore(exports.TABLE_NAMES.EVENTS, {
                keyPath: 'id',
                autoIncrement: true,
            });
            // Create index on time for chronological queries
            eventsStore.createIndex('time_idx', 'time', { unique: false });
        }
        // Create internal table for storing internal data like flush timestamps
        if (!db.objectStoreNames.contains(exports.TABLE_NAMES.INTERNAL)) {
            db.createObjectStore(exports.TABLE_NAMES.INTERNAL, { keyPath: 'key' });
        }
    };
    DiagnosticsStorage.prototype.setTags = function (tags) {
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            var db, transaction_1, store_1, error_1;
            var _this = this;
            return tslib_1.__generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        _a.trys.push([0, 2, , 3]);
                        if (Object.entries(tags).length === 0) {
                            return [2 /*return*/];
                        }
                        return [4 /*yield*/, this.getDB()];
                    case 1:
                        db = _a.sent();
                        transaction_1 = db.transaction([exports.TABLE_NAMES.TAGS], 'readwrite');
                        store_1 = transaction_1.objectStore(exports.TABLE_NAMES.TAGS);
                        return [2 /*return*/, new Promise(function (resolve) {
                                var entries = Object.entries(tags);
                                transaction_1.oncomplete = function () {
                                    resolve();
                                };
                                transaction_1.onabort = function (event) {
                                    _this.logger.debug('DiagnosticsStorage: Failed to set tags', event);
                                    resolve();
                                };
                                entries.forEach(function (_a) {
                                    var _b = tslib_1.__read(_a, 2), key = _b[0], value = _b[1];
                                    var putRequest = store_1.put({ key: key, value: value });
                                    putRequest.onerror = function (event) {
                                        _this.logger.debug('DiagnosticsStorage: Failed to set tag', key, value, event);
                                    };
                                });
                            })];
                    case 2:
                        error_1 = _a.sent();
                        this.logger.debug('DiagnosticsStorage: Failed to set tags', error_1);
                        return [3 /*break*/, 3];
                    case 3: return [2 /*return*/];
                }
            });
        });
    };
    DiagnosticsStorage.prototype.incrementCounters = function (counters) {
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            var db, transaction_2, store_2, error_2;
            var _this = this;
            return tslib_1.__generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        _a.trys.push([0, 2, , 3]);
                        if (Object.entries(counters).length === 0) {
                            return [2 /*return*/];
                        }
                        return [4 /*yield*/, this.getDB()];
                    case 1:
                        db = _a.sent();
                        transaction_2 = db.transaction([exports.TABLE_NAMES.COUNTERS], 'readwrite');
                        store_2 = transaction_2.objectStore(exports.TABLE_NAMES.COUNTERS);
                        return [2 /*return*/, new Promise(function (resolve) {
                                var entries = Object.entries(counters);
                                transaction_2.oncomplete = function () {
                                    resolve();
                                };
                                transaction_2.onabort = function (event) {
                                    _this.logger.debug('DiagnosticsStorage: Failed to increment counters', event);
                                    resolve();
                                };
                                // Read existing values and update them
                                entries.forEach(function (_a) {
                                    var _b = tslib_1.__read(_a, 2), key = _b[0], incrementValue = _b[1];
                                    var getRequest = store_2.get(key);
                                    getRequest.onsuccess = function () {
                                        var existingRecord = getRequest.result;
                                        /* istanbul ignore next */
                                        var existingValue = existingRecord ? existingRecord.value : 0;
                                        var putRequest = store_2.put({ key: key, value: existingValue + incrementValue });
                                        putRequest.onerror = function (event) {
                                            _this.logger.debug('DiagnosticsStorage: Failed to update counter', key, event);
                                        };
                                    };
                                    getRequest.onerror = function (event) {
                                        _this.logger.debug('DiagnosticsStorage: Failed to read existing counter', key, event);
                                    };
                                });
                            })];
                    case 2:
                        error_2 = _a.sent();
                        this.logger.debug('DiagnosticsStorage: Failed to increment counters', error_2);
                        return [3 /*break*/, 3];
                    case 3: return [2 /*return*/];
                }
            });
        });
    };
    DiagnosticsStorage.prototype.setHistogramStats = function (histogramStats) {
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            var db, transaction_3, store_3, error_3;
            var _this = this;
            return tslib_1.__generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        _a.trys.push([0, 2, , 3]);
                        if (Object.entries(histogramStats).length === 0) {
                            return [2 /*return*/];
                        }
                        return [4 /*yield*/, this.getDB()];
                    case 1:
                        db = _a.sent();
                        transaction_3 = db.transaction([exports.TABLE_NAMES.HISTOGRAMS], 'readwrite');
                        store_3 = transaction_3.objectStore(exports.TABLE_NAMES.HISTOGRAMS);
                        return [2 /*return*/, new Promise(function (resolve) {
                                var entries = Object.entries(histogramStats);
                                transaction_3.oncomplete = function () {
                                    resolve();
                                };
                                transaction_3.onabort = function (event) {
                                    _this.logger.debug('DiagnosticsStorage: Failed to set histogram stats', event);
                                    resolve();
                                };
                                // Read existing values and update them
                                entries.forEach(function (_a) {
                                    var _b = tslib_1.__read(_a, 2), key = _b[0], newStats = _b[1];
                                    var getRequest = store_3.get(key);
                                    getRequest.onsuccess = function () {
                                        var existingRecord = getRequest.result;
                                        var updatedStats;
                                        /* istanbul ignore next */
                                        if (existingRecord) {
                                            // Accumulate with existing stats
                                            updatedStats = {
                                                key: key,
                                                count: existingRecord.count + newStats.count,
                                                min: Math.min(existingRecord.min, newStats.min),
                                                max: Math.max(existingRecord.max, newStats.max),
                                                sum: existingRecord.sum + newStats.sum,
                                            };
                                        }
                                        else {
                                            // Create new stats
                                            updatedStats = {
                                                key: key,
                                                count: newStats.count,
                                                min: newStats.min,
                                                max: newStats.max,
                                                sum: newStats.sum,
                                            };
                                        }
                                        var putRequest = store_3.put(updatedStats);
                                        putRequest.onerror = function (event) {
                                            _this.logger.debug('DiagnosticsStorage: Failed to set histogram stats', key, event);
                                        };
                                    };
                                    getRequest.onerror = function (event) {
                                        _this.logger.debug('DiagnosticsStorage: Failed to read existing histogram stats', key, event);
                                    };
                                });
                            })];
                    case 2:
                        error_3 = _a.sent();
                        this.logger.debug('DiagnosticsStorage: Failed to set histogram stats', error_3);
                        return [3 /*break*/, 3];
                    case 3: return [2 /*return*/];
                }
            });
        });
    };
    DiagnosticsStorage.prototype.addEventRecords = function (events) {
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            var db, transaction_4, store_4, error_4;
            var _this = this;
            return tslib_1.__generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        _a.trys.push([0, 2, , 3]);
                        if (events.length === 0) {
                            return [2 /*return*/];
                        }
                        return [4 /*yield*/, this.getDB()];
                    case 1:
                        db = _a.sent();
                        transaction_4 = db.transaction([exports.TABLE_NAMES.EVENTS], 'readwrite');
                        store_4 = transaction_4.objectStore(exports.TABLE_NAMES.EVENTS);
                        return [2 /*return*/, new Promise(function (resolve) {
                                transaction_4.oncomplete = function () {
                                    resolve();
                                };
                                /* istanbul ignore next */
                                transaction_4.onabort = function (event) {
                                    _this.logger.debug('DiagnosticsStorage: Failed to add event records', event);
                                    resolve();
                                };
                                // First, check how many events are currently stored
                                var countRequest = store_4.count();
                                countRequest.onsuccess = function () {
                                    var currentCount = countRequest.result;
                                    // Calculate how many events we can add
                                    var availableSlots = Math.max(0, MAX_PERSISTENT_STORAGE_EVENTS_COUNT - currentCount);
                                    if (availableSlots < events.length) {
                                        _this.logger.debug("DiagnosticsStorage: Only added ".concat(availableSlots, " of ").concat(events.length, " events due to storage limit"));
                                    }
                                    // Only add events up to the available slots (take the least recent ones)
                                    events.slice(0, availableSlots).forEach(function (event) {
                                        var request = store_4.add(event);
                                        request.onerror = function (event) {
                                            _this.logger.debug('DiagnosticsStorage: Failed to add event record', event);
                                        };
                                    });
                                };
                                countRequest.onerror = function (event) {
                                    _this.logger.debug('DiagnosticsStorage: Failed to count existing events', event);
                                };
                            })];
                    case 2:
                        error_4 = _a.sent();
                        this.logger.debug('DiagnosticsStorage: Failed to add event records', error_4);
                        return [3 /*break*/, 3];
                    case 3: return [2 /*return*/];
                }
            });
        });
    };
    DiagnosticsStorage.prototype.setInternal = function (key, value) {
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            var db, transaction_5, store_5, error_5;
            return tslib_1.__generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        _a.trys.push([0, 2, , 3]);
                        return [4 /*yield*/, this.getDB()];
                    case 1:
                        db = _a.sent();
                        transaction_5 = db.transaction([exports.TABLE_NAMES.INTERNAL], 'readwrite');
                        store_5 = transaction_5.objectStore(exports.TABLE_NAMES.INTERNAL);
                        return [2 /*return*/, new Promise(function (resolve, reject) {
                                /* istanbul ignore next */
                                transaction_5.onabort = function () { return reject(new Error('Failed to set internal value')); };
                                var request = store_5.put({ key: key, value: value });
                                request.onsuccess = function () { return resolve(); };
                                /* istanbul ignore next */
                                request.onerror = function () { return reject(new Error('Failed to set internal value')); };
                            })];
                    case 2:
                        error_5 = _a.sent();
                        /* istanbul ignore next */
                        this.logger.debug('DiagnosticsStorage: Failed to set internal value', error_5);
                        return [3 /*break*/, 3];
                    case 3: return [2 /*return*/];
                }
            });
        });
    };
    DiagnosticsStorage.prototype.getInternal = function (key) {
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            var db, transaction_6, store_6, error_6;
            return tslib_1.__generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        _a.trys.push([0, 2, , 3]);
                        return [4 /*yield*/, this.getDB()];
                    case 1:
                        db = _a.sent();
                        transaction_6 = db.transaction([exports.TABLE_NAMES.INTERNAL], 'readonly');
                        store_6 = transaction_6.objectStore(exports.TABLE_NAMES.INTERNAL);
                        return [2 /*return*/, new Promise(function (resolve, reject) {
                                /* istanbul ignore next */
                                transaction_6.onabort = function () { return reject(new Error('Failed to get internal value')); };
                                var request = store_6.get(key);
                                request.onsuccess = function () { return resolve(request.result); };
                                /* istanbul ignore next */
                                request.onerror = function () { return reject(new Error('Failed to get internal value')); };
                            })];
                    case 2:
                        error_6 = _a.sent();
                        this.logger.debug('DiagnosticsStorage: Failed to get internal value', error_6);
                        return [2 /*return*/, undefined];
                    case 3: return [2 /*return*/];
                }
            });
        });
    };
    DiagnosticsStorage.prototype.getLastFlushTimestamp = function () {
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            var record, error_7;
            return tslib_1.__generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        _a.trys.push([0, 2, , 3]);
                        return [4 /*yield*/, this.getInternal(exports.INTERNAL_KEYS.LAST_FLUSH_TIMESTAMP)];
                    case 1:
                        record = _a.sent();
                        return [2 /*return*/, record ? parseInt(record.value, 10) : undefined];
                    case 2:
                        error_7 = _a.sent();
                        /* istanbul ignore next */
                        this.logger.debug('DiagnosticsStorage: Failed to get last flush timestamp', error_7);
                        /* istanbul ignore next */
                        return [2 /*return*/, undefined];
                    case 3: return [2 /*return*/];
                }
            });
        });
    };
    DiagnosticsStorage.prototype.setLastFlushTimestamp = function (timestamp) {
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            var error_8;
            return tslib_1.__generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        _a.trys.push([0, 2, , 3]);
                        return [4 /*yield*/, this.setInternal(exports.INTERNAL_KEYS.LAST_FLUSH_TIMESTAMP, timestamp.toString())];
                    case 1:
                        _a.sent();
                        return [3 /*break*/, 3];
                    case 2:
                        error_8 = _a.sent();
                        /* istanbul ignore next */
                        this.logger.debug('DiagnosticsStorage: Failed to set last flush timestamp', error_8);
                        return [3 /*break*/, 3];
                    case 3: return [2 /*return*/];
                }
            });
        });
    };
    /* istanbul ignore next */
    DiagnosticsStorage.prototype.clearTable = function (transaction, tableName) {
        return new Promise(function (resolve, reject) {
            var store = transaction.objectStore(tableName);
            var request = store.clear();
            request.onsuccess = function () { return resolve(); };
            request.onerror = function () { return reject(new Error("Failed to clear table ".concat(tableName))); };
        });
    };
    /* istanbul ignore next */
    DiagnosticsStorage.prototype.getAllAndClear = function () {
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            var db, transaction, _a, tags, counters, histogramStats, events, error_9;
            return tslib_1.__generator(this, function (_b) {
                switch (_b.label) {
                    case 0:
                        _b.trys.push([0, 4, , 5]);
                        return [4 /*yield*/, this.getDB()];
                    case 1:
                        db = _b.sent();
                        transaction = db.transaction([exports.TABLE_NAMES.TAGS, exports.TABLE_NAMES.COUNTERS, exports.TABLE_NAMES.HISTOGRAMS, exports.TABLE_NAMES.EVENTS], 'readwrite');
                        return [4 /*yield*/, Promise.all([
                                this.getAllFromStore(transaction, exports.TABLE_NAMES.TAGS),
                                this.getAllFromStore(transaction, exports.TABLE_NAMES.COUNTERS),
                                this.getAllFromStore(transaction, exports.TABLE_NAMES.HISTOGRAMS),
                                this.getAllFromStore(transaction, exports.TABLE_NAMES.EVENTS),
                            ])];
                    case 2:
                        _a = tslib_1.__read.apply(void 0, [_b.sent(), 4]), tags = _a[0], counters = _a[1], histogramStats = _a[2], events = _a[3];
                        // Clear all data in the same transaction
                        return [4 /*yield*/, Promise.all([
                                this.clearTable(transaction, exports.TABLE_NAMES.COUNTERS),
                                this.clearTable(transaction, exports.TABLE_NAMES.HISTOGRAMS),
                                this.clearTable(transaction, exports.TABLE_NAMES.EVENTS),
                            ])];
                    case 3:
                        // Clear all data in the same transaction
                        _b.sent();
                        return [2 /*return*/, { tags: tags, counters: counters, histogramStats: histogramStats, events: events }];
                    case 4:
                        error_9 = _b.sent();
                        this.logger.debug('DiagnosticsStorage: Failed to get all and clear data', error_9);
                        return [2 /*return*/, { tags: [], counters: [], histogramStats: [], events: [] }];
                    case 5: return [2 /*return*/];
                }
            });
        });
    };
    /**
     * Helper method to get all records from a store within a transaction
     */
    /* istanbul ignore next */
    DiagnosticsStorage.prototype.getAllFromStore = function (transaction, tableName) {
        return new Promise(function (resolve, reject) {
            var store = transaction.objectStore(tableName);
            var request = store.getAll();
            request.onsuccess = function () { return resolve(request.result); };
            request.onerror = function () { return reject(new Error("Failed to get all from ".concat(tableName))); };
        });
    };
    return DiagnosticsStorage;
}());
exports.DiagnosticsStorage = DiagnosticsStorage;
//# sourceMappingURL=diagnostics-storage.js.map