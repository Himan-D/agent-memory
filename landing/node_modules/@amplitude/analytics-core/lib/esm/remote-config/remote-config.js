import { __awaiter, __generator } from "tslib";
import { RemoteConfigLocalStorage } from './remote-config-localstorage';
import { UUID } from '../utils/uuid';
export var US_SERVER_URL = 'https://sr-client-cfg.amplitude.com/config';
export var EU_SERVER_URL = 'https://sr-client-cfg.eu.amplitude.com/config';
export var DEFAULT_MAX_RETRIES = 3;
var CODE_STATUS = {
    INVALID_API_KEY: 401,
    FORBIDDEN: 403,
    RATE_LIMIT: 429,
};
/**
 * The default timeout for fetch in milliseconds.
 * Linear backoff policy: timeout / retry times is the interval between fetch retry.
 */
var DEFAULT_TIMEOUT = 1000;
/**
 * The minimum time between fetches in milliseconds.
 * This prevents too many requests from being sent in a short period of time.
 */
var DEFAULT_MIN_TIME_BETWEEN_FETCHES = 5 * 60 * 1000; // 5 minutes
var RemoteConfigClient = /** @class */ (function () {
    function RemoteConfigClient(apiKey, logger, serverZone, serverUrl) {
        if (serverZone === void 0) { serverZone = 'US'; }
        // Registered callbackInfos by subscribe().
        this.callbackInfos = [];
        // Track the last successful fetch time for throttling (timestamp in milliseconds).
        this.lastSuccessfulFetch = null;
        // Store the in-flight fetch promise for deduplication.
        this.fetchPromise = null;
        // Used to skip periodic updateConfigs calls when API key is invalid.
        this.isLastFetchInvalidApiKey = false;
        this.apiKey = apiKey;
        this.serverUrl = serverUrl || (serverZone === 'US' ? US_SERVER_URL : EU_SERVER_URL);
        this.logger = logger;
        this.storage = new RemoteConfigLocalStorage(apiKey, logger);
    }
    RemoteConfigClient.prototype.subscribe = function (key, deliveryMode, callback) {
        var id = UUID();
        var callbackInfo = {
            id: id,
            key: key,
            deliveryMode: deliveryMode,
            callback: callback,
        };
        this.callbackInfos.push(callbackInfo);
        if (deliveryMode === 'all') {
            void this.subscribeAll(callbackInfo);
        }
        else {
            void this.subscribeWaitForRemote(callbackInfo, deliveryMode.timeout);
        }
        return id;
    };
    RemoteConfigClient.prototype.unsubscribe = function (id) {
        var index = this.callbackInfos.findIndex(function (callbackInfo) { return callbackInfo.id === id; });
        if (index === -1) {
            this.logger.debug("Remote config client unsubscribe failed because callback with id ".concat(id, " doesn't exist."));
            return false;
        }
        this.callbackInfos.splice(index, 1);
        this.logger.debug("Remote config client unsubscribe succeeded removing callback with id ".concat(id, "."));
        return true;
    };
    RemoteConfigClient.prototype.updateConfigs = function () {
        return __awaiter(this, void 0, void 0, function () {
            var timeSinceLastFetch, result;
            var _this = this;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        // Check if we need to throttle based on last successful fetch time
                        if (this.lastSuccessfulFetch) {
                            timeSinceLastFetch = Date.now() - this.lastSuccessfulFetch;
                            if (timeSinceLastFetch < DEFAULT_MIN_TIME_BETWEEN_FETCHES) {
                                this.logger.debug('Remote config client skipping updateConfigs: Too recent');
                                return [2 /*return*/];
                            }
                        }
                        return [4 /*yield*/, this.getOrCreateFetchPromise()];
                    case 1:
                        result = _a.sent();
                        void this.storage.setConfig(result);
                        this.callbackInfos.forEach(function (callbackInfo) {
                            _this.sendCallback(callbackInfo, result, 'remote');
                        });
                        return [2 /*return*/];
                }
            });
        });
    };
    /**
     * Get the in-flight fetch promise or create a new one.
     * This ensures multiple subscribe calls share the same network request.
     */
    RemoteConfigClient.prototype.getOrCreateFetchPromise = function () {
        var _this = this;
        if (this.fetchPromise) {
            return this.fetchPromise;
        }
        if (this.isLastFetchInvalidApiKey) {
            this.logger.debug('Remote config client skipping fetch: Invalid API key');
            this.fetchPromise = Promise.resolve({
                remoteConfig: null,
                lastFetch: new Date(),
            }).finally(function () {
                _this.fetchPromise = null;
            });
            return this.fetchPromise;
        }
        this.fetchPromise = this.fetch()
            .then(function (result) {
            // Update last successful fetch time if we got a valid config
            if (result.remoteConfig !== null) {
                _this.lastSuccessfulFetch = Date.now();
            }
            return result;
        })
            .finally(function () {
            // Clear the promise after it settles (success or failure)
            _this.fetchPromise = null;
        });
        return this.fetchPromise;
    };
    /**
     * Send remote first. If it's already complete, we can skip the cached response.
     * - if remote is fetched first, no cache fetch.
     * - if cache is fetched first, still fetching remote.
     */
    RemoteConfigClient.prototype.subscribeAll = function (callbackInfo) {
        return __awaiter(this, void 0, void 0, function () {
            var remotePromise, cachePromise, result;
            var _this = this;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        remotePromise = this.getOrCreateFetchPromise().then(function (result) {
                            _this.logger.debug("Remote config client subscription all mode fetched from remote: ".concat(JSON.stringify(result)));
                            _this.sendCallback(callbackInfo, result, 'remote');
                            void _this.storage.setConfig(result);
                        });
                        cachePromise = this.storage.fetchConfig().then(function (result) {
                            return result;
                        });
                        return [4 /*yield*/, Promise.race([remotePromise, cachePromise])];
                    case 1:
                        result = _a.sent();
                        // If cache is fetched first, wait for remote.
                        if (result !== undefined) {
                            this.logger.debug("Remote config client subscription all mode fetched from cache: ".concat(JSON.stringify(result)));
                            // Skip sending callback if cache is empty (first time user).
                            if (result.remoteConfig !== null) {
                                this.sendCallback(callbackInfo, result, 'cache');
                            }
                            else {
                                this.logger.debug('Remote config client skips sending callback because cache is empty (first time user).');
                            }
                        }
                        return [4 /*yield*/, remotePromise];
                    case 2:
                        _a.sent();
                        return [2 /*return*/];
                }
            });
        });
    };
    /**
     * Waits for a remote response until the given timeout, then return a cached copy, if available.
     */
    RemoteConfigClient.prototype.subscribeWaitForRemote = function (callbackInfo, timeout) {
        return __awaiter(this, void 0, void 0, function () {
            var timeoutPromise, result, error_1, result;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        timeoutPromise = new Promise(function (_, reject) {
                            setTimeout(function () {
                                reject('Timeout exceeded');
                            }, timeout);
                        });
                        _a.label = 1;
                    case 1:
                        _a.trys.push([1, 3, , 5]);
                        return [4 /*yield*/, Promise.race([
                                this.getOrCreateFetchPromise(),
                                timeoutPromise,
                            ])];
                    case 2:
                        result = (_a.sent());
                        this.logger.debug('Remote config client subscription wait for remote mode returns from remote.');
                        this.sendCallback(callbackInfo, result, 'remote');
                        void this.storage.setConfig(result);
                        return [3 /*break*/, 5];
                    case 3:
                        error_1 = _a.sent();
                        this.logger.debug('Remote config client subscription wait for remote mode exceeded timeout. Try to fetch from cache.');
                        return [4 /*yield*/, this.storage.fetchConfig()];
                    case 4:
                        result = _a.sent();
                        if (result.remoteConfig !== null) {
                            this.logger.debug('Remote config client subscription wait for remote mode returns a cached copy.');
                            this.sendCallback(callbackInfo, result, 'cache');
                        }
                        else {
                            this.logger.debug('Remote config client subscription wait for remote mode failed to fetch cache.');
                            this.sendCallback(callbackInfo, result, 'remote');
                        }
                        return [3 /*break*/, 5];
                    case 5: return [2 /*return*/];
                }
            });
        });
    };
    /**
     * Call the callback with filtered remote config based on key.
     * @param remoteConfigInfo - the whole remote config object without filtering by key.
     */
    RemoteConfigClient.prototype.sendCallback = function (callbackInfo, remoteConfigInfo, source) {
        callbackInfo.lastCallback = new Date();
        var filteredConfig;
        if (callbackInfo.key) {
            // Filter remote config by key.
            // For example, if remote config is {a: {b: {c: 1}}},
            // if key = 'a', filter result is {b: {c: 1}};
            // if key = 'a.b', filter result is {c: 1}
            filteredConfig = callbackInfo.key.split('.').reduce(function (config, key) {
                if (config === null) {
                    return config;
                }
                return key in config ? config[key] : null;
            }, remoteConfigInfo.remoteConfig);
        }
        else {
            filteredConfig = remoteConfigInfo.remoteConfig;
        }
        callbackInfo.callback(filteredConfig, source, remoteConfigInfo.lastFetch);
    };
    /**
     * Fetch remote config from remote.
     * @param retries - the number of retries. default is 3.
     * @param timeout - the timeout in milliseconds. Default is 1000.
     * This timeout serves two purposes:
     * 1. It determines how long to wait for each remote config fetch request before aborting it.
     *    If the fetch does not complete within the specified timeout, the request is cancelled using AbortController,
     *    and the attempt is considered failed (and may be retried if retries remain).
     * 2. It is also used to calculate the interval between retries. The total timeout is divided by the number of retries,
     *    so each retry waits for (timeout / retries) milliseconds before the next attempt (linear backoff).
     * Retry behavior by status code:
     * - 401: invalid API key (stop retries and disable future updateConfigs calls).
     * - 429: retry up to max retries.
     * - other 4xx: no retry.
     * - 5xx and network failures: retry up to max retries.
     * @returns the remote config info. null if failed to fetch or the response is not valid JSON.
     */
    RemoteConfigClient.prototype.fetch = function (retries, timeout) {
        if (retries === void 0) { retries = DEFAULT_MAX_RETRIES; }
        if (timeout === void 0) { timeout = DEFAULT_TIMEOUT; }
        return __awaiter(this, void 0, void 0, function () {
            var interval, failedRemoteConfigInfo, _loop_1, this_1, attempt, state_1;
            var _this = this;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        interval = timeout / retries;
                        failedRemoteConfigInfo = {
                            remoteConfig: null,
                            lastFetch: new Date(),
                        };
                        _loop_1 = function (attempt) {
                            var shouldRetry, abortController, timeoutId, res, body, remoteConfig, error_2;
                            return __generator(this, function (_b) {
                                switch (_b.label) {
                                    case 0:
                                        shouldRetry = true;
                                        abortController = new AbortController();
                                        timeoutId = setTimeout(function () { return abortController.abort(); }, timeout);
                                        _b.label = 1;
                                    case 1:
                                        _b.trys.push([1, 7, 8, 9]);
                                        return [4 /*yield*/, fetch(this_1.getUrlParams(), {
                                                method: 'GET',
                                                headers: {
                                                    Accept: '*/*',
                                                },
                                                signal: abortController.signal,
                                            })];
                                    case 2:
                                        res = _b.sent();
                                        if (!!res.ok) return [3 /*break*/, 4];
                                        return [4 /*yield*/, res.text()];
                                    case 3:
                                        body = _b.sent();
                                        this_1.logger.debug("Remote config client fetch with retry time ".concat(retries, " failed with ").concat(res.status, ": ").concat(body));
                                        if (res.status === CODE_STATUS.INVALID_API_KEY || res.status === CODE_STATUS.FORBIDDEN) {
                                            this_1.logger.error("Remote config client fetch failed with ".concat(res.status, ". Invalid API key; future fetches will be skipped."));
                                            this_1.isLastFetchInvalidApiKey = true;
                                            shouldRetry = false;
                                        }
                                        else if (res.status >= 400 && res.status < 500 && res.status !== CODE_STATUS.RATE_LIMIT) {
                                            shouldRetry = false;
                                        }
                                        return [3 /*break*/, 6];
                                    case 4: return [4 /*yield*/, res.json()];
                                    case 5:
                                        remoteConfig = (_b.sent());
                                        return [2 /*return*/, { value: {
                                                    remoteConfig: remoteConfig,
                                                    lastFetch: new Date(),
                                                } }];
                                    case 6: return [3 /*break*/, 9];
                                    case 7:
                                        error_2 = _b.sent();
                                        // Handle rejects when the request fails, for example, a network error or timeout
                                        if (error_2 instanceof Error && error_2.name === 'AbortError') {
                                            this_1.logger.debug("Remote config client fetch with retry time ".concat(retries, " timed out after ").concat(timeout, "ms"));
                                        }
                                        else {
                                            this_1.logger.debug("Remote config client fetch with retry time ".concat(retries, " is rejected because: "), error_2);
                                        }
                                        return [3 /*break*/, 9];
                                    case 8:
                                        // Clear the timeout since request completed or failed
                                        clearTimeout(timeoutId);
                                        return [7 /*endfinally*/];
                                    case 9:
                                        if (!shouldRetry) {
                                            return [2 /*return*/, "break"];
                                        }
                                        if (!(attempt < retries - 1)) return [3 /*break*/, 11];
                                        return [4 /*yield*/, new Promise(function (resolve) { return setTimeout(resolve, _this.getJitterDelay(interval)); })];
                                    case 10:
                                        _b.sent();
                                        _b.label = 11;
                                    case 11: return [2 /*return*/];
                                }
                            });
                        };
                        this_1 = this;
                        attempt = 0;
                        _a.label = 1;
                    case 1:
                        if (!(attempt < retries)) return [3 /*break*/, 4];
                        return [5 /*yield**/, _loop_1(attempt)];
                    case 2:
                        state_1 = _a.sent();
                        if (typeof state_1 === "object")
                            return [2 /*return*/, state_1.value];
                        if (state_1 === "break")
                            return [3 /*break*/, 4];
                        _a.label = 3;
                    case 3:
                        attempt++;
                        return [3 /*break*/, 1];
                    case 4: return [2 /*return*/, failedRemoteConfigInfo];
                }
            });
        });
    };
    /**
     * Return jitter in the bound of [0,baseDelay) and then floor round.
     */
    RemoteConfigClient.prototype.getJitterDelay = function (baseDelay) {
        return Math.floor(Math.random() * baseDelay);
    };
    RemoteConfigClient.prototype.getUrlParams = function () {
        // URL encode the API key to handle special characters
        var encodedApiKey = encodeURIComponent(this.apiKey);
        var urlParams = new URLSearchParams();
        urlParams.append('config_group', RemoteConfigClient.CONFIG_GROUP);
        return "".concat(this.serverUrl, "/").concat(encodedApiKey, "?").concat(urlParams.toString());
    };
    RemoteConfigClient.CONFIG_GROUP = 'browser';
    return RemoteConfigClient;
}());
export { RemoteConfigClient };
//# sourceMappingURL=remote-config.js.map