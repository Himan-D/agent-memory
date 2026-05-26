"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getDomain = exports.KNOWN_2LDS = exports.getDefaultExcludedReferrers = exports.createCampaignEvent = exports.isSubdomainOf = exports.isExcludedReferrer = exports.isNewCampaign = void 0;
var tslib_1 = require("tslib");
var analytics_core_1 = require("@amplitude/analytics-core");
var types_1 = require("../types");
var domainWithoutSubdomain = function (domain) {
    var parts = domain.split('.');
    if (parts.length <= 2) {
        return domain;
    }
    return parts.slice(parts.length - 2, parts.length).join('.');
};
//Direct traffic mean no external referral, no UTMs, no click-ids, and no other customer identified marketing campaign url params.
var isDirectTraffic = function (current) {
    return Object.values(current).every(function (value) { return !value; });
};
var isEmptyCampaign = function (campaign) {
    var campaignWithoutReferrer = tslib_1.__assign(tslib_1.__assign({}, campaign), { referring_domain: undefined, referrer: undefined });
    return Object.values(campaignWithoutReferrer).every(function (value) { return !value; });
};
var isNewCampaign = function (current, previous, options, logger, isNewSession, topLevelDomain) {
    if (isNewSession === void 0) { isNewSession = true; }
    var referrer = current.referrer, referring_domain = current.referring_domain, currentCampaign = tslib_1.__rest(current, ["referrer", "referring_domain"]);
    var _a = previous || {}, _previous_referrer = _a.referrer, prevReferringDomain = _a.referring_domain, previousCampaign = tslib_1.__rest(_a, ["referrer", "referring_domain"]);
    var excludeInternalReferrers = options.excludeInternalReferrers;
    if (excludeInternalReferrers) {
        var condition = getExcludeInternalReferrersCondition(excludeInternalReferrers, logger);
        if (!(condition instanceof TypeError) &&
            current.referring_domain &&
            isInternalReferrer(current.referring_domain, topLevelDomain)) {
            if (condition === 'always') {
                debugLogInternalReferrerExclude(condition, current.referring_domain, logger);
                return false;
            }
            else if (condition === 'ifEmptyCampaign' && isEmptyCampaign(current)) {
                debugLogInternalReferrerExclude(condition, current.referring_domain, logger);
                return false;
            }
        }
    }
    if ((0, exports.isExcludedReferrer)(options.excludeReferrers, current.referring_domain)) {
        // eslint-disable-next-line @typescript-eslint/restrict-template-expressions
        logger.debug("This is not a new campaign because ".concat(current.referring_domain, " is in the exclude referrer list."));
        return false;
    }
    //In the same session, direct traffic should not override or unset any persisting query params
    if (!isNewSession && isDirectTraffic(current) && previous) {
        logger.debug('This is not a new campaign because this is a direct traffic in the same session.');
        return false;
    }
    var hasNewCampaign = JSON.stringify(currentCampaign) !== JSON.stringify(previousCampaign);
    var hasNewDomain = domainWithoutSubdomain(referring_domain || '') !== domainWithoutSubdomain(prevReferringDomain || '');
    var result = !previous || hasNewCampaign || hasNewDomain;
    if (!result) {
        logger.debug("This is not a new campaign because it's the same as the previous one.");
    }
    else {
        logger.debug("This is a new campaign. An $identify event will be sent.");
    }
    return result;
};
exports.isNewCampaign = isNewCampaign;
var isExcludedReferrer = function (excludeReferrers, referringDomain) {
    if (excludeReferrers === void 0) { excludeReferrers = []; }
    if (referringDomain === void 0) { referringDomain = ''; }
    return excludeReferrers.some(function (value) {
        return value instanceof RegExp ? value.test(referringDomain) : value === referringDomain;
    });
};
exports.isExcludedReferrer = isExcludedReferrer;
var isSubdomainOf = function (subDomain, domain) {
    var cookieDomainWithLeadingDot = domain.startsWith('.') ? domain : ".".concat(domain);
    var subDomainWithLeadingDot = subDomain.startsWith('.') ? subDomain : ".".concat(subDomain);
    if (subDomainWithLeadingDot.endsWith(cookieDomainWithLeadingDot))
        return true;
    return false;
};
exports.isSubdomainOf = isSubdomainOf;
var createCampaignEvent = function (campaign, options) {
    var campaignParameters = tslib_1.__assign(tslib_1.__assign({}, analytics_core_1.BASE_CAMPAIGN), campaign);
    var identifyEvent = Object.entries(campaignParameters).reduce(function (identify, _a) {
        var _b;
        var _c = tslib_1.__read(_a, 2), key = _c[0], value = _c[1];
        identify.setOnce("initial_".concat(key), (_b = value !== null && value !== void 0 ? value : options.initialEmptyValue) !== null && _b !== void 0 ? _b : 'EMPTY');
        if (value) {
            return identify.set(key, value);
        }
        return identify.unset(key);
    }, new analytics_core_1.Identify());
    return (0, analytics_core_1.createIdentifyEvent)(identifyEvent);
};
exports.createCampaignEvent = createCampaignEvent;
var getDefaultExcludedReferrers = function (cookieDomain) {
    var domain = cookieDomain;
    if (domain) {
        if (domain.startsWith('.')) {
            domain = domain.substring(1);
        }
        return [new RegExp("".concat(domain.replace('.', '\\.'), "$"))];
    }
    return [];
};
exports.getDefaultExcludedReferrers = getDefaultExcludedReferrers;
/**
 * Parses the excludeInternalReferrers configuration to determine the condition on which to
 * exclude internal referrers for campaign attribution.
 *
 * If the config is invalid type, log and return a TypeError.
 *
 * (this does explicit type checking so don't have to rely on TS compiler to catch invalid types)
 *
 * @param excludeInternalReferrers - attribution.excludeInternalReferrers configuration
 * @param logger - logger instance to log error when TypeError
 * @returns The condition if the config is valid, TypeError if the config is invalid.
 */
