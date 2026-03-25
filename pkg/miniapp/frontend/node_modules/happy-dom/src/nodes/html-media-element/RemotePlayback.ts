import EventTarget from '../../event/EventTarget.js';
import * as PropertySymbol from '../../PropertySymbol.js';

/**
 * RemotePlayback.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/RemotePlayback
 */
export default class RemotePlayback extends EventTarget {
	// Internal properties
	public [PropertySymbol.state]: 'connecting' | 'connected' | 'disconnected' = 'disconnected';

	// Events
	public onconnecting: (event: Event) => void = null;
	public onconnect: (event: Event) => void = null;
	public ondisconnect: (event: Event) => void = null;

	/**
	 * Returns the state of the remote playback.
	 */
	public get state(): 'connecting' | 'connected' | 'disconnected' {
		return this[PropertySymbol.state];
	}

	/**
	 * Watches the list of available remote playback devices and returns a Promise that resolves with a callbackId of an available remote playback device.
	 *
	 * @returns Promise.
	 */
	public async watchAvailability(): Promise<void> {
		// TODO: Implement
	}

	/**
	 * Cancels the request to monitor the availability of remote playback devices.
	 */
	public cancelWatchAvailability(): void {
		// TODO: Implement
	}

	/**
	 * Prompts the user to select and give permission to connect to a remote playback device.
	 */
	public prompt(): void {
		// TODO: Implement
	}
}
