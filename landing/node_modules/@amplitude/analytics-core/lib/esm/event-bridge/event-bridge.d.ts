import { Event } from '../types/event/event';
import { EventBridgeChannel } from './event-bridge-channel';
export interface IEventBridge {
    sendEvent(channel: string, event: Event): void;
    setReceiver(channel: string, receiver: IEventBridgeReceiver): void;
}
export interface IEventBridgeReceiver {
    receive(channel: string, event: Event): void;
}
export declare class EventBridge implements IEventBridge {
    eventBridgeChannels: Record<string, EventBridgeChannel | undefined>;
    sendEvent(channel: string, event: Event): void;
    setReceiver(channel: string, receiver: IEventBridgeReceiver): void;
}
//# sourceMappingURL=event-bridge.d.ts.map