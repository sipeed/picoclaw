"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const File_js_1 = __importDefault(require("../file/File.cjs"));
const DataTransferItem_js_1 = __importDefault(require("./DataTransferItem.cjs"));
/**
 *
 */
class DataTransferItemList extends Array {
    /**
     * Adds an item.
     *
     * @param item Item.
     * @param type Type.
     */
    add(item, type) {
        if (item instanceof File_js_1.default) {
            this.push(new DataTransferItem_js_1.default(item));
            return;
        }
        if (!type) {
            throw new TypeError(`Failed to execute 'add' on 'DataTransferItemList': parameter 1 is not of type 'File'.`);
        }
        this.push(new DataTransferItem_js_1.default(item, type));
    }
    /**
     * Removes an item.
     *
     * @param index Index.
     */
    remove(index) {
        this.splice(index, 1);
    }
    /**
     * Clears list.
     */
    clear() {
        while (this.length) {
            this.pop();
        }
    }
}
exports.default = DataTransferItemList;
//# sourceMappingURL=DataTransferItemList.cjs.map