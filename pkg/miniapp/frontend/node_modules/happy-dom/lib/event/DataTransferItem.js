/**
 * Data transfer item.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/DataTransferItem.
 */
export default class DataTransferItem {
    kind;
    type;
    #item = null;
    /**
     * Constructor.
     *
     * @param item Item.
     * @param type Type.
     */
    constructor(item, type = '') {
        this.kind = typeof item === 'string' ? 'string' : 'file';
        this.type = this.kind === 'string' ? type : item.type;
        this.#item = item;
    }
    /**
     * Returns file.
     */
    getAsFile() {
        if (this.kind === 'string') {
            return null;
        }
        return this.#item;
    }
    /**
     * Returns string.
     *
     * @param callback Callback.
     */
    getAsString(callback) {
        if (this.kind === 'file') {
            callback;
        }
        callback(this.#item);
    }
}
//# sourceMappingURL=DataTransferItem.js.map