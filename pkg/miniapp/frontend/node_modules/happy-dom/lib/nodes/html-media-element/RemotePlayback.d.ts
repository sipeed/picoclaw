import EventTarget from '../../event/EventTarget.js';
import * as PropertySymbol from '../../PropertySymbol.js';
/**
 * RemotePlayback.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/RemotePlayback
 */
export default class RemotePlayback extends EventTarget {
    [PropertySymbol.state]: 'connecting' | 'connected' | 'disconnected';
    onconnecting: (event: Event) => void;
    onconnect: (event: Event) => void;
    ondisconnect: (event: Event) => void;
    /**
     * Returns the state of the remote playback.
     */
    get state(): 'connecting' | 'connected' | 'disconnected';
    /**
     * Watches the list of available remote playback devices and returns a Promise that resolves with a callbackId of an available remote playback device.
     *
     * @returns Promise.
     */
    watchAvailability(): Promise<void>;
    /**
     * Cancels the request to monitor the availability of remote playback devices.
     */
    cancelWatchAvailability(): void;
    /**
     * Prompts the user to select and give permission to connect to a remote playback device.
     */
    prompt(): void;
}
//# sourceMappingURL=RemotePlayback.d.ts.map