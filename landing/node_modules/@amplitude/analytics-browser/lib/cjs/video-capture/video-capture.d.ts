import { BrowserClient, EmbeddedVideoPlayer, VideoVendor } from '@amplitude/analytics-core';
export declare class VideoCapture {
    private readonly amplitude;
    private videoEl;
    private embeddedVideoPlayer;
    private vendor?;
    private extraEventProperties;
    private listeners;
    private onRemoveListeners;
    constructor(amplitude: BrowserClient);
    /**
     * Specify a video element to capture events from
     *
     * @param videoEl - The HTML video element to capture events from.
     * @returns The VideoCapture instance.
     */
    withVideoElement(videoEl: HTMLVideoElement): VideoCapture;
    /**
     * Specify an embedded video player.js instance to capture events from
     * @param player - The embedded video player.js instance to capture events from.
     * @returns The VideoCapture instance.
     */
    withEmbeddedPlayer(player: EmbeddedVideoPlayer): VideoCapture;
    /**
     * Specify a vendor to capture extra vendor-specific event properties
     *
     * @param vendor - The vendor of the video player. Currently only "mux" is supported.
     * @returns The VideoCapture instance.
     */
    withVendor(vendor: VideoVendor): VideoCapture;
    /**
     * Specify extra event properties to include in all captured events
     *
     * @param properties - The extra event properties to include in the Amplitude event.
     * @returns The VideoCapture instance.
     */
    withExtraEventProperties(properties: Record<string, string | number | boolean>): VideoCapture;
    /**
     * Track a "Video Content Started" event every time the video starts playing
     * @returns The VideoCapture instance.
     */
    captureVideoStarted(): VideoCapture;
    /**
     * Track a "Video Content Stopped" event every time the video stops playing
     * @returns The VideoCapture instance.
     */
    captureVideoStopped(): VideoCapture;
    /**
     * Start capturing analytics events for the video element
     * @returns The VideoCapture instance.
     * @throws An error if the video element is not specified.
     */
    start(): VideoCapture;
    stop(): void;
}
export type VideoCaptureOptions = {
    vendor?: VideoVendor;
    extraEventProperties?: Record<string, string | number | boolean>;
};
/**
 * Track video analytics events for an HTML video element or embedded video player.js instance.
 *
 * Captures Video Started and Video Stopped events.
 *
 * @experimental This function is experimental and may not be stable.
 * @param amplitude - The Amplitude client instance.
 * @param videoEl - The HTML video element or embedded video player.js instance to capture events from.
 * @param options - The options for the video capture.
 * @returns A function to stop the video capture.
 */
export declare function trackVideo(amplitude: BrowserClient, videoEl: HTMLVideoElement | EmbeddedVideoPlayer, options?: VideoCaptureOptions): () => void;
//# sourceMappingURL=video-capture.d.ts.map