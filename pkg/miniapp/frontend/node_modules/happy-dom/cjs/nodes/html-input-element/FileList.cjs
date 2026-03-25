"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
/**
 * FileList.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/FileList
 */
class FileList extends Array {
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
exports.default = FileList;
//# sourceMappingURL=FileList.cjs.map