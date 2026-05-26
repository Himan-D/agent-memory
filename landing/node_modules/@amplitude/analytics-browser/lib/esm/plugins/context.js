import { __assign, __awaiter, __generator } from "tslib";
import { UUID, getLanguage } from '@amplitude/analytics-core';
import { VERSION } from '../version';
import { LIBPREFIX } from '../lib-prefix';
var BROWSER_PLATFORM = 'Web';
var IP_ADDRESS = '$remote';
var Context = /** @class */ (function () {
    function Context() {
        this.name = '@amplitude/plugin-context-browser';
        this.type = 'before';
        this.library = "".concat(LIBPREFIX, "/").concat(VERSION);
        /* istanbul ignore else */
        if (typeof navigator !== 'undefined') {
            this.userAgent = navigator.userAgent;
        }
    }
    Context.prototype.setup = function (config) {
        this.config = config;
        return Promise.resolve(undefined);
    };
    Context.prototype.execute = function (context) {
        var _a, _b;
        return __awaiter(this, void 0, void 0, function () {
            var time, lastEventId, nextEventId, event;
            return __generator(this, function (_c) {
                time = new Date().getTime();
                lastEventId = (_a = this.config.lastEventId) !== null && _a !== void 0 ? _a : -1;
                nextEventId = (_b = context.event_id) !== null && _b !== void 0 ? _b : lastEventId + 1;
                this.config.lastEventId = nextEventId;
                if (!context.time) {
                    this.config.lastEventTime = time;
                }
                event = __assign(__assign(__assign(__assign(__assign(__assign(__assign(__assign({ user_id: this.config.userId, device_id: this.config.deviceId, session_id: this.config.sessionId, time: time }, (this.config.appVersion && { app_version: this.config.appVersion })), (this.config.trackingOptions.platform && { platform: BROWSER_PLATFORM })), (this.config.trackingOptions.language && { language: getLanguage() })), (this.config.trackingOptions.ipAddress && { ip: IP_ADDRESS })), { insert_id: UUID(), partner_id: this.config.partnerId, plan: this.config.plan }), (this.config.ingestionMetadata && {
                    ingestion_metadata: {
                        source_name: this.config.ingestionMetadata.sourceName,
                        source_version: this.config.ingestionMetadata.sourceVersion,
                    },
                })), context), { event_id: nextEventId, library: this.library, user_agent: this.userAgent });
                return [2 /*return*/, event];
            });
        });
    };
    return Context;
}());
export { Context };
//# sourceMappingURL=context.js.map