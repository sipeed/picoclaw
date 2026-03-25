/**
 * MimeTypeArray.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/MimeTypeArray.
 */
export default class MimeTypeArray {
    length;
    /**
     * Constructor.
     *
     * @param mimeTypes
     */
    constructor(mimeTypes) {
        for (let i = 0, max = mimeTypes.length; i < max; i++) {
            this[i] = mimeTypes[i];
            this[mimeTypes[i].type] = mimeTypes[i];
        }
        this.length = mimeTypes.length;
    }
    /**
     * @param index
     */
    item(index) {
        return this[index] || null;
    }
    /**
     * @param name
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
        return '[object MimeTypeArray]';
    }
}
//# sourceMappingURL=MimeTypeArray.js.map