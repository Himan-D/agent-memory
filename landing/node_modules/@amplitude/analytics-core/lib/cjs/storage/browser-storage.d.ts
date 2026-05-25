import { Storage as AmplitudeStorage } from '../types/storage';
export declare class BrowserStorage<T> implements AmplitudeStorage<T> {
    private storage?;
    constructor(storage?: Storage | undefined);
    isEnabled(): Promise<boolean>;
    get(key: string): Promise<T | undefined>;
    getRaw(key: string): Promise<string | undefined>;
    set(key: string, value: T): Promise<void>;
    remove(key: string): Promise<void>;
    reset(): Promise<void>;
}
//# sourceMappingURL=browser-storage.d.ts.map