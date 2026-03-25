import File from '../file/File.js';
import DataTransferItem from './DataTransferItem.js';

/**
 *
 */
export default class DataTransferItemList extends Array<DataTransferItem> {
	/**
	 * Adds an item.
	 *
	 * @param item Item.
	 * @param type Type.
	 */
	public add(item: File | string, type?: string): void {
		if (item instanceof File) {
			this.push(new DataTransferItem(item));
			return;
		}
		if (!type) {
			throw new TypeError(
				`Failed to execute 'add' on 'DataTransferItemList': parameter 1 is not of type 'File'.`
			);
		}
		this.push(new DataTransferItem(item, type));
	}

	/**
	 * Removes an item.
	 *
	 * @param index Index.
	 */
	public remove(index: number): void {
		this.splice(index, 1);
	}

	/**
	 * Clears list.
	 */
	public clear(): void {
		while (this.length) {
			this.pop();
		}
	}
}
