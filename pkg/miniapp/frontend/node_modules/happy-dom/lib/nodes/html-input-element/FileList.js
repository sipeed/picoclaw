/**
 * FileList.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/FileList
 */
export default class FileList extends Array {
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
    get [Symbol.toStringTag]() {
        return this.constructor.name;
    }
    /**
     * Returns item by index.
     *
     * @param index Index.
     * @returns Item.
     */
    item(index) {
        return this[index] || null;
    }
}
//# sourceMappingURL=FileList.js.map