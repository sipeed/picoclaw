import File from '../file/File.js';
import DataTransferItem from './DataTransferItem.js';
import DataTransferItemList from './DataTransferItemList.js';

/**
 * DataTransfer.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/DataTransfer
 */
export default class DataTransfer {
	public dropEffect: 'none' | 'copy' | 'link' | 'move' = 'none';
	public effectAllowed:
		| 'none'
		| 'copy'
		| 'copyLink'
		| 'copyMove'
		| 'link'
		| 'linkMove'
		| 'move'
		| 'all'
		| 'uninitialized' = 'none';
	public readonly items: DataTransferItemList = new DataTransferItemList();

	/**
	 * Returns files.
	 *
	 * @returns Files.
	 */
	public get files(): File[] {
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
	public get types(): string[] {
		return this.items.map((item) => item.type);
	}

	/**
	 * Clears the data.
	 */
	public clearData(): void {
		this.items.clear();
	}

	/**
	 * Sets the data.
	 *
	 * @param format Format.
	 * @param data Data.
	 */
	public setData(format: string, data: string): void {
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
	public getData(format: string): string {
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
	public setDragImage(): void {
		throw new Error('Not implemented.');
	}
}