var getExcludeInternalReferrersCondition = function (excludeInternalReferrers, logger) {
    if (excludeInternalReferrers === true) {
        return types_1.EXCLUDE_INTERNAL_REFERRERS_CONDITIONS.always;
    }
    if (typeof excludeInternalReferrers === 'object') {
        var condition = excludeInternalReferrers.condition;
        if (typeof condition === 'string' && Object.keys(types_1.EXCLUDE_INTERNAL_REFERRERS_CONDITIONS).includes(condition)) {
            return condition;
        }
        else if (typeof condition === 'undefined') {
            return types_1.EXCLUDE_INTERNAL_REFERRERS_CONDITIONS.always;
        }
    }
    var errorMessage = "Invalid configuration provided for attribution.excludeInternalReferrers: ".concat(JSON.stringify(excludeInternalReferrers));
    logger.error(errorMessage);
    return new TypeError(errorMessage);
};
// helper function to log debug message when internal referrer is excluded
// (added this to prevent code duplication and improve readability)
function debugLogInternalReferrerExclude(condition, referringDomain, logger) {
    var baseMessage = "This is not a new campaign because referring_domain=".concat(referringDomain, " is on the same domain as the current page and it is configured to exclude internal referrers");
    if (condition === 'always') {
        logger.debug(baseMessage);
    }
    else if (condition === 'ifEmptyCampaign') {
        logger.debug("".concat(baseMessage, " with empty campaign parameters"));
    }
}
// list of domains that are known ccTLDs that are commonly used
// and are in the Public Suffix List (https://publicsuffix.org/)
exports.KNOWN_2LDS = [
    'ac.in',
    'ac.jp',
    'ac.kr',
    'ac.th',
    'ac.uk',
    'ac.za',
    'appspot.com',
    'asn.au',
    'azurewebsites.net',
    'cloudfront.net',
    'myshopify.com',
    'blogspot.com',
    'co.ca',
    'co.in',
    'co.jp',
    'co.kr',
    'co.nz',
    'co.th',
    'co.uk',
    'co.za',
    'com.ar',
    'com.au',
    'com.br',
    'com.cn',
    'com.hk',
    'com.in',
    'com.jp',
    'com.kr',
    'com.mx',
    'com.pl',
    'com.sg',
    'com.tr',
    'com.tw',
    'ed.jp',
    'edu.au',
    'edu.br',
    'edu.cn',
    'edu.hk',
    'edu.sg',
    'edu.th',
    'edu.tr',
    'edu.tw',
    'firebaseapp.com',
    'fly.dev',
    'gc.ca',
    'geek.nz',
    'github.io',
    'gitlab.io',
    'go.jp',
    'go.kr',
    'go.th',
    'gob.ar',
    'gob.mx',
    'gov.au',
    'gov.br',
    'gov.cn',
    'gov.hk',
    'gov.in',
    'gov.pl',
    'gov.sg',
    'gov.tr',
    'gov.tw',
    'gov.uk',
    'gov.za',
    'govt.nz',
    'gr.jp',
    'herokuapp.com',
    'id.au',
    'idv.hk',
    'iwi.nz',
    'lg.jp',
    'ltd.uk',
    'maori.nz',
    'me.uk',
    'mil.kr',
    'ne.jp',
    'ne.kr',
    'net.au',
    'net.br',
    'net.cn',
    'net.hk',
    'net.in',
    'net.nz',
    'net.pl',
    'net.sg',
    'net.tr',
    'net.tw',
    'net.za',
    'onrender.com',
    'or.jp',
    'or.kr',
    'or.th',
    'org.ar',
    'org.au',
    'org.br',
    'org.cn',
    'org.hk',
    'org.in',
    'org.mx',
    'org.nz',
    'org.pl',
    'org.sg',
    'org.tw',
    'org.uk',
    'org.za',
    'pages.dev',
    'pe.kr',
    'plc.uk',
    're.kr',
    'res.in',
    'sch.uk',
    'vercel.app',
    'netlify.app',
    'workers.dev',
];
var getDomain = function (hostnameParam) {
    var _a, _b;
    /* istanbul ignore next */
    var hostname = hostnameParam || ((_b = (_a = (0, analytics_core_1.getGlobalScope)()) === null || _a === void 0 ? void 0 : _a.location) === null || _b === void 0 ? void 0 : _b.hostname);
    if (!hostname) {
        return '';
    }
    var parts = hostname.split('.');
    var tld = parts[parts.length - 1];
    var name = parts[parts.length - 2];
    if (exports.KNOWN_2LDS.find(function (tld) { return hostname.endsWith(".".concat(tld)); })) {
        tld = parts[parts.length - 2] + '.' + parts[parts.length - 1];
        name = parts[parts.length - 3];
    }
    if (!name)
        return tld;
    return "".concat(name, ".").concat(tld);
};
exports.getDomain = getDomain;
var isInternalReferrer = function (referringDomain, topLevelDomain) {
    var globalScope = (0, analytics_core_1.getGlobalScope)();
    /* istanbul ignore if */
    if (!globalScope)
        return false;
    // if referring domain is subdomain of config.cookieDomain, return true
    var internalDomain = (topLevelDomain || '').trim() || (0, exports.getDomain)(globalScope.location.hostname);
    return (0, exports.isSubdomainOf)(referringDomain, internalDomain);
};
//# sourceMappingURL=helpers.js.map