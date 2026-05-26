export interface IRevenue {
    getEventProperties(): RevenueEventProperties;
    setProductId(productId: string): IRevenue;
    setQuantity(quantity: number): IRevenue;
    setPrice(price: number): IRevenue;
    setRevenueType(revenueType: string): IRevenue;
    setCurrency(currency: string): IRevenue;
    setEventProperties(properties: {
        [key: string]: any;
    }): IRevenue;
    setRevenue(revenue: number): IRevenue;
    setReceipt(receipt: string): IRevenue;
    setReceiptSig(receiptSig: string): IRevenue;
}
export declare class Revenue implements IRevenue {
    private productId;
    private quantity;
    private price;
    private revenueType?;
    private currency?;
    private properties?;
    private revenue?;
    private receipt?;
    private receiptSig?;
    constructor();
    setProductId(productId: string): this;
    setQuantity(quantity: number): this;
    setPrice(price: number): this;
    setRevenueType(revenueType: string): this;
    setCurrency(currency: string): this;
    setRevenue(revenue: number): this;
    setReceipt(receipt: string): this;
    setReceiptSig(receiptSig: string): this;
    setEventProperties(properties: {
        [key: string]: ValidPropertyType;
    }): this;
    getEventProperties(): RevenueEventProperties;
}
export interface RevenueEventProperties {
    [RevenueProperty.REVENUE_PRODUCT_ID]?: string;
    [RevenueProperty.REVENUE_QUANTITY]?: number;
    [RevenueProperty.REVENUE_PRICE]?: number;
    [RevenueProperty.REVENUE_TYPE]?: string;
    [RevenueProperty.REVENUE_CURRENCY]?: string;
    [RevenueProperty.REVENUE]?: number;
    [RevenueProperty.RECEIPT]?: string;
    [RevenueProperty.RECEIPT_SIG]?: string;
}
export declare enum RevenueProperty {
    REVENUE_PRODUCT_ID = "$productId",
    REVENUE_QUANTITY = "$quantity",
    REVENUE_PRICE = "$price",
    REVENUE_TYPE = "$revenueType",
    REVENUE_CURRENCY = "$currency",
    REVENUE = "$revenue",
    RECEIPT = "$receipt",
    RECEIPT_SIG = "$receiptSig"
}
export type ValidPropertyType = number | string | boolean | Array<string | number> | {
    [key: string]: ValidPropertyType;
} | Array<{
    [key: string]: ValidPropertyType;
}>;
//# sourceMappingURL=revenue.d.ts.map