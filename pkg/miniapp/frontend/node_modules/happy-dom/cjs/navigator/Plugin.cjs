"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
/**
 * Plugin.
 */
class Plugin {
    length = 0;
    description;
    filename;
    name;
    /**
     * Constructor.
     *
     * @param mimeTypes Mime types.
     * @param description Description.
     * @param filename Filename.
     * @param name Name.
     */
    constructor(mimeTypes, description, filename, name) {
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
    item(index) {
        return this[index] || null;
    }
    /**
     * NamedItem.
     *
     * @param name String.
     * @returns IMimeType.
     */
    namedItem(name) {
        return this[name] || null;
    }
    /**
     * Returns the object as a string.
     *
     * @returns String.
     */
    toString() {
        return '[object Plugin]';
    }
}
exports.default = Plugin;
//# sourceMappingURL=Plugin.cjs.map