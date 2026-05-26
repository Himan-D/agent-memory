"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.ServerZone = void 0;
/**
 * @deprecated use ServerZoneType instead
 */
var ServerZone;
(function (ServerZone) {
    ServerZone["US"] = "US";
    ServerZone["EU"] = "EU";
    /**
     * Add for session-replay-browser migration from analytics-type v1.x.
     */
    ServerZone["STAGING"] = "STAGING";
})(ServerZone = exports.ServerZone || (exports.ServerZone = {}));
//# sourceMappingURL=server-zone.js.map