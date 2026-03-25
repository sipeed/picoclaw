import DOMException from '../exception/DOMException.js';
import Blob from '../file/Blob.js';

/**
 * Clipboard Item API.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/ClipboardItem.
 */
export default class ClipboardItem {
	public readonly presentationStyle: 'unspecified' | 'inline' | 'attachment' = 'unspecified';
	#data: { [mimeType: string]: Blob | string | Promise<Blob | string> };

	/**
	 * Constructor.
	 *
	 * @param data Data.
	 * @param [options] Options.
	 * @param [options.presentationStyle] Presentation style.
	 */
	constructor(
		data: { [mimeType: string]: Blob | string | Promise<Blob | string> },
		options?: { presentationStyle?: 'unspecified' | 'inline' | 'attachment' }
	) {
		this.#data = data;
		if (options?.presentationStyle) {
			this.presentationStyle = options.presentationStyle;
		}
	}

	/**
	 * Returns types.
	 *
	 * @returns Types.
	 */
	public get types(): string[] {
		return Object.keys(this.#data);
	}

	/**
	 * Returns data by type.
	 *
	 * @param type Type.
	 * @returns Data.
	 */
	public async getType(type: string): Promise<Blob> {
		if (!this.#data[type]) {
			throw new DOMException(
				`Failed to execute 'getType' on 'ClipboardItem': The type '${type}' was not found`
			);
		}
		if (this.#data[type] instanceof Blob) {
			return this.#data[type];
		}
		return new Blob([await this.#data[type]], { type });
	}
}
