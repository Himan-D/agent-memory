import { VideoHandler, EmbeddedVideoPlayer, MuxElement, Vendor } from './types';
/**
 * Track a standard HTML video element.
 *
 * @param videoEl - The HTML video element to track.
 * @param handlers - The video handlers to call when on video lifecycle events.
 * @returns A function to untrack the video.
 */
export declare function trackHtmlVideo(videoEl: HTMLVideoElement | MuxElement, handlers: VideoHandler, vendor?: Vendor): () => void;
export declare function trackEmbeddedVideo(player: EmbeddedVideoPlayer, handlers: VideoHandler, vendor?: Vendor | null): () => void;
//# sourceMappingURL=track-video.d.ts.map