import File from '../file/File.js';
import DataTransferItem from './DataTransferItem.js';
/**
 *
 */
export default class DataTransferItemList extends Array {
    /**
     * Adds an item.
     *
     * @param item Item.
     * @param type Type.
     */
    add(item, type) {
        if (item instanceof File) {
            this.push(new DataTransferItem(item));
            return;
        }
        if (!type) {
            throw new TypeError(`Failed to execute 'add' on 'DataTransferItemList': parameter 1 is not of type 'File'.`);
        }
        this.push(new DataTransferItem(item, type));
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
//# sourceMappingURL=DataTransferItemList.js.map