import File from '../file/File.js';

/**
 * Data transfer item.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/DataTransferItem.
 */
export default class DataTransferItem {
	public readonly kind: 'string' | 'file';
	public readonly type: string;
	#item: string | File = null;

	/**
	 * Constructor.
	 *
	 * @param item Item.
	 * @param type Type.
	 */
	constructor(item: string | File, type = '') {
		this.kind = typeof item === 'string' ? 'string' : 'file';
		this.type = this.kind === 'string' ? type : (<File>item).type;
		this.#item = item;
	}

	/**
	 * Returns file.
	 */
	public getAsFile(): File {
		if (this.kind === 'string') {
			return null;
		}
		return <File>this.#item;
	}

	/**
	 * Returns string.
	 *
	 * @param callback Callback.
	 */
	public getAsString(callback: (text: string) => void): void {
		if (this.kind === 'file') {
			callback;
		}
		callback(<string>this.#item);
	}
}
