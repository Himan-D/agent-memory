"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.FORBIDDEN_HEADERS = exports.SAFE_HEADERS = exports.MKTG = exports.BASE_CAMPAIGN = exports.EMPTY_VALUE = exports.WBRAID = exports.TWCLID = exports.TTCLID = exports.RDT_CID = exports.MSCLKID = exports.LI_FAT_ID = exports.KO_CLICK_ID = exports.GCLID = exports.GBRAID = exports.FBCLID = exports.DCLID = exports.UTM_TERM = exports.UTM_SOURCE = exports.UTM_MEDIUM = exports.UTM_ID = exports.UTM_CONTENT = exports.UTM_CAMPAIGN = exports.EU_AMPLITUDE_BATCH_SERVER_URL = exports.AMPLITUDE_BATCH_SERVER_URL = exports.EU_AMPLITUDE_SERVER_URL = exports.AMPLITUDE_SERVER_URL = exports.DEFAULT_INSTANCE_NAME = exports.STORAGE_PREFIX = exports.AMPLITUDE_PREFIX = exports.UNSET_VALUE = void 0;
exports.UNSET_VALUE = '-';
exports.AMPLITUDE_PREFIX = 'AMP';
exports.STORAGE_PREFIX = "".concat(exports.AMPLITUDE_PREFIX, "_unsent");
exports.DEFAULT_INSTANCE_NAME = '$default_instance';
exports.AMPLITUDE_SERVER_URL = 'https://api2.amplitude.com/2/httpapi';
exports.EU_AMPLITUDE_SERVER_URL = 'https://api.eu.amplitude.com/2/httpapi';
exports.AMPLITUDE_BATCH_SERVER_URL = 'https://api2.amplitude.com/batch';
exports.EU_AMPLITUDE_BATCH_SERVER_URL = 'https://api.eu.amplitude.com/batch';
// Campaign constants
exports.UTM_CAMPAIGN = 'utm_campaign';
exports.UTM_CONTENT = 'utm_content';
exports.UTM_ID = 'utm_id';
exports.UTM_MEDIUM = 'utm_medium';
exports.UTM_SOURCE = 'utm_source';
exports.UTM_TERM = 'utm_term';
exports.DCLID = 'dclid';
exports.FBCLID = 'fbclid';
exports.GBRAID = 'gbraid';
exports.GCLID = 'gclid';
exports.KO_CLICK_ID = 'ko_click_id';
exports.LI_FAT_ID = 'li_fat_id';
exports.MSCLKID = 'msclkid';
exports.RDT_CID = 'rdt_cid';
exports.TTCLID = 'ttclid';
exports.TWCLID = 'twclid';
exports.WBRAID = 'wbraid';
exports.EMPTY_VALUE = 'EMPTY';
exports.BASE_CAMPAIGN = {
    utm_campaign: undefined,
    utm_content: undefined,
    utm_id: undefined,
    utm_medium: undefined,
    utm_source: undefined,
    utm_term: undefined,
    referrer: undefined,
    referring_domain: undefined,
    dclid: undefined,
    gbraid: undefined,
    gclid: undefined,
    fbclid: undefined,
    ko_click_id: undefined,
    li_fat_id: undefined,
    msclkid: undefined,
    rdt_cid: undefined,
    ttclid: undefined,
    twclid: undefined,
    wbraid: undefined,
};
exports.MKTG = 'MKTG';
// list of Network headers that are safe to capture
exports.SAFE_HEADERS = [
    'access-control-allow-origin',
    'access-control-allow-credentials',
    'access-control-expose-headers',
    'access-control-max-age',
    'access-control-allow-methods',
    'access-control-allow-headers',
    'accept-patch',
    'accept-ranges',
    'age',
    'allow',
    'alt-svc',
    'cache-control',
    'connection',
    'content-disposition',
    'content-encoding',
    'content-language',
    'content-length',
    'content-location',
    'content-md5',
    'content-range',
    'content-type',
    'date',
    'delta-base',
    'etag',
    'expires',
    'im',
    'last-modified',
    'link',
    'location',
    'permanent',
    'p3p',
    'pragma',
    'proxy-authenticate',
    'public-key-pins',
    'retry-after',
    'server',
    'status',
    'strict-transport-security',
    'trailer',
    'transfer-encoding',
    'tk',
    'upgrade',
    'vary',
    'via',
    'warning',
    'www-authenticate',
    'x-b3-traceid',
    'x-frame-options',
];
// list of Network headers to never capture
exports.FORBIDDEN_HEADERS = ['authorization', 'cookie', 'set-cookie'];
//# sourceMappingURL=constants.js.map