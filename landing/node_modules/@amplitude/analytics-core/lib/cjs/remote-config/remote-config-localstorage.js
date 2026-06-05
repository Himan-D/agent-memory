"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.RemoteConfigLocalStorage = void 0;
var RemoteConfigLocalStorage = /** @class */ (function () {
    function RemoteConfigLocalStorage(apiKey, logger) {
        this.key = "AMP_remote_config_".concat(apiKey.substring(0, 10));
        this.logger = logger;
    }
    RemoteConfigLocalStorage.prototype.fetchConfig = function () {
        var result = null;
        var failedRemoteConfigInfo = {
            remoteConfig: null,
            lastFetch: new Date(),
        };
        try {
            result = localStorage.getItem(this.key);
        }
        catch (error) {
            this.logger.debug('Remote config localstorage failed to access: ', error);
            return Promise.resolve(failedRemoteConfigInfo);
        }
        if (result === null) {
            this.logger.debug('Remote config localstorage gets null because the key does not exist');
            return Promise.resolve(failedRemoteConfigInfo);
        }
        try {
            var remoteConfigInfo = JSON.parse(result);
            this.logger.debug("Remote config localstorage parsed successfully: ".concat(JSON.stringify(remoteConfigInfo)));
            return Promise.resolve({
                remoteConfig: remoteConfigInfo.remoteConfig,
                lastFetch: new Date(remoteConfigInfo.lastFetch),
            });
        }
        catch (error) {
            this.logger.debug('Remote config localstorage failed to parse: ', error);
            localStorage.removeItem(this.key);
            return Promise.resolve(failedRemoteConfigInfo);
        }
    };
    RemoteConfigLocalStorage.prototype.setConfig = function (config) {
        try {
            localStorage.setItem(this.key, JSON.stringify(config));
            this.logger.debug('Remote config localstorage set successfully.');
            return Promise.resolve(true);
        }
        catch (error) {
            this.logger.debug('Remote config localstorage failed to set: ', error);
        }
        return Promise.resolve(false);
    };
    return RemoteConfigLocalStorage;
}());
exports.RemoteConfigLocalStorage = RemoteConfigLocalStorage;
//# sourceMappingURL=remote-config-localstorage.js.map