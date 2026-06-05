export interface AmplitudeReturn<T> {
    promise: Promise<T>;
}
export declare const returnWrapper: {
    (): AmplitudeReturn<void>;
    <T>(awaitable: Promise<T>): AmplitudeReturn<T>;
};
//# sourceMappingURL=return-wrapper.d.ts.map