import { __assign, __awaiter, __generator, __read } from "tslib";
import { getStorageKey, isNewSession, BASE_CAMPAIGN, CampaignParser, } from '@amplitude/analytics-core';
import { getDefaultExcludedReferrers, createCampaignEvent, isNewCampaign } from './helpers';
var WebAttribution = /** @class */ (function () {
    function WebAttribution(options, config) {
        var _a;
        this.shouldTrackNewCampaign = false;
        this.options = __assign({ initialEmptyValue: 'EMPTY', resetSessionOnNewCampaign: false, excludeReferrers: getDefaultExcludedReferrers(((_a = config.cookieOptions) === null || _a === void 0 ? void 0 : _a.domain) || config.topLevelDomain), optOut: config.optOut }, options);
        this.storage = config.cookieStorage;
        this.storageKey = getStorageKey(config.apiKey, 'MKTG');
        this.webExpStorageKey = getStorageKey(config.apiKey, 'MKTG_ORIGINAL');
        this.currentCampaign = BASE_CAMPAIGN;
        this.sessionTimeout = config.sessionTimeout;
        this.lastEventTime = config.lastEventTime;
        this.logger = config.loggerProvider;
        this.topLevelDomain = config.topLevelDomain;
        config.loggerProvider.log('Installing web attribution tracking.');
    }
    WebAttribution.prototype.init = function () {
        return __awaiter(this, void 0, void 0, function () {
            var isEventInNewSession;
            var _a;
            return __generator(this, function (_b) {
                switch (_b.label) {
                    case 0:
                        // skip attribution if optOut is true
                        if (this.options.optOut) {
                            return [2 /*return*/];
                        }
                        return [4 /*yield*/, this.fetchCampaign()];
                    case 1:
                        _a = __read.apply(void 0, [_b.sent(), 2]), this.currentCampaign = _a[0], this.previousCampaign = _a[1];
                        isEventInNewSession = !this.lastEventTime ? true : isNewSession(this.sessionTimeout, this.lastEventTime);
                        if (!isNewCampaign(this.currentCampaign, this.previousCampaign, this.options, this.logger, isEventInNewSession, this.topLevelDomain)) return [3 /*break*/, 3];
                        this.shouldTrackNewCampaign = true;
                        return [4 /*yield*/, this.storage.set(this.storageKey, this.currentCampaign)];
                    case 2:
                        _b.sent();
                        _b.label = 3;
                    case 3: return [2 /*return*/];
                }
            });
        });
    };
    WebAttribution.prototype.fetchCampaign = function () {
        return __awaiter(this, void 0, void 0, function () {
            var originalCampaign;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, this.storage.get(this.webExpStorageKey)];
                    case 1:
                        originalCampaign = _a.sent();
                        if (!originalCampaign) return [3 /*break*/, 3];
                        return [4 /*yield*/, this.storage.remove(this.webExpStorageKey)];
                    case 2:
                        _a.sent();
                        _a.label = 3;
                    case 3: return [4 /*yield*/, Promise.all([originalCampaign || new CampaignParser().parse(), this.storage.get(this.storageKey)])];
                    case 4: return [2 /*return*/, _a.sent()];
                }
            });
        });
    };
    /**
     * This can be called when enable web attribution and either
     * 1. set a new session
     * 2. has new campaign and enable resetSessionOnNewCampaign
     */
    WebAttribution.prototype.generateCampaignEvent = function (event_id) {
        // Mark this campaign has been tracked
        this.shouldTrackNewCampaign = false;
        var campaignEvent = createCampaignEvent(this.currentCampaign, this.options);
        if (event_id) {
            campaignEvent.event_id = event_id;
        }
        return campaignEvent;
    };
    WebAttribution.prototype.shouldSetSessionIdOnNewCampaign = function () {
        return this.shouldTrackNewCampaign && !!this.options.resetSessionOnNewCampaign;
    };
    return WebAttribution;
}());
export { WebAttribution };
//# sourceMappingURL=web-attribution.js.map