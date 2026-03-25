import BrowserWindow from '../window/BrowserWindow.js';
import ClipboardItem from './ClipboardItem.js';
import Blob from '../file/Blob.js';

/**
 * Clipboard API.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/Clipboard.
 */
export default class Clipboard {
	#window: BrowserWindow;
	#data: ClipboardItem[] = [];

	/**
	 * Constructor.
	 *
	 * @param window Owner window.
	 */
	constructor(window: BrowserWindow) {
		if (!window) {
			throw new TypeError('Illegal constructor');
		}
		this.#window = window;
	}

	/**
	 * Returns data.
	 *
	 * @returns Data.
	 */
	public async read(): Promise<ClipboardItem[]> {
		const permissionStatus = await this.#window.navigator.permissions.query({
			name: 'clipboard-read'
		});
		if (permissionStatus.state === 'denied') {
			throw new this.#window.DOMException(
				`Failed to execute 'read' on 'Clipboard': The request is not allowed`
			);
		}
		return this.#data;
	}

	/**
	 * Returns text.
	 *
	 * @returns Text.
	 */
	public async readText(): Promise<string> {
		const permissionStatus = await this.#window.navigator.permissions.query({
			name: 'clipboard-read'
		});
		if (permissionStatus.state === 'denied') {
			throw new this.#window.DOMException(
				`Failed to execute 'readText' on 'Clipboard': The request is not allowed`
			);
		}
		let text = '';
		for (const item of this.#data) {
			if (item.types.includes('text/plain')) {
				const data = await item.getType('text/plain');
				if (typeof data === 'string') {
					text += data;
				} else {
					// Instance of Blob
					text += await data.text();
				}
			}
		}
		return text;
	}

	/**
	 * Writes data.
	 *
	 * @param data Data.
	 */
	public async write(data: ClipboardItem[]): Promise<void> {
		const permissionStatus = await this.#window.navigator.permissions.query({
			name: 'clipboard-write'
		});
		if (permissionStatus.state === 'denied') {
			throw new this.#window.DOMException(
				`Failed to execute 'write' on 'Clipboard': The request is not allowed`
			);
		}
		this.#data = data;
	}

	/**
	 * Writes text.
	 *
	 * @param text Text.
	 */
	public async writeText(text: string): Promise<void> {
		const permissionStatus = await this.#window.navigator.permissions.query({
			name: 'clipboard-write'
		});
		if (permissionStatus.state === 'denied') {
			throw new this.#window.DOMException(
				`Failed to execute 'writeText' on 'Clipboard': The request is not allowed`
			);
		}
		this.#data = [
			new this.#window.ClipboardItem({ 'text/plain': new Blob([text], { type: 'text/plain' }) })
		];
	}
}
