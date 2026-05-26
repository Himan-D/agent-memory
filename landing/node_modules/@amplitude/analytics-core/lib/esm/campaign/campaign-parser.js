import { __assign, __awaiter, __generator } from "tslib";
import { getQueryParams } from '../query-params';
import { UTM_CAMPAIGN, UTM_CONTENT, UTM_MEDIUM, UTM_SOURCE, UTM_TERM, GCLID, FBCLID, BASE_CAMPAIGN, DCLID, MSCLKID, RDT_CID, TWCLID, TTCLID, KO_CLICK_ID, LI_FAT_ID, GBRAID, WBRAID, UTM_ID, } from '../types/constants';
var CampaignParser = /** @class */ (function () {
    function CampaignParser() {
    }
    CampaignParser.prototype.parse = function () {
        return __awaiter(this, void 0, void 0, function () {
            return __generator(this, function (_a) {
                return [2 /*return*/, __assign(__assign(__assign(__assign({}, BASE_CAMPAIGN), this.getUtmParam()), this.getReferrer()), this.getClickIds())];
            });
        });
    };
    CampaignParser.prototype.getUtmParam = function () {
        var params = getQueryParams();
        var utmCampaign = params[UTM_CAMPAIGN];
        var utmContent = params[UTM_CONTENT];
        var utmId = params[UTM_ID];
        var utmMedium = params[UTM_MEDIUM];
        var utmSource = params[UTM_SOURCE];
        var utmTerm = params[UTM_TERM];
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
        var params = getQueryParams();
        return _a = {},
            _a[DCLID] = params[DCLID],
            _a[FBCLID] = params[FBCLID],
            _a[GBRAID] = params[GBRAID],
            _a[GCLID] = params[GCLID],
            _a[KO_CLICK_ID] = params[KO_CLICK_ID],
            _a[LI_FAT_ID] = params[LI_FAT_ID],
            _a[MSCLKID] = params[MSCLKID],
            _a[RDT_CID] = params[RDT_CID],
            _a[TTCLID] = params[TTCLID],
            _a[TWCLID] = params[TWCLID],
            _a[WBRAID] = params[WBRAID],
            _a;
    };
    return CampaignParser;
}());
export { CampaignParser };
//# sourceMappingURL=campaign-parser.js.map