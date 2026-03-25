import Plugin from './Plugin.js';

/**
 * PluginArray.
 */
export default class PluginArray {
	[n: number]: Plugin;
	public readonly length: number;

	/**
	 * Constructor.
	 *
	 * @param plugins Plugins.
	 */
	constructor(plugins: Plugin[]) {
		for (let i = 0, max = plugins.length; i < max; i++) {
			this[i] = plugins[i];
			this[plugins[i].name] = plugins[i];
		}
		this.length = plugins.length;
	}

	/**
	 * Returns an item.
	 *
	 * @param index Index.
	 * @returns Plugin.
	 */
	public item(index: number): Plugin {
		return this[index] || null;
	}

	/**
	 * Returns an item.
	 *
	 * @param name Name.
	 * @returns Plugin.
	 */
	public namedItem(name: string): Plugin {
		return this[name] || null;
	}

	/**
	 * Refreshes the list.
	 */
	public refresh(): void {
		// Do nothing
	}

	/**
	 * Returns the object as a string.
	 *
	 * @returns String.
	 */
	public toString(): string {
		return '[object PluginArray]';
	}
}
