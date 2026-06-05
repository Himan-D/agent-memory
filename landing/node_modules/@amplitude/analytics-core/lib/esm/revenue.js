import { __assign } from "tslib";
import { isValidObject } from './utils/valid-properties';
var Revenue = /** @class */ (function () {
    function Revenue() {
        this.productId = '';
        this.quantity = 1;
        this.price = 0.0;
    }
    Revenue.prototype.setProductId = function (productId) {
        this.productId = productId;
        return this;
    };
    Revenue.prototype.setQuantity = function (quantity) {
        if (quantity > 0) {
            this.quantity = quantity;
        }
        return this;
    };
    Revenue.prototype.setPrice = function (price) {
        this.price = price;
        return this;
    };
    Revenue.prototype.setRevenueType = function (revenueType) {
        this.revenueType = revenueType;
        return this;
    };
    Revenue.prototype.setCurrency = function (currency) {
        this.currency = currency;
        return this;
    };
    Revenue.prototype.setRevenue = function (revenue) {
        this.revenue = revenue;
        return this;
    };
    Revenue.prototype.setReceipt = function (receipt) {
        this.receipt = receipt;
        return this;
    };
    Revenue.prototype.setReceiptSig = function (receiptSig) {
        this.receiptSig = receiptSig;
        return this;
    };
    Revenue.prototype.setEventProperties = function (properties) {
        try {
            // JSON.stringify drops undefined/function/symbol values before validation,
            // so a single undefined property no longer causes the whole object to be rejected.
            var filtered = JSON.parse(JSON.stringify(properties));
            if (isValidObject(filtered)) {
                this.properties = filtered;
            }
        }
        catch (_a) {
            // no-op: invalid properties are ignored
        }
        return this;
    };
    Revenue.prototype.getEventProperties = function () {
        var eventProperties = this.properties ? __assign({}, this.properties) : {};
        eventProperties[RevenueProperty.REVENUE_PRODUCT_ID] = this.productId;
        eventProperties[RevenueProperty.REVENUE_QUANTITY] = this.quantity;
        eventProperties[RevenueProperty.REVENUE_PRICE] = this.price;
        eventProperties[RevenueProperty.REVENUE_TYPE] = this.revenueType;
        eventProperties[RevenueProperty.REVENUE_CURRENCY] = this.currency;
        eventProperties[RevenueProperty.REVENUE] = this.revenue;
        eventProperties[RevenueProperty.RECEIPT] = this.receipt;
        eventProperties[RevenueProperty.RECEIPT_SIG] = this.receiptSig;
        return eventProperties;
    };
    return Revenue;
}());
export { Revenue };
export var RevenueProperty;
(function (RevenueProperty) {
    RevenueProperty["REVENUE_PRODUCT_ID"] = "$productId";
    RevenueProperty["REVENUE_QUANTITY"] = "$quantity";
    RevenueProperty["REVENUE_PRICE"] = "$price";
    RevenueProperty["REVENUE_TYPE"] = "$revenueType";
    RevenueProperty["REVENUE_CURRENCY"] = "$currency";
    RevenueProperty["REVENUE"] = "$revenue";
    RevenueProperty["RECEIPT"] = "$receipt";
    RevenueProperty["RECEIPT_SIG"] = "$receiptSig";
})(RevenueProperty || (RevenueProperty = {}));
//# sourceMappingURL=revenue.js.map