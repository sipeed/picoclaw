import MimeType from './MimeType.js';

/**
 * Plugin.
 */
export default class Plugin {
	[n: number]: MimeType;
	public readonly length: number = 0;
	public readonly description: string;
	public readonly filename: string;
	public readonly name: string;

	/**
	 * Constructor.
	 *
	 * @param mimeTypes Mime types.
	 * @param description Description.
	 * @param filename Filename.
	 * @param name Name.
	 */
	constructor(mimeTypes: MimeType[], description: string, filename: string, name: string) {
		this.description = description;
		this.filename = filename;
		this.name = name;

		for (let i = 0, max = mimeTypes.length; i < max; i++) {
			this[i] = mimeTypes[i];
			this[mimeTypes[i].type] = mimeTypes[i];
		}
		this.length = mimeTypes.length;
	}

	/**
	 * Item.
	 *
	 * @param index Number.
	 * @returns IMimeType.
	 */
	public item(index: number): MimeType {
		return this[index] || null;
	}

	/**
	 * NamedItem.
	 *
	 * @param name String.
	 * @returns IMimeType.
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
		return '[object Plugin]';
	}
}
