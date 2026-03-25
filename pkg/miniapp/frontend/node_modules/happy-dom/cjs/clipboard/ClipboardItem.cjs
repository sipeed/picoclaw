"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const DOMException_js_1 = __importDefault(require("../exception/DOMException.cjs"));
const Blob_js_1 = __importDefault(require("../file/Blob.cjs"));
/**
 * Clipboard Item API.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/ClipboardItem.
 */
class ClipboardItem {
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
            throw new DOMException_js_1.default(`Failed to execute 'getType' on 'ClipboardItem': The type '${type}' was not found`);
        }
        if (this.#data[type] instanceof Blob_js_1.default) {
            return this.#data[type];
        }
        return new Blob_js_1.default([await this.#data[type]], { type });
    }
}
exports.default = ClipboardItem;
//# sourceMappingURL=ClipboardItem.cjs.map