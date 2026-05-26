import { LogLevel } from './types/loglevel';
export interface ILogger {
    disable(): void;
    enable(logLevel: LogLevel): void;
    log(...args: any[]): void;
    warn(...args: any[]): void;
    error(...args: any[]): void;
    debug(...args: any[]): void;
}
export interface LogConfig {
    logger: ILogger;
    logLevel: LogLevel;
}
type TimeKey = 'start' | 'end';
export interface DebugContext {
    type: string;
    name: string;
    args: string[] | string;
    stacktrace?: string[] | string;
    time?: {
        [key in TimeKey]?: string;
    };
    states?: {
        [key: string]: any;
    };
}
export declare class Logger implements ILogger {
    logLevel: LogLevel;
    constructor();
    disable(): void;
    enable(logLevel?: LogLevel): void;
    log(...args: any[]): void;
    warn(...args: any[]): void;
    error(...args: any[]): void;
    debug(...args: any[]): void;
}
export {};
//# sourceMappingURL=logger.d.ts.map