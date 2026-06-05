"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.IdentityEventSender = void 0;
var tslib_1 = require("tslib");
var analytics_connector_1 = require("../analytics-connector");
var IdentityEventSender = /** @class */ (function () {
    function IdentityEventSender() {
        this.name = 'identity';
        this.type = 'before';
        this.identityStore = (0, analytics_connector_1.getAnalyticsConnector)().identityStore;
    }
    IdentityEventSender.prototype.execute = function (context) {
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            var userProperties;
            return tslib_1.__generator(this, function (_a) {
                userProperties = context.user_properties;
                if (userProperties) {
                    this.identityStore.editIdentity().updateUserProperties(userProperties).commit();
                }
                return [2 /*return*/, context];
            });
        });
    };
    IdentityEventSender.prototype.setup = function (config) {
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            return tslib_1.__generator(this, function (_a) {
                if (config.instanceName) {
                    this.identityStore = (0, analytics_connector_1.getAnalyticsConnector)(config.instanceName).identityStore;
                }
                return [2 /*return*/];
            });
        });
    };
    return IdentityEventSender;
}());
exports.IdentityEventSender = IdentityEventSender;
//# sourceMappingURL=identity.js.map