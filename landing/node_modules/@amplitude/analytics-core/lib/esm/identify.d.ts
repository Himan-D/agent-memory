export interface IIdentify {
    getUserProperties(): IdentifyUserProperties;
    set(property: string, value: ValidPropertyType): IIdentify;
    setOnce(property: string, value: ValidPropertyType): IIdentify;
    append(property: string, value: ValidPropertyType): IIdentify;
    prepend(property: string, value: ValidPropertyType): IIdentify;
    postInsert(property: string, value: ValidPropertyType): IIdentify;
    preInsert(property: string, value: ValidPropertyType): IIdentify;
    remove(property: string, value: ValidPropertyType): IIdentify;
    add(property: string, value: number): IIdentify;
    unset(property: string): IIdentify;
    clearAll(): IIdentify;
}
export declare class Identify implements IIdentify {
    protected readonly _propertySet: Set<string>;
    protected _properties: IdentifyUserProperties;
    getUserProperties(): IdentifyUserProperties;
    set(property: string, value: ValidPropertyType): Identify;
    setOnce(property: string, value: ValidPropertyType): Identify;
    append(property: string, value: ValidPropertyType): Identify;
    prepend(property: string, value: ValidPropertyType): Identify;
    postInsert(property: string, value: ValidPropertyType): Identify;
    preInsert(property: string, value: ValidPropertyType): Identify;
    remove(property: string, value: ValidPropertyType): Identify;
    add(property: string, value: number): Identify;
    unset(property: string): Identify;
    clearAll(): Identify;
    private _safeSet;
    private _validate;
}
export type ValidPropertyType = number | string | boolean | Array<string | number> | {
    [key: string]: ValidPropertyType;
} | Array<{
    [key: string]: ValidPropertyType;
}>;
interface BaseOperationConfig {
    [key: string]: ValidPropertyType;
}
export interface IdentifyUserProperties {
    [IdentifyOperation.ADD]?: {
        [key: string]: number;
    };
    [IdentifyOperation.UNSET]?: BaseOperationConfig;
    [IdentifyOperation.CLEAR_ALL]?: any;
    [IdentifyOperation.SET]?: BaseOperationConfig;
    [IdentifyOperation.SET_ONCE]?: BaseOperationConfig;
    [IdentifyOperation.APPEND]?: BaseOperationConfig;
    [IdentifyOperation.PREPEND]?: BaseOperationConfig;
    [IdentifyOperation.POSTINSERT]?: BaseOperationConfig;
    [IdentifyOperation.PREINSERT]?: BaseOperationConfig;
    [IdentifyOperation.REMOVE]?: BaseOperationConfig;
}
export declare enum IdentifyOperation {
    SET = "$set",
    SET_ONCE = "$setOnce",
    ADD = "$add",
    APPEND = "$append",
    PREPEND = "$prepend",
    REMOVE = "$remove",
    PREINSERT = "$preInsert",
    POSTINSERT = "$postInsert",
    UNSET = "$unset",
    CLEAR_ALL = "$clearAll"
}
/**
 * Note that the order of operations should align with https://github.com/amplitude/nova/blob/7701b5986b565d4b2fb53b99a9f2175df055dea8/src/main/java/com/amplitude/ingestion/core/UserPropertyUtils.java#L210
 */
export declare const OrderedIdentifyOperations: IdentifyOperation[];
export {};
//# sourceMappingURL=identify.d.ts.map