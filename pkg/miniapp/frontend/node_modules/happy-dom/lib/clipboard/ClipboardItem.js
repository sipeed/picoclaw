import DOMException from '../exception/DOMException.js';
import Blob from '../file/Blob.js';
/**
 * Clipboard Item API.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/ClipboardItem.
 */
export default class ClipboardItem {
    presentationStyle = 'unspecified';
    #data;
    /**
     * Constructor.
     *
     * @param data Data.
     * @param [options] Options.
     * @param [options.presentationStyle] Presentation style.
     */
    constructor(data, options) {
        this.#data = data;
        if (options?.presentationStyle) {
            this.presentationStyle = options.presentationStyle;
        }
    }
    /**
     * Returns types.
     *
     * @returns Types.
     */
    get types() {
        return Object.keys(this.#data);
    }
    /**
     * Returns data by type.
     *
     * @param type Type.
     * @returns Data.
     */
    async getType(type) {
        if (!this.#data[type]) {
            throw new DOMException(`Failed to execute 'getType' on 'ClipboardItem': The type '${type}' was not found`);
        }
        if (this.#data[type] instanceof Blob) {
            return this.#data[type];
        }
        return new Blob([await this.#data[type]], { type });
    }
}
//# sourceMappingURL=ClipboardItem.js.map