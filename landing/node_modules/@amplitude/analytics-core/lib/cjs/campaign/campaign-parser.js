"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.CampaignParser = void 0;
var tslib_1 = require("tslib");
var query_params_1 = require("../query-params");
var constants_1 = require("../types/constants");
var CampaignParser = /** @class */ (function () {
    function CampaignParser() {
    }
    CampaignParser.prototype.parse = function () {
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            return tslib_1.__generator(this, function (_a) {
                return [2 /*return*/, tslib_1.__assign(tslib_1.__assign(tslib_1.__assign(tslib_1.__assign({}, constants_1.BASE_CAMPAIGN), this.getUtmParam()), this.getReferrer()), this.getClickIds())];
            });
        });
    };
    CampaignParser.prototype.getUtmParam = function () {
        var params = (0, query_params_1.getQueryParams)();
        var utmCampaign = params[constants_1.UTM_CAMPAIGN];
        var utmContent = params[constants_1.UTM_CONTENT];
        var utmId = params[constants_1.UTM_ID];
        var utmMedium = params[constants_1.UTM_MEDIUM];
        var utmSource = params[constants_1.UTM_SOURCE];
        var utmTerm = params[constants_1.UTM_TERM];
        return {
            utm_campaign: utmCampaign,
            utm_content: utmContent,
            utm_id: utmId,
            utm_medium: utmMedium,
            utm_source: utmSource,
            utm_term: utmTerm,
        };
    };
    CampaignParser.prototype.getReferrer = function () {
        var _a, _b;
        var data = {
            referrer: undefined,
            referring_domain: undefined,
        };
        try {
            data.referrer = document.referrer || undefined;
            data.referring_domain = (_b = (_a = data.referrer) === null || _a === void 0 ? void 0 : _a.split('/')[2]) !== null && _b !== void 0 ? _b : undefined;
        }
        catch (_c) {
            // nothing to track
        }
        return data;
    };
    CampaignParser.prototype.getClickIds = function () {
        var _a;
        var params = (0, query_params_1.getQueryParams)();
        return _a = {},
            _a[constants_1.DCLID] = params[constants_1.DCLID],
            _a[constants_1.FBCLID] = params[constants_1.FBCLID],
            _a[constants_1.GBRAID] = params[constants_1.GBRAID],
            _a[constants_1.GCLID] = params[constants_1.GCLID],
            _a[constants_1.KO_CLICK_ID] = params[constants_1.KO_CLICK_ID],
            _a[constants_1.LI_FAT_ID] = params[constants_1.LI_FAT_ID],
            _a[constants_1.MSCLKID] = params[constants_1.MSCLKID],
            _a[constants_1.RDT_CID] = params[constants_1.RDT_CID],
            _a[constants_1.TTCLID] = params[constants_1.TTCLID],
            _a[constants_1.TWCLID] = params[constants_1.TWCLID],
            _a[constants_1.WBRAID] = params[constants_1.WBRAID],
            _a;
    };
    return CampaignParser;
}());
exports.CampaignParser = CampaignParser;
//# sourceMappingURL=campaign-parser.js.map