export var UNSET_VALUE = '-';
export var AMPLITUDE_PREFIX = 'AMP';
export var STORAGE_PREFIX = "".concat(AMPLITUDE_PREFIX, "_unsent");
export var DEFAULT_INSTANCE_NAME = '$default_instance';
export var AMPLITUDE_SERVER_URL = 'https://api2.amplitude.com/2/httpapi';
export var EU_AMPLITUDE_SERVER_URL = 'https://api.eu.amplitude.com/2/httpapi';
export var AMPLITUDE_BATCH_SERVER_URL = 'https://api2.amplitude.com/batch';
export var EU_AMPLITUDE_BATCH_SERVER_URL = 'https://api.eu.amplitude.com/batch';
// Campaign constants
export var UTM_CAMPAIGN = 'utm_campaign';
export var UTM_CONTENT = 'utm_content';
export var UTM_ID = 'utm_id';
export var UTM_MEDIUM = 'utm_medium';
export var UTM_SOURCE = 'utm_source';
export var UTM_TERM = 'utm_term';
export var DCLID = 'dclid';
export var FBCLID = 'fbclid';
export var GBRAID = 'gbraid';
export var GCLID = 'gclid';
export var KO_CLICK_ID = 'ko_click_id';
export var LI_FAT_ID = 'li_fat_id';
export var MSCLKID = 'msclkid';
export var RDT_CID = 'rdt_cid';
export var TTCLID = 'ttclid';
export var TWCLID = 'twclid';
export var WBRAID = 'wbraid';
export var EMPTY_VALUE = 'EMPTY';
export var BASE_CAMPAIGN = {
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
export var MKTG = 'MKTG';
// list of Network headers that are safe to capture
export var SAFE_HEADERS = [
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
export var FORBIDDEN_HEADERS = ['authorization', 'cookie', 'set-cookie'];
//# sourceMappingURL=constants.js.map