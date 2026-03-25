import DataTransferItem from './DataTransferItem.js';
import DataTransferItemList from './DataTransferItemList.js';
/**
 * DataTransfer.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/DataTransfer
 */
export default class DataTransfer {
    dropEffect = 'none';
    effectAllowed = 'none';
    items = new DataTransferItemList();
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
                this.items[i] = new DataTransferItem(data, format);
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
//# sourceMappingURL=DataTransfer.js.map