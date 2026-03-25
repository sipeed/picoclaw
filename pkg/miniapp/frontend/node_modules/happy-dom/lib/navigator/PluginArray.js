/**
 * PluginArray.
 */
export default class PluginArray {
    length;
    /**
     * Constructor.
     *
     * @param plugins Plugins.
     */
    constructor(plugins) {
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
    item(index) {
        return this[index] || null;
    }
    /**
     * Returns an item.
     *
     * @param name Name.
     * @returns Plugin.
     */
    namedItem(name) {
        return this[name] || null;
    }
    /**
     * Refreshes the list.
     */
    refresh() {
        // Do nothing
    }
    /**
     * Returns the object as a string.
     *
     * @returns String.
     */
    toString() {
        return '[object PluginArray]';
    }
}
//# sourceMappingURL=PluginArray.js.map