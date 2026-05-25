"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.OrderedIdentifyOperations = exports.IdentifyOperation = exports.Identify = void 0;
var tslib_1 = require("tslib");
var constants_1 = require("./types/constants");
var valid_properties_1 = require("./utils/valid-properties");
var Identify = /** @class */ (function () {
    function Identify() {
        this._propertySet = new Set();
        this._properties = {};
    }
    Identify.prototype.getUserProperties = function () {
        return tslib_1.__assign({}, this._properties);
    };
    Identify.prototype.set = function (property, value) {
        this._safeSet(IdentifyOperation.SET, property, value);
        return this;
    };
    Identify.prototype.setOnce = function (property, value) {
        this._safeSet(IdentifyOperation.SET_ONCE, property, value);
        return this;
    };
    Identify.prototype.append = function (property, value) {
        this._safeSet(IdentifyOperation.APPEND, property, value);
        return this;
    };
    Identify.prototype.prepend = function (property, value) {
        this._safeSet(IdentifyOperation.PREPEND, property, value);
        return this;
    };
    Identify.prototype.postInsert = function (property, value) {
        this._safeSet(IdentifyOperation.POSTINSERT, property, value);
        return this;
    };
    Identify.prototype.preInsert = function (property, value) {
        this._safeSet(IdentifyOperation.PREINSERT, property, value);
        return this;
    };
    Identify.prototype.remove = function (property, value) {
        this._safeSet(IdentifyOperation.REMOVE, property, value);
        return this;
    };
    Identify.prototype.add = function (property, value) {
        this._safeSet(IdentifyOperation.ADD, property, value);
        return this;
    };
    Identify.prototype.unset = function (property) {
        this._safeSet(IdentifyOperation.UNSET, property, constants_1.UNSET_VALUE);
        return this;
    };
    Identify.prototype.clearAll = function () {
        // When clear all happens, all properties are unset. Reset the entire object.
        this._properties = {};
        this._properties[IdentifyOperation.CLEAR_ALL] = constants_1.UNSET_VALUE;
        return this;
    };
    // Returns whether or not this set actually worked.
    Identify.prototype._safeSet = function (operation, property, value) {
        if (this._validate(operation, property, value)) {
            // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
            var userPropertyMap = this._properties[operation];
            if (userPropertyMap === undefined) {
                userPropertyMap = {};
                // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
                this._properties[operation] = userPropertyMap;
            }
            // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
            userPropertyMap[property] = value;
            this._propertySet.add(property);
            return true;
        }
        return false;
    };
    Identify.prototype._validate = function (operation, property, value) {
        if (this._properties[IdentifyOperation.CLEAR_ALL] !== undefined) {
            // clear all already set. Skipping operation;
            return false;
        }
        if (this._propertySet.has(property)) {
            // Property already used. Skipping operation
            return false;
        }
        if (operation === IdentifyOperation.ADD) {
            return typeof value === 'number';
        }
        if (operation !== IdentifyOperation.UNSET && operation !== IdentifyOperation.REMOVE) {
            return (0, valid_properties_1.isValidProperties)(property, value);
        }
        return true;
    };
    return Identify;
}());
exports.Identify = Identify;
var IdentifyOperation;
(function (IdentifyOperation) {
    // Base Operations to set values
    IdentifyOperation["SET"] = "$set";
    IdentifyOperation["SET_ONCE"] = "$setOnce";
    // Operations around modifying existing values
    IdentifyOperation["ADD"] = "$add";
    IdentifyOperation["APPEND"] = "$append";
    IdentifyOperation["PREPEND"] = "$prepend";
    IdentifyOperation["REMOVE"] = "$remove";
    // Operations around appending values *if* they aren't present
    IdentifyOperation["PREINSERT"] = "$preInsert";
    IdentifyOperation["POSTINSERT"] = "$postInsert";
    // Operations around removing properties/values
    IdentifyOperation["UNSET"] = "$unset";
    IdentifyOperation["CLEAR_ALL"] = "$clearAll";
})(IdentifyOperation = exports.IdentifyOperation || (exports.IdentifyOperation = {}));
/**
 * Note that the order of operations should align with https://github.com/amplitude/nova/blob/7701b5986b565d4b2fb53b99a9f2175df055dea8/src/main/java/com/amplitude/ingestion/core/UserPropertyUtils.java#L210
 */
exports.OrderedIdentifyOperations = [
    IdentifyOperation.CLEAR_ALL,
    IdentifyOperation.UNSET,
    IdentifyOperation.SET,
    IdentifyOperation.SET_ONCE,
    IdentifyOperation.ADD,
    IdentifyOperation.APPEND,
    IdentifyOperation.PREPEND,
    IdentifyOperation.PREINSERT,
    IdentifyOperation.POSTINSERT,
    IdentifyOperation.REMOVE,
];
//# sourceMappingURL=identify.js.map