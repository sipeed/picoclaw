import Plugin from './Plugin.js';

/**
 * MimeType.
 */
export default class MimeType {
	public readonly description: string;
	public readonly enabledPlugin: Plugin;
	public readonly suffixes: string;
	public readonly type: string;

	/**
	 * Constructor.
	 *
	 * @param description
	 * @param enabledPlugin
	 * @param suffixes
	 * @param type
	 */
	constructor(description: string, enabledPlugin: Plugin, suffixes: string, type: string) {
		this.description = description;
		this.enabledPlugin = enabledPlugin;
		this.suffixes = suffixes;
		this.type = type;
	}

	/**
	 * Returns the object as a string.
	 *
	 * @returns String.
	 */
	public toString(): string {
		return '[object MimeType]';
	}
}
