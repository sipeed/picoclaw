/**
 * MimeType.
 */
export default class MimeType {
    description;
    enabledPlugin;
    suffixes;
    type;
    /**
     * Constructor.
     *
     * @param description
     * @param enabledPlugin
     * @param suffixes
     * @param type
     */
    constructor(description, enabledPlugin, suffixes, type) {
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
    toString() {
        return '[object MimeType]';
    }
}
//# sourceMappingURL=MimeType.js.map