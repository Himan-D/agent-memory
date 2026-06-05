import { IEventBridgeReceiver } from './event-bridge';
import { Event } from '../types/event/event';
export declare class EventBridgeChannel {
    channel: string;
    queue: Event[];
    receiver: IEventBridgeReceiver | undefined;
    constructor(channel: string);
    sendEvent(event: Event): void;
    setReceiver(receiver: IEventBridgeReceiver): void;
}
//# sourceMappingURL=event-bridge-channel.d.ts.map