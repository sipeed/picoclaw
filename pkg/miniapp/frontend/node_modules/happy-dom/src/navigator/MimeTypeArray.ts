import MimeType from './MimeType.js';

/**
 * MimeTypeArray.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/MimeTypeArray.
 */
export default class MimeTypeArray {
	[n: number]: MimeType;
	public readonly length: number;

	/**
	 * Constructor.
	 *
	 * @param mimeTypes
	 */
	constructor(mimeTypes: MimeType[]) {
		for (let i = 0, max = mimeTypes.length; i < max; i++) {
			this[i] = mimeTypes[i];
			this[mimeTypes[i].type] = mimeTypes[i];
		}
		this.length = mimeTypes.length;
	}

	/**
	 * @param index
	 */
	public item(index: number): MimeType {
		return this[index] || null;
	}

	/**
	 * @param name
	 */
	public namedItem(name: string): MimeType {
		return this[name] || null;
	}

	/**
	 * Returns the object as a string.
	 *
	 * @returns String.
	 */
	public toString(): string {
		return '[object MimeTypeArray]';
	}
}
