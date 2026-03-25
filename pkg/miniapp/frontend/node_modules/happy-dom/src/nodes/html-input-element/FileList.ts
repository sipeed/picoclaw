import File from '../../file/File.js';

/**
 * FileList.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/FileList
 */
export default class FileList extends Array<File> {
	/**
	 * Constructor.
	 */
	constructor() {
		super(0);
	}

	/**
	 * Returns `Symbol.toStringTag`.
	 *
	 * @returns `Symbol.toStringTag`.
	 */
	public get [Symbol.toStringTag](): string {
		return this.constructor.name;
	}

	/**
	 * Returns item by index.
	 *
	 * @param index Index.
	 * @returns Item.
	 */
	public item(index: number): File | null {
		return this[index] || null;
	}
}
