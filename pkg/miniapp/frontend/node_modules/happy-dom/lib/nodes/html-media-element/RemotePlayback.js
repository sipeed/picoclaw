import EventTarget from '../../event/EventTarget.js';
import * as PropertySymbol from '../../PropertySymbol.js';
/**
 * RemotePlayback.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/RemotePlayback
 */
export default class RemotePlayback extends EventTarget {
    // Internal properties
    [PropertySymbol.state] = 'disconnected';
    // Events
    onconnecting = null;
    onconnect = null;
    ondisconnect = null;
    /**
     * Returns the state of the remote playback.
     */
    get state() {
        return this[PropertySymbol.state];
    }
    /**
     * Watches the list of available remote playback devices and returns a Promise that resolves with a callbackId of an available remote playback device.
     *
     * @returns Promise.
     */
    async watchAvailability() {
        // TODO: Implement
    }
    /**
     * Cancels the request to monitor the availability of remote playback devices.
     */
    cancelWatchAvailability() {
        // TODO: Implement
    }
    /**
     * Prompts the user to select and give permission to connect to a remote playback device.
     */
    prompt() {
        // TODO: Implement
    }
}
//# sourceMappingURL=RemotePlayback.js.map