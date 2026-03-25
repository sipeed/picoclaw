import Plugin from './Plugin.cjs';
/**
 * PluginArray.
 */
export default class PluginArray {
    [n: number]: Plugin;
    readonly length: number;
    /**
     * Constructor.
     *
     * @param plugins Plugins.
     */
    constructor(plugins: Plugin[]);
    /**
     * Returns an item.
     *
     * @param index Index.
     * @returns Plugin.
     */
    item(index: number): Plugin;
    /**
     * Returns an item.
     *
     * @param name Name.
     * @returns Plugin.
     */
    namedItem(name: string): Plugin;
    /**
     * Refreshes the list.
     */
    refresh(): void;
    /**
     * Returns the object as a string.
     *
     * @returns String.
     */
    toString(): string;
}
//# sourceMappingURL=PluginArray.d.ts.map