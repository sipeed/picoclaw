"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const DataTransferItem_js_1 = __importDefault(require("./DataTransferItem.cjs"));
const DataTransferItemList_js_1 = __importDefault(require("./DataTransferItemList.cjs"));
/**
 * DataTransfer.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/DataTransfer
 */
class DataTransfer {
    dropEffect = 'none';
    effectAllowed = 'none';
    items = new DataTransferItemList_js_1.default();
    /**
     * Returns files.
     *
     * @returns Files.
     */
    get files() {
        const files = [];
        for (const item of this.items) {
            if (item.kind === 'file') {
                files.push(item.getAsFile());
            }
        }
        return files;
    }
    /**
     * Returns types.
     *
     * @returns Types.
     */
    get types() {
        return this.items.map((item) => item.type);
    }
    /**
     * Clears the data.
     */
    clearData() {
        this.items.clear();
    }
    /**
     * Sets the data.
     *
     * @param format Format.
     * @param data Data.
     */
    setData(format, data) {
        for (let i = 0, max = this.items.length; i < max; i++) {
            if (this.items[i].type === format) {
                this.items[i] = new DataTransferItem_js_1.default(data, format);
                return;
            }
        }
        this.items.add(data, format);
    }
    /**
     * Gets the data.
     *
     * @param format Format.
     * @returns Data.
     */
    getData(format) {
        for (let i = 0, max = this.items.length; i < max; i++) {
            if (this.items[i].type === format) {
                let data = '';
                this.items[i].getAsString((s) => (data = s));
                return data;
            }
        }
        return '';
    }
    /**
     * Sets drag image.
     *
     * TODO: Implement.
     */
    setDragImage() {
        throw new Error('Not implemented.');
    }
}
exports.default = DataTransfer;
//# sourceMappingURL=DataTransfer.cjs.map